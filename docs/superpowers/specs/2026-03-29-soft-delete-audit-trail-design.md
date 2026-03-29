# Soft Delete & Audit Trail Design

**Date:** 2026-03-29
**Status:** Approved

## Overview

XAN-Argus is a business application that must be compliance and audit-ready. All data mutations must be traceable, and no records may be physically deleted. This design introduces soft delete (via `deleted_at` column) and an append-only audit log (via PostgreSQL triggers) across all 9 tables.

## Goals

- Replace all hard deletes with soft deletes
- Track every data mutation (INSERT, UPDATE, DELETE) in a central audit log
- Keep the Go layer as thin as possible — PostgreSQL triggers do the heavy lifting
- Zero UI changes, zero handler changes, zero new model fields
- Maintain existing API contract (DELETE endpoints still return 204)

## Non-Goals

- Actor tracking (deferred until authentication is implemented)
- UI for viewing deleted records or audit history
- Compliance extras (immutability guarantees, retention policies, signatures)
- Undelete/restore functionality

## Design

### 1. Soft Delete: `deleted_at` Column

Every table gets a new nullable column:

```sql
ALTER TABLE <table> ADD COLUMN deleted_at TIMESTAMPTZ;
```

**Tables affected (all 9):**
- `customers`
- `users`
- `services`
- `user_assignments`
- `assets`
- `licenses`
- `customer_services`
- `hardware_categories`
- `category_field_definitions`

**Semantics:**
- `NULL` = active record
- Timestamp = soft-deleted record
- All `List` and `GetByID` queries filter with `WHERE deleted_at IS NULL`
- Delete operations become `UPDATE ... SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`

### 2. Partial Unique Indexes

Existing UNIQUE constraints must be replaced with partial unique indexes so that soft-deleted records don't block new records with the same name.

| Table | Current Constraint | Replacement |
|---|---|---|
| `customers` | `UNIQUE(name)` | `CREATE UNIQUE INDEX idx_customers_name_active ON customers(name) WHERE deleted_at IS NULL` |
| `hardware_categories` | `UNIQUE(name)` | `CREATE UNIQUE INDEX idx_hardware_categories_name_active ON hardware_categories(name) WHERE deleted_at IS NULL` |
| `user_assignments` | `UNIQUE(user_id, customer_id)` | `CREATE UNIQUE INDEX idx_user_assignments_active ON user_assignments(user_id, customer_id) WHERE deleted_at IS NULL` |
| `customer_services` | `UNIQUE(customer_id, service_id)` | `CREATE UNIQUE INDEX idx_customer_services_active ON customer_services(customer_id, service_id) WHERE deleted_at IS NULL` |
| `category_field_definitions` | `UNIQUE(category_id, name)` | `CREATE UNIQUE INDEX idx_field_definitions_name_active ON category_field_definitions(category_id, name) WHERE deleted_at IS NULL` |

### 3. Soft Delete Guard Triggers

When a parent record is soft-deleted, PostgreSQL must verify that no active dependent records exist. This replaces the `ON DELETE RESTRICT` behavior (which only fires on physical `DELETE`).

Each guard trigger uses `ERRCODE = '23503'` (foreign key violation) so the existing `isFKViolation()` handler code works without changes.

**Dependency map (parent → dependents):**

| Parent Table | Active Dependents to Check |
|---|---|
| `customers` | `user_assignments`, `assets`, `licenses`, `customer_services` |
| `users` | `user_assignments` |
| `services` | `customer_services` |
| `user_assignments` | `assets` (via `user_assignment_id`), `licenses` (via `user_assignment_id`) |
| `hardware_categories` | `category_field_definitions`, `assets` (via `category_id`) |

**Tables with no dependents (no guard trigger needed):**
- `assets`
- `licenses`
- `customer_services`
- `category_field_definitions`

**Trigger pattern:**

```sql
CREATE FUNCTION check_soft_delete_<table>() RETURNS TRIGGER AS $$
BEGIN
    IF OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL THEN
        IF EXISTS (SELECT 1 FROM <dependent> WHERE <fk> = OLD.id AND deleted_at IS NULL)
        -- OR EXISTS (...) for additional dependents
        THEN
            RAISE EXCEPTION 'cannot delete: active dependent records exist'
                USING ERRCODE = '23503';
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_<table>_soft_delete_guard
    BEFORE UPDATE ON <table>
    FOR EACH ROW EXECUTE FUNCTION check_soft_delete_<table>();
```

### 4. Audit Log Table

A single append-only table for all mutation tracking:

```sql
CREATE TABLE audit_log (
    id BIGSERIAL PRIMARY KEY,
    table_name TEXT NOT NULL,
    record_id UUID NOT NULL,
    action TEXT NOT NULL CHECK (action IN ('INSERT', 'UPDATE', 'DELETE')),
    old_values JSONB,
    new_values JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_log_record ON audit_log(table_name, record_id);
CREATE INDEX idx_audit_log_created ON audit_log(created_at);
```

**Design decisions:**
- `BIGSERIAL` primary key — monotonically increasing, more efficient than UUID for append-only data
- No `updated_at` — audit entries are immutable
- No `actor_id` — deferred until authentication is implemented
- `old_values` / `new_values` store complete row state as JSONB via `row_to_json()`

### 5. Audit Trigger

