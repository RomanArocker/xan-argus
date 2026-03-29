# Database & Migration Conventions

PostgreSQL 18.3 via pgx. Migrations managed by goose (pure SQL, run on startup).

## Migration Files

Location: `db/migrations/`. Naming: `NNN_description.sql` (sequential numbering).

```sql
-- db/migrations/001_base_schema.sql

-- +goose Up

-- Table definition
CREATE TABLE customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    contact_email TEXT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Updated_at trigger (every table gets one)
CREATE TRIGGER trg_customers_updated_at
    BEFORE UPDATE ON customers
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE IF EXISTS customers;
```

### Migration Rules

- Always include both `-- +goose Up` and `-- +goose Down` sections
- Down section reverses in opposite order of Up
- PL/pgSQL functions (triggers) must be wrapped in `-- +goose StatementBegin` / `-- +goose StatementEnd`
- Migrations run automatically on application startup via `internal/database/migrate.go`

## Table Conventions

Every table follows this pattern:

```sql
CREATE TABLE entity (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- entity-specific columns --
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);
```

- **Primary key**: always `id UUID` with `gen_random_uuid()`
- **Timestamps**: every table has `created_at`, `updated_at`, and `deleted_at`
- **Soft delete**: `deleted_at` is nullable — `NULL` = active, timestamp = soft-deleted
- **Updated trigger**: every table gets a `trg_entity_updated_at` trigger using shared `set_updated_at()` function
- **Audit trigger**: every table gets a `trg_entity_audit` trigger using shared `audit_trigger()` function

## Foreign Keys

```sql
customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE RESTRICT
```

- **Always `ON DELETE RESTRICT`** — no cascading deletes, ever
- FK constraints remain as a safety net but won't fire during normal operation (soft deletes use `UPDATE`, not `DELETE`)
- **Soft-delete guard triggers** enforce referential integrity: parent tables have `BEFORE UPDATE` triggers that check for active dependents before allowing soft delete, raising `ERRCODE '23503'`
- FK violations are caught in handlers via `isFKViolation()` (PG code `23503`) — works for both hard FK violations and soft-delete guard triggers
- Unique violations via `isUniqueViolation()` (PG code `23505`)

## Constraints

Use PostgreSQL's constraint system directly:

```sql
-- CHECK constraints for enums
type TEXT NOT NULL CHECK (type IN ('customer_staff', 'internal_staff'))

-- UNIQUE constraints (use partial indexes for soft-delete compatibility)
-- name TEXT NOT NULL UNIQUE  ← don't use plain UNIQUE
CREATE UNIQUE INDEX idx_entity_name_active ON entity(name) WHERE deleted_at IS NULL;

-- Cross-table consistency via triggers
-- Example: user_assignment.customer_id must match license.customer_id
```

## JSONB Columns

```sql
metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
field_values JSONB NOT NULL DEFAULT '{}'::jsonb
```

- Default to empty object `'{}'::jsonb`
- Add GIN indexes for queryable JSONB: `CREATE INDEX idx_entity_metadata ON entity USING GIN (metadata)`
- field_values keys are field definition UUIDs, not human-readable names

## Seed Data

Seed data goes in its own migration file (e.g., `005_seed_data.sql`). Use `INSERT ... ON CONFLICT DO NOTHING` for idempotency.

## Audit Log

All data mutations are tracked automatically via PostgreSQL triggers in an append-only `audit_log` table:

```sql
audit_log (id BIGSERIAL, table_name, record_id UUID, action, old_values JSONB, new_values JSONB, created_at)
```

- Actions: `INSERT`, `UPDATE`, `DELETE` (soft delete logged as `DELETE`)
- No Go code needed — the `audit_trigger()` function fires `AFTER INSERT OR UPDATE` on every table
- Actor tracking deferred until authentication is implemented

## PostgreSQL-First Philosophy

Leverage the database for:
- Data validation (CHECK constraints, NOT NULL, UNIQUE)
- Referential integrity (foreign keys with RESTRICT, soft-delete guard triggers)
- Automatic timestamps (triggers)
- Audit trail (append-only audit_log via triggers)
- Soft delete enforcement (guard triggers prevent deleting parents with active children)
- Cross-record consistency (custom triggers)
- Full-text search (ILIKE, GIN indexes)
- Dynamic schemas (JSONB with GIN)

Keep the Go layer thin — if PostgreSQL can enforce it, let PostgreSQL enforce it.
