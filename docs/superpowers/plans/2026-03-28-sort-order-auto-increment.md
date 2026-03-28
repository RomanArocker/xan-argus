# Sort Order Auto-Increment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** New `category_field_definitions` records automatically receive a `sort_order` equal to the current maximum for that category plus one.

**Architecture:** Replace the static `sortOrder` local variable in `HardwareCategoryRepository.CreateField` with a PostgreSQL subquery `(SELECT COALESCE(MAX(sort_order), -1) + 1 FROM category_field_definitions WHERE category_id = $1)`. No other layers change.

**Tech Stack:** Go, PostgreSQL 18, pgx, goose

---

### Task 1: Add failing test for sort_order auto-increment

**Files:**
- Modify: `internal/repository/hardware_category_test.go` — extend `TestFieldDefinitionCRUD` with a second field and assert incrementing sort_order values

- [ ] **Step 1: Add the test assertions**

In `TestFieldDefinitionCRUD`, add two things:

**a)** Immediately after the `if field.Required { ... }` block (before the `GetByID` section), assert the first field's sort_order. Note: this assertion will pass even before the fix (current code defaults to 0), but it guards against regressions:

```go
// sort_order of first field should be 0
if field.SortOrder != 0 {
    t.Errorf("First field SortOrder = %d, want 0", field.SortOrder)
}
```

**b)** After the existing `UpdateField` section (and before `DeleteField`), add a second field creation and assert its sort_order increments. This is the assertion that FAILS before the fix. Place it here to avoid conflicting with the `GetByID` assertion that checks `len(got.Fields) != 1`:

```go
// Create a second field — sort_order should auto-increment to 1
field2Input := model.CreateFieldDefinitionInput{
    CategoryID: cat.ID,
    Name:       "CPU Cores",
    FieldType:  "number",
}
field2, err := repo.CreateField(ctx, field2Input)
if err != nil {
    t.Fatalf("CreateField (2nd): %v", err)
}
if field2.SortOrder != 1 {
    t.Errorf("Second field SortOrder = %d, want 1", field2.SortOrder)
}
// Cleanup field2 (category cleanup at top also handles this)
defer repo.DeleteField(ctx, field2.ID) //nolint:errcheck
```

Note: `SortOrder *int` remains on `CreateFieldDefinitionInput` but is no longer used by `CreateField`. Any caller-supplied value is silently ignored. This is intentional — the field is kept to avoid a breaking change and can be removed in a future cleanup.

- [ ] **Step 2: Verify it compiles**

```bash
go vet ./internal/repository/
```

Expected: no output (no errors)

- [ ] **Step 3: Run test to confirm the second assertion fails**

```bash
go test ./internal/repository/ -run TestFieldDefinitionCRUD -v
```

Expected: FAIL — `Second field SortOrder = 0, want 1` (current code assigns `sort_order = 0` to all fields).

---

### Task 2: Update repository to use subquery

**Files:**
- Modify: `internal/repository/hardware_category.go:118-134` — replace static sort_order logic with SQL subquery

- [ ] **Step 1: Replace `CreateField` implementation**

Remove the `sortOrder` local variable and the `if input.SortOrder != nil` block. Change the INSERT query to compute sort_order via subquery. The `$4` parameter is removed — only 3 parameters remain:

```go
func (r *HardwareCategoryRepository) CreateField(ctx context.Context, input model.CreateFieldDefinitionInput) (model.FieldDefinition, error) {
	var f model.FieldDefinition
	err := r.pool.QueryRow(ctx,
		`INSERT INTO category_field_definitions (category_id, name, field_type, sort_order)
		 VALUES ($1, $2, $3,
		   (SELECT COALESCE(MAX(sort_order), -1) + 1
		    FROM category_field_definitions WHERE category_id = $1))
		 RETURNING id, category_id, name, field_type, required, sort_order, created_at, updated_at`,
		input.CategoryID, input.Name, input.FieldType,
	).Scan(&f.ID, &f.CategoryID, &f.Name, &f.FieldType, &f.Required, &f.SortOrder, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return f, fmt.Errorf("creating field definition: %w", err)
	}
	return f, nil
}
```

- [ ] **Step 2: Run the targeted test**

```bash
go test ./internal/repository/ -run TestFieldDefinitionCRUD -v
```

Expected: PASS

- [ ] **Step 3: Run full test suite**

```bash
go test ./...
```

Expected: all PASS

- [ ] **Step 4: Static analysis**

```bash
go vet ./...
```

Expected: no output (no errors)

- [ ] **Step 5: Commit**

```bash
git add internal/repository/hardware_category.go internal/repository/hardware_category_test.go
git commit -m "feat: auto-increment sort_order on field definition create"
```