A single generic trigger function attached to all 9 tables as `AFTER INSERT OR UPDATE`:

```sql
CREATE FUNCTION audit_trigger() RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        INSERT INTO audit_log(table_name, record_id, action, new_values)
        VALUES (TG_TABLE_NAME, NEW.id, 'INSERT', row_to_json(NEW)::jsonb);
    ELSIF TG_OP = 'UPDATE' THEN
        IF NEW.deleted_at IS NOT NULL AND OLD.deleted_at IS NULL THEN
            INSERT INTO audit_log(table_name, record_id, action, old_values)
            VALUES (TG_TABLE_NAME, OLD.id, 'DELETE', row_to_json(OLD)::jsonb);
        ELSE
            INSERT INTO audit_log(table_name, record_id, action, old_values, new_values)
            VALUES (TG_TABLE_NAME, OLD.id, 'UPDATE', row_to_json(OLD)::jsonb, row_to_json(NEW)::jsonb);
        END IF;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
```

**Action mapping:**
- Physical `INSERT` → audit action `INSERT`
- Physical `UPDATE` where `deleted_at` transitions `NULL → non-NULL` → audit action `DELETE`
- Physical `UPDATE` otherwise → audit action `UPDATE`

### 6. Repository Changes

All 9 repository files require changes:

**Delete method:** `DELETE FROM <table> WHERE id = $1` becomes `UPDATE <table> SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, keeping `RowsAffected()` check for 404 detection. Method signature stays `Delete(ctx, id) error`.

**All read queries:** Add `WHERE deleted_at IS NULL` (or `AND deleted_at IS NULL` if WHERE clause already exists) to: `GetByID`, `List`, `ListByCustomer`, `ListByCategory`, `ListFields`, and any other read method.

### 7. Handler Changes

**None.** DELETE endpoints continue to return 204 No Content. The `isFKViolation()` check catches the `23503` error from soft-delete guard triggers. No new routes, no new error handling.

### 8. Model Changes

**None.** `deleted_at` is not added to Go structs — it is never exposed via API or UI. The database manages it entirely.

### 9. UI Changes

**None.** Deleted records are invisible because repository queries filter them out. Delete buttons continue to work via the existing DELETE API endpoints.

### 10. FK Constraint Adjustments

The existing `ON DELETE RESTRICT` and `ON DELETE CASCADE` constraints remain in place as a safety net against accidental physical deletes. However, since all deletes now go through soft delete (UPDATE), these constraints will not fire in normal operation.

Notable exceptions:

- `category_field_definitions` has `ON DELETE CASCADE` from `hardware_categories`. Since we're soft-deleting categories instead of physically deleting them, this cascade will no longer fire. The soft-delete guard trigger on `hardware_categories` handles this case instead.
- `assets.category_id` has `ON DELETE SET NULL`. Previously, physically deleting a category would silently nullify the FK on assets. With soft delete, the guard trigger on `hardware_categories` now **blocks** deletion if active assets reference the category. This is an intentional behavioral change — it is stricter but more correct for a compliance-oriented system.

### 11. isFKViolation() Compatibility

The existing `isFKViolation()` helper checks PostgreSQL error code `23503` regardless of statement type. Since soft-delete guard triggers raise `23503` on an `UPDATE` statement (not `DELETE`), pgx surfaces the error identically. No changes to error detection logic are needed.

## Migration Strategy

All changes are delivered in a single migration file (`008_soft_delete_audit.sql`).

**Important:** All PL/pgSQL function bodies must be wrapped in `-- +goose StatementBegin` / `-- +goose StatementEnd` blocks (goose requirement for multi-statement SQL).

### Up Migration

1. Create `audit_log` table with indexes
2. Create generic `audit_trigger()` function (wrapped in StatementBegin/End)
3. Add `deleted_at` column to all 9 tables
4. Drop existing UNIQUE constraints, create partial unique indexes
5. Create soft-delete guard trigger functions for parent tables (each wrapped in StatementBegin/End)
6. Create soft-delete guard triggers on parent tables
7. Attach audit trigger to all 9 tables

### Down Migration (reverse order)

1. Drop audit triggers from all 9 tables
2. Drop soft-delete guard triggers and functions
3. Drop partial unique indexes, restore original UNIQUE constraints
4. Remove `deleted_at` column from all 9 tables (note: this discards soft-delete state)
5. Drop `audit_trigger()` function
6. Drop `audit_log` table

## Known Tradeoffs

- **Audit log size:** Every UPDATE generates a full JSONB snapshot of both old and new row states. For tables with large JSONB columns (e.g., `assets.field_values`), this can grow quickly. Acceptable for now; a future optimization could skip audit entries where `OLD IS NOT DISTINCT FROM NEW` or store only changed fields.
- **No `deleted_at` index:** Queries filter by `WHERE deleted_at IS NULL` on every read. For the current data volume this is fine. A partial index (`WHERE deleted_at IS NULL`) on high-traffic tables can be added later if needed.

## Testing Considerations

- Soft-deleted records must not appear in list/get queries
- Soft-deleting a parent with active children must fail with FK violation error
- Soft-deleting a parent with only soft-deleted children must succeed
- Audit log entries must be created for INSERT, UPDATE, and soft DELETE
- Partial unique indexes must allow duplicate names when one record is soft-deleted
- `RowsAffected() == 0` must still work for already-deleted or non-existent records
