# Soft Delete & Audit Trail Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace all hard deletes with soft deletes and add an append-only audit log via PostgreSQL triggers across all 9 tables.

**Architecture:** Single migration adds `deleted_at` column, partial unique indexes, soft-delete guard triggers, and audit log table+triggers. Repository layer changes `DELETE FROM` to `UPDATE SET deleted_at` and adds `WHERE deleted_at IS NULL` to all reads. No handler, model, or UI changes needed.

**Tech Stack:** PostgreSQL 18.3, Go (pgx), goose migrations

**Spec:** `docs/superpowers/specs/2026-03-29-soft-delete-audit-trail-design.md`

---

### Task 1: Create migration file — audit log table and trigger function

**Files:**
- Create: `db/migrations/008_soft_delete_audit.sql`

This task creates the first part of the migration: the `audit_log` table and the generic `audit_trigger()` function.

- [ ] **Step 1: Create migration file with Up header, audit_log table, and audit trigger function**

```sql
-- +goose Up

-- 1. Audit log table
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

-- 2. Generic audit trigger function
-- +goose StatementBegin
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
-- +goose StatementEnd
```

- [ ] **Step 2: Commit**

```bash
git add db/migrations/008_soft_delete_audit.sql
git commit -m "feat: add audit_log table and audit trigger function (migration 008, part 1)"
```

---

### Task 2: Add deleted_at columns and partial unique indexes to migration

**Files:**
- Modify: `db/migrations/008_soft_delete_audit.sql`

Append to the Up section: `deleted_at` columns for all 9 tables, drop existing UNIQUE constraints, create partial unique indexes.

- [ ] **Step 1: Add deleted_at columns for all 9 tables**

Append to the migration file:

```sql
-- 3. Add deleted_at column to all tables
ALTER TABLE customers ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE services ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE user_assignments ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE assets ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE licenses ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE customer_services ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE hardware_categories ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE category_field_definitions ADD COLUMN deleted_at TIMESTAMPTZ;
```

- [ ] **Step 2: Replace UNIQUE constraints with partial unique indexes**

Append to the migration file:

```sql
-- 4. Replace UNIQUE constraints with partial unique indexes
-- customers: UNIQUE(name) → partial
ALTER TABLE customers DROP CONSTRAINT customers_name_key;
CREATE UNIQUE INDEX idx_customers_name_active ON customers(name) WHERE deleted_at IS NULL;

-- hardware_categories: UNIQUE(name) → partial
ALTER TABLE hardware_categories DROP CONSTRAINT hardware_categories_name_key;
CREATE UNIQUE INDEX idx_hardware_categories_name_active ON hardware_categories(name) WHERE deleted_at IS NULL;

-- user_assignments: UNIQUE(user_id, customer_id) → partial
ALTER TABLE user_assignments DROP CONSTRAINT user_assignments_user_id_customer_id_key;
CREATE UNIQUE INDEX idx_user_assignments_active ON user_assignments(user_id, customer_id) WHERE deleted_at IS NULL;

-- customer_services: UNIQUE(customer_id, service_id) → partial
ALTER TABLE customer_services DROP CONSTRAINT customer_services_customer_id_service_id_key;
CREATE UNIQUE INDEX idx_customer_services_active ON customer_services(customer_id, service_id) WHERE deleted_at IS NULL;

-- category_field_definitions: UNIQUE(category_id, name) → partial
ALTER TABLE category_field_definitions DROP CONSTRAINT category_field_definitions_category_id_name_key;
CREATE UNIQUE INDEX idx_field_definitions_name_active ON category_field_definitions(category_id, name) WHERE deleted_at IS NULL;
```

**Note:** The constraint names above follow PostgreSQL's auto-naming convention (`tablename_column_key`). If the actual names differ, check with: `SELECT conname FROM pg_constraint WHERE conrelid = 'tablename'::regclass AND contype = 'u';`

- [ ] **Step 3: Commit**

```bash
git add db/migrations/008_soft_delete_audit.sql
git commit -m "feat: add deleted_at columns and partial unique indexes (migration 008, part 2)"
```

