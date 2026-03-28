# Asset User Assignment — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add optional `user_assignment_id` to assets, allowing an asset to be assigned to a user of the same customer — mirroring the existing license pattern.

**Architecture:** New migration adds the nullable FK column + consistency trigger (same pattern as `check_license_customer_consistency`). Model, repository, and handler are updated to include the new field in all CRUD operations.

**Tech Stack:** PostgreSQL (goose migration), Go (pgx)

---

## File Map

| Action | File | Responsibility |
|--------|------|----------------|
| Create | `db/migrations/006_asset_user_assignment.sql` | Migration: column + trigger |
| Modify | `docs/data-model.mmd` | Add FK field + relationship line |
| Modify | `internal/model/asset.go` | Add `UserAssignmentID` to all structs |
| Modify | `internal/repository/asset.go` | Add column to all SQL queries |
| Modify | `internal/handler/asset.go` | Pass new field through CRUD |

---

### Task 1: Database migration

**Files:**
- Create: `db/migrations/006_asset_user_assignment.sql`

- [ ] **Step 1: Write the migration file**

```sql
-- +goose Up

ALTER TABLE assets
    ADD COLUMN user_assignment_id UUID REFERENCES user_assignments(id) ON DELETE RESTRICT;

-- Consistency trigger: ensure user_assignment belongs to same customer as the asset
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION check_asset_customer_consistency()
RETURNS TRIGGER AS $$
DECLARE
    assignment_customer_id UUID;
BEGIN
    IF NEW.user_assignment_id IS NOT NULL THEN
        SELECT customer_id INTO assignment_customer_id
        FROM user_assignments
        WHERE id = NEW.user_assignment_id;

        IF assignment_customer_id IS NULL THEN
            RAISE EXCEPTION 'user_assignment_id % does not exist', NEW.user_assignment_id;
        END IF;

        IF assignment_customer_id != NEW.customer_id THEN
            RAISE EXCEPTION 'asset customer_id (%) does not match user_assignment customer_id (%)',
                NEW.customer_id, assignment_customer_id;
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_assets_customer_consistency
    BEFORE INSERT OR UPDATE ON assets
    FOR EACH ROW EXECUTE FUNCTION check_asset_customer_consistency();

-- +goose Down
DROP TRIGGER IF EXISTS trg_assets_customer_consistency ON assets;
DROP FUNCTION IF EXISTS check_asset_customer_consistency;
ALTER TABLE assets DROP COLUMN IF EXISTS user_assignment_id;
```

- [ ] **Step 2: Rebuild and verify migration runs**

```bash
docker compose up --build -d
docker compose logs app | tail -20
```

Expected: no migration errors, app starts normally.

- [ ] **Step 3: Commit**

```bash
git add db/migrations/006_asset_user_assignment.sql
git commit -m "feat: add user_assignment_id column and consistency trigger to assets"
```

---

### Task 2: Update data model diagram

**Files:**
- Modify: `docs/data-model.mmd`

- [ ] **Step 1: Add `user_assignment_id` field to `assets` entity**

In the `assets` block, add after `category_id`:

```
UUID user_assignment_id FK "nullable"
```

- [ ] **Step 2: Add relationship line**

Add after the existing `hardware_categories ||--o{ assets : "categorizes"` line:

```
user_assignments ||--o{ assets : "assigned to (optional)"
```

- [ ] **Step 3: Commit**

```bash
git add docs/data-model.mmd
git commit -m "docs: add asset-user_assignment relationship to data model"
```

---

### Task 3: Update model structs

**Files:**
- Modify: `internal/model/asset.go`

- [ ] **Step 1: Add `UserAssignmentID` to `Asset` struct**

Add after `CategoryID` line:

```go
UserAssignmentID pgtype.UUID `json:"user_assignment_id"`
```

- [ ] **Step 2: Add `UserAssignmentID` to `CreateAssetInput`**

Add after `CategoryID` line:

```go
UserAssignmentID pgtype.UUID `json:"user_assignment_id,omitempty"`
```

- [ ] **Step 3: Add `UserAssignmentID` to `UpdateAssetInput`**

Add after `CategoryID` line (matching the order in `Asset` and `CreateAssetInput`):

```go
UserAssignmentID pgtype.UUID `json:"user_assignment_id,omitempty"`
```

- [ ] **Step 4: Verify compilation**

```bash
go vet ./...
```

