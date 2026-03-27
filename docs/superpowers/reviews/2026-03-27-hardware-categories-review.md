# Hardware Categories Feature - Code Review

**Reviewer:** Senior Code Reviewer (automated)
**Date:** 2026-03-27
**Branch commits:** 7f3db7d through 9859121 (12 commits)

## Overall Assessment: APPROVED with minor issues

The implementation is well-executed, follows the spec closely, and adheres to established codebase patterns. SQL column orders are correct, error handling is consistent, and the PostgreSQL-first approach (trigger for field cleanup) is properly implemented.

---

## Spec Compliance: PASS

All spec requirements are implemented:

- [x] `hardware_categories` and `category_field_definitions` tables with correct constraints
- [x] `strip_deleted_field_values` trigger (PostgreSQL-first cleanup)
- [x] `assets.type` removed, `category_id` and `field_values` added
- [x] ON DELETE SET NULL on `assets.category_id`, ON DELETE CASCADE on field definitions
- [x] GIN index on `field_values`
- [x] 5 seed categories match spec
- [x] All 8 API endpoints registered (5 category, 3 field definition)
- [x] `field_values` validation: unknown keys rejected, type checking for text/number/date/boolean
- [x] `AssetResponse` with inline category on GET
- [x] `category_id` filter on asset list
- [x] Category change resets `field_values` to `{}`
- [x] `required` flag stored but not enforced (per MVP spec)
- [x] Data model diagram updated

## Code Quality: GOOD

### What was done well

- **Consistent patterns**: New code mirrors existing handler/repository structure exactly
- **Proper error wrapping**: All repository errors use `fmt.Errorf("context: %w", err)`
- **Smart struct separation**: `AssetResponse` wraps `Asset` to keep `RowToStructByPos` working
- **Manual scan for categories**: Correctly avoids `RowToStructByPos` for `HardwareCategory` since the `Fields` slice isn't a DB column
- **Field type validation in handler**: Validates `field_type` before hitting DB CHECK constraint
- **Duplicate name handling**: `isUniqueViolation` checks in both category and field creation
- **Clean migration with proper DOWN block**: Reversible migration with correct ordering

### SQL Correctness: PASS

All `SELECT`, `RETURNING`, and `.Scan()` calls have columns in the correct order matching struct field definitions. Verified for:
- `Asset`: 9 columns (id, customer_id, category_id, name, description, metadata, field_values, created_at, updated_at)
- `HardwareCategory`: 5 columns (id, name, description, created_at, updated_at) -- Fields populated separately
- `FieldDefinition`: 8 columns matching struct order exactly

---

## Issues Found

### Important (should fix)

**1. Asset `Delete` error inconsistency** (`internal/repository/asset.go:109`)

The asset repository's `Delete` returns `fmt.Errorf("asset not found")` (a plain error) when `RowsAffected() == 0`, while all other repositories return `pgx.ErrNoRows`. The handler works around this by using a fallthrough to 404, but this is fragile and inconsistent.

**Recommendation:** Change to match the hardware_category pattern:
```go
if result.RowsAffected() == 0 {
    return pgx.ErrNoRows
}
```
And update the handler's delete to check `errors.Is(err, pgx.ErrNoRows)` like other handlers do.

**2. Missing test: `category_id` filter in `ListByCustomer`** (`internal/repository/asset_test.go`)

The asset test creates assets with and without categories but never tests `ListByCustomer` with `params.Filter` set to a category ID. This is a key new feature that should have test coverage.

**Recommendation:** Add a test case:
```go
// ListByCustomer with category filter
filtered, err := repo.ListByCustomer(ctx, customer.ID, model.ListParams{Limit: 10, Filter: cat.ID.Bytes[:]...})
```

### Suggestions (nice to have)

**3. Missing test: unique constraint violations** (`internal/repository/hardware_category_test.go`)

The handler correctly handles unique violations for category names and field names, but the repository tests don't verify that creating a duplicate name produces the expected error. Adding this would guard against accidentally dropping the UNIQUE constraint in future migrations.

**4. Missing test: not-found error on delete of nonexistent record** (`internal/repository/hardware_category_test.go`)

After the delete test, verifying that deleting the same ID again returns `pgx.ErrNoRows` would strengthen the test.

**5. Missing test: update with category change** (`internal/repository/asset_test.go`)

The update test only changes `Name`. A test that changes `CategoryID` and verifies `field_values` is reset to `{}` would validate the spec requirement for category change behavior.

**6. `ListByCustomer` filter passes raw string to UUID parameter** (`internal/repository/asset.go:72`)

The `params.Filter` is a string from the query parameter, passed directly as `args = append(args, params.Filter)` for a UUID column comparison. PostgreSQL will cast it, but if someone passes a non-UUID string, the error message will be a DB error rather than a clean 400. The handler doesn't validate that `category_id` query param is a valid UUID before passing it to the repo.

**Recommendation:** Validate the `category_id` query parameter with `parseUUID()` before setting `params.Filter`, or parse it in the repository.

---

## Test Coverage Summary

| Area | Coverage |
|------|----------|
| Category CRUD | Full |
| Field Definition CRUD | Full |
| Asset CRUD with category | Partial (missing filter, update-category) |
| Field validation types | Full (text, number, boolean, date) |
| Field validation edges | Good (null, empty, unknown keys, bad format) |
| Unique constraint errors | Not tested at repo level |
| Handler integration | Not tested (acceptable for MVP) |

---

## Files Reviewed

- `db/migrations/003_hardware_categories.sql` -- Clean, correct
- `internal/model/hardware_category.go` -- Clean, matches plan
- `internal/model/asset.go` -- Clean, AssetResponse design is good
- `internal/repository/hardware_category.go` -- Clean, correct SQL
- `internal/repository/asset.go` -- One inconsistency (Delete error)
- `internal/handler/hardware_category.go` -- Clean, proper error handling
- `internal/handler/asset.go` -- Clean, good validation flow
- `internal/handler/field_validation.go` -- Clean, all types covered
- `cmd/server/main.go` -- Properly wired
- `internal/repository/hardware_category_test.go` -- Good, missing edge cases
- `internal/repository/asset_test.go` -- Good, missing filter test
- `internal/handler/field_validation_test.go` -- Good coverage