---

### Task 3: Add soft-delete guard triggers to migration

**Files:**
- Modify: `db/migrations/008_soft_delete_audit.sql`

Append guard trigger functions and triggers for parent tables that have active dependents.

- [ ] **Step 1: Add guard trigger for customers**

```sql
-- 5. Soft-delete guard triggers

-- customers → user_assignments, assets, licenses, customer_services
-- +goose StatementBegin
CREATE FUNCTION check_soft_delete_customers() RETURNS TRIGGER AS $$
BEGIN
    IF OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL THEN
        IF EXISTS (SELECT 1 FROM user_assignments WHERE customer_id = OLD.id AND deleted_at IS NULL)
        OR EXISTS (SELECT 1 FROM assets WHERE customer_id = OLD.id AND deleted_at IS NULL)
        OR EXISTS (SELECT 1 FROM licenses WHERE customer_id = OLD.id AND deleted_at IS NULL)
        OR EXISTS (SELECT 1 FROM customer_services WHERE customer_id = OLD.id AND deleted_at IS NULL)
        THEN
            RAISE EXCEPTION 'cannot delete: active dependent records exist'
                USING ERRCODE = '23503';
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_customers_soft_delete_guard
    BEFORE UPDATE ON customers
    FOR EACH ROW EXECUTE FUNCTION check_soft_delete_customers();
```

- [ ] **Step 2: Add guard trigger for users**

```sql
-- users → user_assignments
-- +goose StatementBegin
CREATE FUNCTION check_soft_delete_users() RETURNS TRIGGER AS $$
BEGIN
    IF OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL THEN
        IF EXISTS (SELECT 1 FROM user_assignments WHERE user_id = OLD.id AND deleted_at IS NULL)
        THEN
            RAISE EXCEPTION 'cannot delete: active dependent records exist'
                USING ERRCODE = '23503';
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_users_soft_delete_guard
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION check_soft_delete_users();
```

- [ ] **Step 3: Add guard trigger for services**

```sql
-- services → customer_services
-- +goose StatementBegin
CREATE FUNCTION check_soft_delete_services() RETURNS TRIGGER AS $$
BEGIN
    IF OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL THEN
        IF EXISTS (SELECT 1 FROM customer_services WHERE service_id = OLD.id AND deleted_at IS NULL)
        THEN
            RAISE EXCEPTION 'cannot delete: active dependent records exist'
                USING ERRCODE = '23503';
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_services_soft_delete_guard
    BEFORE UPDATE ON services
    FOR EACH ROW EXECUTE FUNCTION check_soft_delete_services();
```

- [ ] **Step 4: Add guard trigger for user_assignments**

```sql
-- user_assignments → assets (via user_assignment_id), licenses (via user_assignment_id)
-- +goose StatementBegin
CREATE FUNCTION check_soft_delete_user_assignments() RETURNS TRIGGER AS $$
BEGIN
    IF OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL THEN
        IF EXISTS (SELECT 1 FROM assets WHERE user_assignment_id = OLD.id AND deleted_at IS NULL)
        OR EXISTS (SELECT 1 FROM licenses WHERE user_assignment_id = OLD.id AND deleted_at IS NULL)
        THEN
            RAISE EXCEPTION 'cannot delete: active dependent records exist'
                USING ERRCODE = '23503';
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_user_assignments_soft_delete_guard
    BEFORE UPDATE ON user_assignments
    FOR EACH ROW EXECUTE FUNCTION check_soft_delete_user_assignments();
```

- [ ] **Step 5: Add guard trigger for hardware_categories**