Expected: no errors (will show warnings about unused field — that's fine until repo/handler are updated).

- [ ] **Step 5: Commit**

```bash
git add internal/model/asset.go
git commit -m "feat: add UserAssignmentID field to asset model structs"
```

---

### Task 4: Update repository SQL queries

**Files:**
- Modify: `internal/repository/asset.go`

All SQL queries must include `user_assignment_id` in the correct position (after `category_id`) to match the struct field order for `RowToStructByPos`.

- [ ] **Step 1: Update `Create` method**

Change the INSERT query to include `user_assignment_id`:

```go
err := r.pool.QueryRow(ctx,
    `INSERT INTO assets (customer_id, category_id, user_assignment_id, name, description, metadata, field_values)
     VALUES ($1, $2, $3, $4, $5, $6, $7)
     RETURNING id, customer_id, category_id, user_assignment_id, name, description, metadata, field_values, created_at, updated_at`,
    input.CustomerID, input.CategoryID, input.UserAssignmentID, input.Name, input.Description, input.Metadata, input.FieldValues,
).Scan(&a.ID, &a.CustomerID, &a.CategoryID, &a.UserAssignmentID, &a.Name, &a.Description, &a.Metadata, &a.FieldValues, &a.CreatedAt, &a.UpdatedAt)
```

- [ ] **Step 2: Update `GetByID` method**

```go
err := r.pool.QueryRow(ctx,
    `SELECT id, customer_id, category_id, user_assignment_id, name, description, metadata, field_values, created_at, updated_at
     FROM assets WHERE id = $1`, id,
).Scan(&a.ID, &a.CustomerID, &a.CategoryID, &a.UserAssignmentID, &a.Name, &a.Description, &a.Metadata, &a.FieldValues, &a.CreatedAt, &a.UpdatedAt)
```

- [ ] **Step 3: Update `ListByCustomer` method**

Change the SELECT column list:

```go
query := `SELECT id, customer_id, category_id, user_assignment_id, name, description, metadata, field_values, created_at, updated_at
          FROM assets WHERE customer_id = $1`
```

(`RowToStructByPos` will automatically pick up the new column since the struct field order matches.)

- [ ] **Step 4: Update `Update` method**

Add `user_assignment_id` to the COALESCE update. Note: parameter numbers shift by 1 for all subsequent params.

```go
err := r.pool.QueryRow(ctx,
    `UPDATE assets SET
        category_id      = COALESCE($2, category_id),
        user_assignment_id = COALESCE($3, user_assignment_id),
        name             = COALESCE($4, name),
        description      = COALESCE($5, description),
        metadata         = COALESCE($6, metadata),
        field_values     = COALESCE($7, field_values)
     WHERE id = $1
     RETURNING id, customer_id, category_id, user_assignment_id, name, description, metadata, field_values, created_at, updated_at`,
    id, input.CategoryID, input.UserAssignmentID, input.Name, input.Description, input.Metadata, input.FieldValues,
).Scan(&a.ID, &a.CustomerID, &a.CategoryID, &a.UserAssignmentID, &a.Name, &a.Description, &a.Metadata, &a.FieldValues, &a.CreatedAt, &a.UpdatedAt)
```

- [ ] **Step 5: Verify compilation**

```bash
go vet ./...
```

Expected: no errors.

- [ ] **Step 6: Commit**

```bash
git add internal/repository/asset.go
git commit -m "feat: include user_assignment_id in all asset repository SQL queries"
```

---

### Task 5: Update handler

**Files:**
- Modify: `internal/handler/asset.go`

The handler already passes through input structs via `decodeJSON`, so the new field flows automatically for create/update. The `AssetResponse` embeds `Asset`, so `user_assignment_id` appears in GET responses automatically. One change is needed: FK violation handling for invalid `user_assignment_id`.

- [ ] **Step 1: Add FK violation handling to `create` handler**

After the existing `h.repo.Create()` error check, add FK violation handling. In the `create` method, change the error block after `h.repo.Create`:

```go
asset, err := h.repo.Create(r.Context(), input)
if err != nil {
    if isFKViolation(err) {
        writeError(w, http.StatusConflict, "invalid user_assignment_id or category_id")
        return
    }
    writeError(w, http.StatusInternalServerError, "failed to create asset")
    return
}
```

- [ ] **Step 2: Add FK violation handling to `update` handler**

Same pattern — in the `update` method, change the error block after `h.repo.Update`:

```go
asset, err := h.repo.Update(r.Context(), id, input)
if err != nil {
    if errors.Is(err, pgx.ErrNoRows) {
        writeError(w, http.StatusNotFound, "asset not found")
        return
    }
    if isFKViolation(err) {
        writeError(w, http.StatusConflict, "invalid user_assignment_id or category_id")
        return
    }
    writeError(w, http.StatusInternalServerError, "failed to update asset")
    return
}
```

- [ ] **Step 3: Rebuild and verify end-to-end**

```bash
docker compose up --build -d
```

Verify via `ctx_execute` sandbox (not curl — blocked per project conventions):

```javascript
// Test 1: Create asset with valid user_assignment_id
const r1 = await fetch('http://localhost:8080/api/v1/customers/{customerId}/assets', {
  method: 'POST',
  headers: {'Content-Type': 'application/json'},
  body: JSON.stringify({name: 'Test Asset', user_assignment_id: '{validAssignmentId}'})
});
console.log('Create:', r1.status, await r1.json());

// Test 2: GET should include user_assignment_id
const r2 = await fetch('http://localhost:8080/api/v1/assets/{assetId}');
console.log('Get:', await r2.json());

// Test 3: Cross-customer assignment should fail with 409
const r3 = await fetch('http://localhost:8080/api/v1/customers/{customerId}/assets', {
  method: 'POST',
  headers: {'Content-Type': 'application/json'},
  body: JSON.stringify({name: 'Bad Asset', user_assignment_id: '{wrongCustomerAssignmentId}'})
});
console.log('Cross-customer:', r3.status); // Expected: 500 (trigger exception)
```

- [ ] **Step 4: Commit**

```bash
git add internal/handler/asset.go
git commit -m "feat: add FK violation handling for user_assignment_id in asset handler"
```

---

## Verification Checklist

- [ ] Migration runs without errors on `docker compose up --build`
- [ ] Existing assets still work (user_assignment_id is NULL)
- [ ] New asset with valid user_assignment_id creates successfully
- [ ] Consistency trigger rejects mismatched customer_id
- [ ] GET /api/v1/assets/{id} returns user_assignment_id in response
- [ ] Update with user_assignment_id works
- [ ] `go vet ./...` passes