```sql
-- hardware_categories → category_field_definitions, assets (via category_id)
-- +goose StatementBegin
CREATE FUNCTION check_soft_delete_hardware_categories() RETURNS TRIGGER AS $$
BEGIN
    IF OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL THEN
        IF EXISTS (SELECT 1 FROM category_field_definitions WHERE category_id = OLD.id AND deleted_at IS NULL)
        OR EXISTS (SELECT 1 FROM assets WHERE category_id = OLD.id AND deleted_at IS NULL)
        THEN
            RAISE EXCEPTION 'cannot delete: active dependent records exist'
                USING ERRCODE = '23503';
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_hardware_categories_soft_delete_guard
    BEFORE UPDATE ON hardware_categories
    FOR EACH ROW EXECUTE FUNCTION check_soft_delete_hardware_categories();
```

- [ ] **Step 6: Commit**

```bash
git add db/migrations/008_soft_delete_audit.sql
git commit -m "feat: add soft-delete guard triggers (migration 008, part 3)"
```

---

### Task 4: Attach audit triggers and add Down migration

**Files:**
- Modify: `db/migrations/008_soft_delete_audit.sql`

- [ ] **Step 1: Attach audit trigger to all 9 tables**

Append to the Up section:

```sql
-- 6. Attach audit trigger to all tables
CREATE TRIGGER trg_customers_audit AFTER INSERT OR UPDATE ON customers FOR EACH ROW EXECUTE FUNCTION audit_trigger();
CREATE TRIGGER trg_users_audit AFTER INSERT OR UPDATE ON users FOR EACH ROW EXECUTE FUNCTION audit_trigger();
CREATE TRIGGER trg_services_audit AFTER INSERT OR UPDATE ON services FOR EACH ROW EXECUTE FUNCTION audit_trigger();
CREATE TRIGGER trg_user_assignments_audit AFTER INSERT OR UPDATE ON user_assignments FOR EACH ROW EXECUTE FUNCTION audit_trigger();
CREATE TRIGGER trg_assets_audit AFTER INSERT OR UPDATE ON assets FOR EACH ROW EXECUTE FUNCTION audit_trigger();
CREATE TRIGGER trg_licenses_audit AFTER INSERT OR UPDATE ON licenses FOR EACH ROW EXECUTE FUNCTION audit_trigger();
CREATE TRIGGER trg_customer_services_audit AFTER INSERT OR UPDATE ON customer_services FOR EACH ROW EXECUTE FUNCTION audit_trigger();
CREATE TRIGGER trg_hardware_categories_audit AFTER INSERT OR UPDATE ON hardware_categories FOR EACH ROW EXECUTE FUNCTION audit_trigger();
CREATE TRIGGER trg_category_field_definitions_audit AFTER INSERT OR UPDATE ON category_field_definitions FOR EACH ROW EXECUTE FUNCTION audit_trigger();
```

- [ ] **Step 2: Add Down migration**

Append the complete Down section:

```sql
-- +goose Down

-- Drop audit triggers
DROP TRIGGER IF EXISTS trg_customers_audit ON customers;
DROP TRIGGER IF EXISTS trg_users_audit ON users;
DROP TRIGGER IF EXISTS trg_services_audit ON services;
DROP TRIGGER IF EXISTS trg_user_assignments_audit ON user_assignments;
DROP TRIGGER IF EXISTS trg_assets_audit ON assets;
DROP TRIGGER IF EXISTS trg_licenses_audit ON licenses;
DROP TRIGGER IF EXISTS trg_customer_services_audit ON customer_services;
DROP TRIGGER IF EXISTS trg_hardware_categories_audit ON hardware_categories;
DROP TRIGGER IF EXISTS trg_category_field_definitions_audit ON category_field_definitions;

-- Drop soft-delete guard triggers and functions
DROP TRIGGER IF EXISTS trg_hardware_categories_soft_delete_guard ON hardware_categories;
DROP FUNCTION IF EXISTS check_soft_delete_hardware_categories();
DROP TRIGGER IF EXISTS trg_user_assignments_soft_delete_guard ON user_assignments;
DROP FUNCTION IF EXISTS check_soft_delete_user_assignments();
DROP TRIGGER IF EXISTS trg_services_soft_delete_guard ON services;
DROP FUNCTION IF EXISTS check_soft_delete_services();
DROP TRIGGER IF EXISTS trg_users_soft_delete_guard ON users;
DROP FUNCTION IF EXISTS check_soft_delete_users();
DROP TRIGGER IF EXISTS trg_customers_soft_delete_guard ON customers;
DROP FUNCTION IF EXISTS check_soft_delete_customers();

-- Drop partial unique indexes, restore original constraints
DROP INDEX IF EXISTS idx_field_definitions_name_active;
ALTER TABLE category_field_definitions ADD CONSTRAINT category_field_definitions_category_id_name_key UNIQUE(category_id, name);

DROP INDEX IF EXISTS idx_customer_services_active;
ALTER TABLE customer_services ADD CONSTRAINT customer_services_customer_id_service_id_key UNIQUE(customer_id, service_id);

DROP INDEX IF EXISTS idx_user_assignments_active;
ALTER TABLE user_assignments ADD CONSTRAINT user_assignments_user_id_customer_id_key UNIQUE(user_id, customer_id);

DROP INDEX IF EXISTS idx_hardware_categories_name_active;
ALTER TABLE hardware_categories ADD CONSTRAINT hardware_categories_name_key UNIQUE(name);

DROP INDEX IF EXISTS idx_customers_name_active;
ALTER TABLE customers ADD CONSTRAINT customers_name_key UNIQUE(name);

-- Drop deleted_at columns
ALTER TABLE category_field_definitions DROP COLUMN deleted_at;
ALTER TABLE hardware_categories DROP COLUMN deleted_at;
ALTER TABLE customer_services DROP COLUMN deleted_at;
ALTER TABLE licenses DROP COLUMN deleted_at;
ALTER TABLE assets DROP COLUMN deleted_at;
ALTER TABLE user_assignments DROP COLUMN deleted_at;
ALTER TABLE services DROP COLUMN deleted_at;
ALTER TABLE users DROP COLUMN deleted_at;
ALTER TABLE customers DROP COLUMN deleted_at;

-- Drop audit trigger function and table
DROP FUNCTION IF EXISTS audit_trigger();
DROP TABLE IF EXISTS audit_log;
```

- [ ] **Step 3: Commit**

```bash
git add db/migrations/008_soft_delete_audit.sql
git commit -m "feat: attach audit triggers and add down migration (migration 008, complete)"
```

---

### Task 5: Update customer repository — soft delete + filtered reads

**Files:**
- Modify: `internal/repository/customer.go`

This is the template for all other repository changes. Follow this exact pattern.

- [ ] **Step 1: Update GetByID — add `AND deleted_at IS NULL`**

In `internal/repository/customer.go`, change the GetByID query from:

```go
`SELECT id, name, contact_email, notes, created_at, updated_at
 FROM customers WHERE id = $1`, id,
```

to:

```go
`SELECT id, name, contact_email, notes, created_at, updated_at
 FROM customers WHERE id = $1 AND deleted_at IS NULL`, id,
```

- [ ] **Step 2: Update List — add `deleted_at IS NULL` filter**

Change the List method. The current pattern conditionally adds WHERE:

```go
query := `SELECT id, name, contact_email, notes, created_at, updated_at FROM customers`
args := []any{}
if params.Search != "" {
    query += ` WHERE name ILIKE $1`
    args = append(args, "%"+params.Search+"%")
}
```

Change to always include the deleted_at filter:

```go
query := `SELECT id, name, contact_email, notes, created_at, updated_at FROM customers WHERE deleted_at IS NULL`
args := []any{}
if params.Search != "" {
    query += ` AND name ILIKE $1`
    args = append(args, "%"+params.Search+"%")
}
```

- [ ] **Step 3: Update Delete — change to soft delete**

Replace the Delete method body. Current:

```go
func (r *CustomerRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM customers WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting customer: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("customer not found")
	}
	return nil
}
```

New:

```go
func (r *CustomerRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	result, err := r.pool.Exec(ctx,
		`UPDATE customers SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("deleting customer: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("customer not found")
	}
	return nil
}
```

- [ ] **Step 4: Commit**

```bash
git add internal/repository/customer.go
git commit -m "feat: soft delete and filtered reads for customer repository"
```

---

### Task 6: Update service repository

**Files:**
- Modify: `internal/repository/service.go`

Same pattern as customer — service.go has identical query structure.

- [ ] **Step 1: Update GetByID — add `AND deleted_at IS NULL`**

```go
`SELECT id, name, description, created_at, updated_at
 FROM services WHERE id = $1 AND deleted_at IS NULL`, id,
```

- [ ] **Step 2: Update List — change `WHERE` to include `deleted_at IS NULL`**

```go
query := `SELECT id, name, description, created_at, updated_at FROM services WHERE deleted_at IS NULL`
args := []any{}
if params.Search != "" {
    query += ` AND name ILIKE $1`
    args = append(args, "%"+params.Search+"%")
}
```

- [ ] **Step 3: Update Delete — change to soft delete**

```go
func (r *ServiceRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	result, err := r.pool.Exec(ctx,
		`UPDATE services SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("deleting service: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("service not found")
	}
	return nil
}
```

- [ ] **Step 4: Commit**

```bash
git add internal/repository/service.go
git commit -m "feat: soft delete and filtered reads for service repository"
```

---

### Task 7: Update user repository

**Files:**
- Modify: `internal/repository/user.go`

The user repo has a special List with `clauses` array for type filter + search. Add `deleted_at IS NULL` as the base clause.

- [ ] **Step 1: Update GetByID — add `AND deleted_at IS NULL`**

```go
`SELECT id, type, first_name, last_name, created_at, updated_at
 FROM users WHERE id = $1 AND deleted_at IS NULL`, id,
```

- [ ] **Step 2: Update List — add `deleted_at IS NULL` as base clause**

Current pattern uses a `clauses` slice. Add `deleted_at IS NULL` as the always-present first clause:

```go
query := `SELECT id, type, first_name, last_name, created_at, updated_at FROM users`
args := []any{}
argN := 1
clauses := []string{"deleted_at IS NULL"}
```

The rest of the method (filter, search, WHERE join) stays exactly the same — `deleted_at IS NULL` will always be in the clauses slice, so `len(clauses) > 0` is always true and the `WHERE` is always appended.

- [ ] **Step 3: Update Delete — change to soft delete**

```go
func (r *UserRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	result, err := r.pool.Exec(ctx,
		`UPDATE users SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}
```

- [ ] **Step 4: Commit**

```bash
git add internal/repository/user.go
git commit -m "feat: soft delete and filtered reads for user repository"
```

---

### Task 8: Update user_assignment repository

**Files:**
- Modify: `internal/repository/user_assignment.go`

Has `ListByCustomer` with `WHERE customer_id = $1` — add `AND deleted_at IS NULL`.

- [ ] **Step 1: Update GetByID — add `AND deleted_at IS NULL`**

```go
`SELECT id, user_id, customer_id, role, email, phone, notes, created_at, updated_at
 FROM user_assignments WHERE id = $1 AND deleted_at IS NULL`, id,
```

- [ ] **Step 2: Update ListByCustomer — add `AND deleted_at IS NULL`**

```go
query := `SELECT id, user_id, customer_id, role, email, phone, notes, created_at, updated_at
          FROM user_assignments WHERE customer_id = $1 AND deleted_at IS NULL`
```

- [ ] **Step 3: Update Delete — change to soft delete**

```go
func (r *UserAssignmentRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	result, err := r.pool.Exec(ctx,
		`UPDATE user_assignments SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("deleting user assignment: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("user assignment not found")
	}
	return nil
}
```

- [ ] **Step 4: Commit**

```bash
git add internal/repository/user_assignment.go
git commit -m "feat: soft delete and filtered reads for user_assignment repository"
```

---

### Task 9: Update asset repository

**Files:**
- Modify: `internal/repository/asset.go`

Has `ListByCustomer` with filter and search — add `AND deleted_at IS NULL` to the base WHERE.

- [ ] **Step 1: Update GetByID — add `AND deleted_at IS NULL`**

```go
`SELECT id, customer_id, category_id, user_assignment_id, name, description, metadata, field_values, created_at, updated_at
 FROM assets WHERE id = $1 AND deleted_at IS NULL`, id,
```

- [ ] **Step 2: Update ListByCustomer — add `AND deleted_at IS NULL`**

```go
query := `SELECT id, customer_id, category_id, user_assignment_id, name, description, metadata, field_values, created_at, updated_at
          FROM assets WHERE customer_id = $1 AND deleted_at IS NULL`
```

- [ ] **Step 3: Update Delete — change to soft delete**

```go
func (r *AssetRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	result, err := r.pool.Exec(ctx,
		`UPDATE assets SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("deleting asset: %w", err)
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
```

**Note:** Asset repo returns `pgx.ErrNoRows` (not `fmt.Errorf`), preserve this existing behavior.

- [ ] **Step 4: Commit**

```bash
git add internal/repository/asset.go
git commit -m "feat: soft delete and filtered reads for asset repository"
```

---

### Task 10: Update license repository

**Files:**
- Modify: `internal/repository/license.go`

Same pattern as asset — `ListByCustomer` with `WHERE customer_id = $1`.

- [ ] **Step 1: Update GetByID — add `AND deleted_at IS NULL`**

```go
`SELECT id, customer_id, user_assignment_id, product_name, license_key, quantity, valid_from, valid_until, created_at, updated_at
 FROM licenses WHERE id = $1 AND deleted_at IS NULL`, id,
```

- [ ] **Step 2: Update ListByCustomer — add `AND deleted_at IS NULL`**

```go
query := `SELECT id, customer_id, user_assignment_id, product_name, license_key, quantity, valid_from, valid_until, created_at, updated_at
          FROM licenses WHERE customer_id = $1 AND deleted_at IS NULL`
```

- [ ] **Step 3: Update Delete — change to soft delete**

```go
func (r *LicenseRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	result, err := r.pool.Exec(ctx,
		`UPDATE licenses SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("deleting license: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("license not found")
	}
	return nil
}
```

- [ ] **Step 4: Commit**

```bash
git add internal/repository/license.go
git commit -m "feat: soft delete and filtered reads for license repository"
```

---

### Task 11: Update customer_service repository

**Files:**
- Modify: `internal/repository/customer_service.go`

Same ListByCustomer pattern.

- [ ] **Step 1: Update GetByID — add `AND deleted_at IS NULL`**

```go
`SELECT id, customer_id, service_id, customizations, notes, created_at, updated_at
 FROM customer_services WHERE id = $1 AND deleted_at IS NULL`, id,
```

- [ ] **Step 2: Update ListByCustomer — add `AND deleted_at IS NULL`**

```go
query := `SELECT id, customer_id, service_id, customizations, notes, created_at, updated_at
          FROM customer_services WHERE customer_id = $1 AND deleted_at IS NULL`
```

- [ ] **Step 3: Update Delete — change to soft delete**

```go
func (r *CustomerServiceRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	result, err := r.pool.Exec(ctx,
		`UPDATE customer_services SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("deleting customer service: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("customer service not found")
	}
	return nil
}
```

- [ ] **Step 4: Commit**

```bash
git add internal/repository/customer_service.go
git commit -m "feat: soft delete and filtered reads for customer_service repository"
```

---

### Task 12: Update hardware_category repository

**Files:**
- Modify: `internal/repository/hardware_category.go`

This repo has special patterns: `List()` with no params, `ListFields()`, and `DeleteField()`.

- [ ] **Step 1: Update GetByID — add `AND deleted_at IS NULL`**

In the GetByID query:

```go
`SELECT id, name, description, created_at, updated_at
 FROM hardware_categories WHERE id = $1 AND deleted_at IS NULL`, id,
```

- [ ] **Step 2: Update List — add `WHERE deleted_at IS NULL`**

Currently has no WHERE clause:

```go
`SELECT id, name, description, created_at, updated_at
 FROM hardware_categories ORDER BY name`
```

Change to:

```go
`SELECT id, name, description, created_at, updated_at
 FROM hardware_categories WHERE deleted_at IS NULL ORDER BY name`
```

- [ ] **Step 3: Update ListFields — add `AND deleted_at IS NULL`**

```go
`SELECT id, category_id, name, field_type, required, sort_order, created_at, updated_at
 FROM category_field_definitions
 WHERE category_id = $1 AND deleted_at IS NULL
 ORDER BY sort_order, name`, categoryID)
```

- [ ] **Step 4: Update Delete — change to soft delete**

```go
func (r *HardwareCategoryRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE hardware_categories SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("deleting hardware category: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
```

- [ ] **Step 5: Update DeleteField — change to soft delete**

```go
func (r *HardwareCategoryRepository) DeleteField(ctx context.Context, fieldID pgtype.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE category_field_definitions SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, fieldID)
	if err != nil {
		return fmt.Errorf("deleting field definition: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
```

- [ ] **Step 6: Commit**

```bash
git add internal/repository/hardware_category.go
git commit -m "feat: soft delete and filtered reads for hardware_category repository"
```

---

### Task 13: Update Update queries — add `AND deleted_at IS NULL`

**Files:**
- Modify: `internal/repository/customer.go`
- Modify: `internal/repository/service.go`
- Modify: `internal/repository/user.go`
- Modify: `internal/repository/user_assignment.go`
- Modify: `internal/repository/asset.go`
- Modify: `internal/repository/license.go`
- Modify: `internal/repository/customer_service.go`
- Modify: `internal/repository/hardware_category.go`

All Update methods have `WHERE id = $1` — add `AND deleted_at IS NULL` so that updating a soft-deleted record returns "not found" instead of silently succeeding.

- [ ] **Step 1: Add `AND deleted_at IS NULL` to all Update queries**

For each repository's Update method, change:

```sql
WHERE id = $1
```

to:

```sql
WHERE id = $1 AND deleted_at IS NULL
```

This applies to all 8 repository files. Also apply to `UpdateField` in `hardware_category.go` if present.

- [ ] **Step 2: Run `go vet ./...` to verify no syntax errors**

```bash
go vet ./...
```

Expected: no output (clean).

- [ ] **Step 3: Commit**

```bash
git add internal/repository/
git commit -m "feat: prevent updates to soft-deleted records across all repositories"
```

---

### Task 14: Build and verify with Docker

- [ ] **Step 1: Run `go vet ./...`**

```bash
go vet ./...
```

Expected: no output (clean).

- [ ] **Step 2: Build and start with Docker Compose**

```bash
docker compose up --build
```

Expected: application starts, migration 008 runs successfully, no errors in logs.

- [ ] **Step 3: Verify seed data is still accessible**

Open `http://localhost:8080/customers` — confirm list loads.
Open `http://localhost:8080/users` — confirm list loads.

- [ ] **Step 4: Test soft delete via API**

Create a test customer, then delete it:

```bash
# Create
curl -X POST http://localhost:8080/api/v1/customers -H 'Content-Type: application/json' -d '{"name":"Test Soft Delete"}'
# Note the returned id

# Delete (soft)
curl -X DELETE http://localhost:8080/api/v1/customers/{id}
# Expected: 204

# Verify hidden from list
curl http://localhost:8080/api/v1/customers
# "Test Soft Delete" should NOT appear

# Verify hidden from get
curl http://localhost:8080/api/v1/customers/{id}
# Expected: 404 or error
```

- [ ] **Step 5: Verify audit log has entries**

Connect to the database and check:

```sql
SELECT * FROM audit_log ORDER BY id DESC LIMIT 10;
```

Expected: INSERT and DELETE entries for the test customer.

- [ ] **Step 6: Test guard trigger — try deleting customer with dependents**

Try to delete a customer that has assets/licenses (use seed data). Expected: 409 Conflict with "has dependent records" message.

- [ ] **Step 7: Commit any fixes if needed, then final commit**

```bash
git add -A
git commit -m "chore: verify soft delete and audit trail implementation"
```
