# Asset Hardware Categories Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the static asset type enum with user-managed hardware categories that carry typed custom field definitions.

**Architecture:** New `hardware_categories` and `category_field_definitions` tables with a new JSONB `field_values` column on `assets`. Categories are global, field values are validated against their category's field definitions. Follows existing model/repository/handler pattern.

**Tech Stack:** Go (stdlib + pgx), PostgreSQL, goose migrations

**Spec:** `docs/superpowers/specs/2026-03-27-asset-hardware-categories-design.md`

---

### Task 1: Database Migration

**Files:**
- Create: `db/migrations/003_hardware_categories.sql`

- [ ] **Step 1: Write the migration UP block**

```sql
-- db/migrations/003_hardware_categories.sql

-- +goose Up

-- hardware_categories
CREATE TABLE hardware_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER trg_hardware_categories_updated_at
    BEFORE UPDATE ON hardware_categories
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- category_field_definitions
CREATE TABLE category_field_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id UUID NOT NULL REFERENCES hardware_categories(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    field_type TEXT NOT NULL CHECK (field_type IN ('text', 'number', 'date', 'boolean')),
    required BOOLEAN NOT NULL DEFAULT false,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(category_id, name)
);

CREATE TRIGGER trg_category_field_definitions_updated_at
    BEFORE UPDATE ON category_field_definitions
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- strip_deleted_field_values trigger
-- +goose StatementBegin
CREATE FUNCTION strip_deleted_field_values() RETURNS trigger AS $$
BEGIN
    UPDATE assets
    SET field_values = field_values - OLD.id::text
    WHERE category_id = OLD.category_id;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_strip_deleted_field_values
    AFTER DELETE ON category_field_definitions
    FOR EACH ROW EXECUTE FUNCTION strip_deleted_field_values();

-- Modify assets table: remove type, add category_id and field_values
DELETE FROM assets WHERE type = 'software';

ALTER TABLE assets DROP COLUMN type;

ALTER TABLE assets ADD COLUMN category_id UUID REFERENCES hardware_categories(id) ON DELETE SET NULL;

ALTER TABLE assets ADD COLUMN field_values JSONB NOT NULL DEFAULT '{}';

CREATE INDEX idx_assets_field_values ON assets USING GIN (field_values);

-- Seed default categories
INSERT INTO hardware_categories (name, description) VALUES
    ('Laptop', 'Portable computers'),
    ('Server', 'Rack-mounted or tower servers'),
    ('Printer', 'Printers and multifunction devices'),
    ('Monitor', 'Displays and screens'),
    ('Network Device', 'Switches, routers, access points');

-- +goose Down
ALTER TABLE assets DROP COLUMN IF EXISTS field_values;
ALTER TABLE assets DROP COLUMN IF EXISTS category_id;
ALTER TABLE assets ADD COLUMN type TEXT NOT NULL DEFAULT 'hardware' CHECK (type IN ('hardware', 'software'));
DROP TRIGGER IF EXISTS trg_strip_deleted_field_values ON category_field_definitions;
DROP FUNCTION IF EXISTS strip_deleted_field_values;
DROP TABLE IF EXISTS category_field_definitions;
DROP TABLE IF EXISTS hardware_categories;
```

- [ ] **Step 2: Verify migration runs**

Run: `docker compose up --build`
Expected: Application starts, logs show "Migrations completed"

- [ ] **Step 3: Commit**

```bash
git add db/migrations/003_hardware_categories.sql
git commit -m "feat: add hardware categories migration with seed data"
```

---

### Task 2: Hardware Category Models

**Files:**
- Create: `internal/model/hardware_category.go`

- [ ] **Step 1: Write the model structs**

```go
package model

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type HardwareCategory struct {
	ID          pgtype.UUID      `json:"id"`
	Name        string           `json:"name"`
	Description pgtype.Text      `json:"description"`
	Fields      []FieldDefinition `json:"fields,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

type FieldDefinition struct {
	ID         pgtype.UUID `json:"id"`
	CategoryID pgtype.UUID `json:"category_id"`
	Name       string      `json:"name"`
	FieldType  string      `json:"field_type"`
	Required   bool        `json:"required"`
	SortOrder  int         `json:"sort_order"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
}

type CreateHardwareCategoryInput struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

type UpdateHardwareCategoryInput struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

type CreateFieldDefinitionInput struct {
	CategoryID pgtype.UUID `json:"-"`
	Name       string      `json:"name"`
	FieldType  string      `json:"field_type"`
	SortOrder  *int        `json:"sort_order,omitempty"`
}

type UpdateFieldDefinitionInput struct {
	Name      *string `json:"name,omitempty"`
	SortOrder *int    `json:"sort_order,omitempty"`
}
```

- [ ] **Step 2: Verify compilation**

Run: `go vet ./internal/model/...`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add internal/model/hardware_category.go
git commit -m "feat: add hardware category and field definition models"
```

---

### Task 3: Update Asset Model

**Files:**
- Modify: `internal/model/asset.go`

- [ ] **Step 1: Update Asset struct — remove Type, add CategoryID and FieldValues**

Replace the `Asset` struct:

```go
package model

import (
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type Asset struct {
	ID          pgtype.UUID     `json:"id"`
	CustomerID  pgtype.UUID     `json:"customer_id"`
	CategoryID  pgtype.UUID     `json:"category_id"`
	Name        string          `json:"name"`
	Description pgtype.Text     `json:"description"`
	Metadata    json.RawMessage `json:"metadata"`
	FieldValues json.RawMessage `json:"field_values"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// AssetResponse wraps Asset with an inline category for API responses.
// Kept separate from Asset so RowToStructByPos still works for DB scanning.
type AssetResponse struct {
	Asset
	Category *HardwareCategory `json:"category,omitempty"`
}

type CreateAssetInput struct {
	CustomerID  pgtype.UUID     `json:"customer_id"`
	CategoryID  pgtype.UUID     `json:"category_id,omitempty"`
	Name        string          `json:"name"`
	Description *string         `json:"description,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	FieldValues json.RawMessage `json:"field_values,omitempty"`
}

type UpdateAssetInput struct {
	CategoryID  pgtype.UUID     `json:"category_id,omitempty"`
	Name        *string         `json:"name,omitempty"`
	Description *string         `json:"description,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	FieldValues json.RawMessage `json:"field_values,omitempty"`
}
```

**Key changes:**
- `Type` field removed from all three structs
- `CategoryID pgtype.UUID` added (nullable — pgtype.UUID handles NULL)
- `FieldValues json.RawMessage` added to `Asset` and both input structs

- [ ] **Step 2: Verify compilation**

Run: `go vet ./internal/model/...`
Expected: Compilation errors in handler/repository (expected — they still reference `Type`). Model package itself should be clean.

- [ ] **Step 3: Commit**

```bash
git add internal/model/asset.go
git commit -m "feat: update asset model with category_id and field_values"
```

---

### Task 4: Hardware Category Repository

**Files:**
- Create: `internal/repository/hardware_category.go`

- [ ] **Step 1: Write the repository**

```go
package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xan-com/xan-argus/internal/model"
)

type HardwareCategoryRepository struct {
	pool *pgxpool.Pool
}

func NewHardwareCategoryRepository(pool *pgxpool.Pool) *HardwareCategoryRepository {
	return &HardwareCategoryRepository{pool: pool}
}

// --- Categories ---

func (r *HardwareCategoryRepository) Create(ctx context.Context, input model.CreateHardwareCategoryInput) (model.HardwareCategory, error) {
	var c model.HardwareCategory
	err := r.pool.QueryRow(ctx,
		`INSERT INTO hardware_categories (name, description)
		 VALUES ($1, $2)
		 RETURNING id, name, description, created_at, updated_at`,
		input.Name, input.Description,
	).Scan(&c.ID, &c.Name, &c.Description, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return c, fmt.Errorf("creating hardware category: %w", err)
	}
	return c, nil
}

func (r *HardwareCategoryRepository) GetByID(ctx context.Context, id pgtype.UUID) (model.HardwareCategory, error) {
	var c model.HardwareCategory
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, description, created_at, updated_at
		 FROM hardware_categories WHERE id = $1`, id,
	).Scan(&c.ID, &c.Name, &c.Description, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return c, fmt.Errorf("getting hardware category: %w", err)
	}

	fields, err := r.ListFields(ctx, id)
	if err != nil {
		return c, fmt.Errorf("listing fields for category: %w", err)
	}
	c.Fields = fields
	return c, nil
}

func (r *HardwareCategoryRepository) List(ctx context.Context) ([]model.HardwareCategory, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, description, created_at, updated_at
		 FROM hardware_categories ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("listing hardware categories: %w", err)
	}
	defer rows.Close()
	// Manual scan — HardwareCategory has a Fields slice that RowToStructByPos can't skip
	var categories []model.HardwareCategory
	for rows.Next() {
		var c model.HardwareCategory
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning hardware category: %w", err)
		}
		categories = append(categories, c)
	}
	return categories, rows.Err()
}

func (r *HardwareCategoryRepository) Update(ctx context.Context, id pgtype.UUID, input model.UpdateHardwareCategoryInput) (model.HardwareCategory, error) {
	var c model.HardwareCategory
	err := r.pool.QueryRow(ctx,
		`UPDATE hardware_categories SET
			name        = COALESCE($2, name),
			description = COALESCE($3, description)
		 WHERE id = $1
		 RETURNING id, name, description, created_at, updated_at`,
		id, input.Name, input.Description,
	).Scan(&c.ID, &c.Name, &c.Description, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return c, fmt.Errorf("updating hardware category: %w", err)
	}
	return c, nil
}

func (r *HardwareCategoryRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM hardware_categories WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting hardware category: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// --- Field Definitions ---

func (r *HardwareCategoryRepository) ListFields(ctx context.Context, categoryID pgtype.UUID) ([]model.FieldDefinition, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, category_id, name, field_type, required, sort_order, created_at, updated_at
		 FROM category_field_definitions
		 WHERE category_id = $1
		 ORDER BY sort_order, name`, categoryID)
	if err != nil {
		return nil, fmt.Errorf("listing field definitions: %w", err)
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToStructByPos[model.FieldDefinition])
}

func (r *HardwareCategoryRepository) CreateField(ctx context.Context, input model.CreateFieldDefinitionInput) (model.FieldDefinition, error) {
	sortOrder := 0
	if input.SortOrder != nil {
		sortOrder = *input.SortOrder
	}
	var f model.FieldDefinition
	err := r.pool.QueryRow(ctx,
		`INSERT INTO category_field_definitions (category_id, name, field_type, sort_order)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, category_id, name, field_type, required, sort_order, created_at, updated_at`,
		input.CategoryID, input.Name, input.FieldType, sortOrder,
	).Scan(&f.ID, &f.CategoryID, &f.Name, &f.FieldType, &f.Required, &f.SortOrder, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return f, fmt.Errorf("creating field definition: %w", err)
	}
	return f, nil
}

func (r *HardwareCategoryRepository) UpdateField(ctx context.Context, fieldID pgtype.UUID, input model.UpdateFieldDefinitionInput) (model.FieldDefinition, error) {
	var f model.FieldDefinition
	err := r.pool.QueryRow(ctx,
		`UPDATE category_field_definitions SET
			name       = COALESCE($2, name),
			sort_order = COALESCE($3, sort_order)
		 WHERE id = $1
		 RETURNING id, category_id, name, field_type, required, sort_order, created_at, updated_at`,
		fieldID, input.Name, input.SortOrder,
	).Scan(&f.ID, &f.CategoryID, &f.Name, &f.FieldType, &f.Required, &f.SortOrder, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return f, fmt.Errorf("updating field definition: %w", err)
	}
	return f, nil
}

func (r *HardwareCategoryRepository) DeleteField(ctx context.Context, fieldID pgtype.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM category_field_definitions WHERE id = $1`, fieldID)
	if err != nil {
		return fmt.Errorf("deleting field definition: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
```

- [ ] **Step 2: Verify compilation**

Run: `go vet ./internal/repository/...`
Expected: No errors (may have errors from asset.go — that's expected, fixed in next task)

- [ ] **Step 3: Commit**

```bash
git add internal/repository/hardware_category.go
git commit -m "feat: add hardware category repository with CRUD and field definitions"
```

---

### Task 5: Update Asset Repository

**Files:**
- Modify: `internal/repository/asset.go`

- [ ] **Step 1: Update all SQL queries to remove `type`, add `category_id` and `field_values`**

Replace all SQL in `asset.go`. The column order in SELECT/INSERT must match the struct field order exactly (for `pgx.RowToStructByPos`).

**Asset struct field order:** `ID, CustomerID, CategoryID, Name, Description, Metadata, FieldValues, CreatedAt, UpdatedAt`

Update `Create`:
```go
func (r *AssetRepository) Create(ctx context.Context, input model.CreateAssetInput) (model.Asset, error) {
	if input.Metadata == nil {
		input.Metadata = json.RawMessage("{}")
	}
	if input.FieldValues == nil {
		input.FieldValues = json.RawMessage("{}")
	}
	var a model.Asset
	err := r.pool.QueryRow(ctx,
		`INSERT INTO assets (customer_id, category_id, name, description, metadata, field_values)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, customer_id, category_id, name, description, metadata, field_values, created_at, updated_at`,
		input.CustomerID, input.CategoryID, input.Name, input.Description, input.Metadata, input.FieldValues,
	).Scan(&a.ID, &a.CustomerID, &a.CategoryID, &a.Name, &a.Description, &a.Metadata, &a.FieldValues, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return a, fmt.Errorf("creating asset: %w", err)
	}
	return a, nil
}
```

Update `GetByID`:
```go
func (r *AssetRepository) GetByID(ctx context.Context, id pgtype.UUID) (model.Asset, error) {
	var a model.Asset
	err := r.pool.QueryRow(ctx,
		`SELECT id, customer_id, category_id, name, description, metadata, field_values, created_at, updated_at
		 FROM assets WHERE id = $1`, id,
	).Scan(&a.ID, &a.CustomerID, &a.CategoryID, &a.Name, &a.Description, &a.Metadata, &a.FieldValues, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return a, fmt.Errorf("getting asset: %w", err)
	}
	return a, nil
}
```

Update `ListByCustomer` — replace `type` filter with `category_id` filter:
```go
func (r *AssetRepository) ListByCustomer(ctx context.Context, customerID pgtype.UUID, params model.ListParams) ([]model.Asset, error) {
	params.Normalize()
	query := `SELECT id, customer_id, category_id, name, description, metadata, field_values, created_at, updated_at
	          FROM assets WHERE customer_id = $1`
	args := []any{customerID}
	argN := 2
	if params.Filter != "" {
		query += fmt.Sprintf(` AND category_id = $%d`, argN)
		args = append(args, params.Filter)
		argN++
	}
	if params.Search != "" {
		query += fmt.Sprintf(` AND name ILIKE $%d`, argN)
		args = append(args, "%"+params.Search+"%")
		argN++
	}
	query += fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, argN, argN+1)
	args = append(args, params.Limit, params.Offset)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing assets: %w", err)
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToStructByPos[model.Asset])
}
```

Update `Update` — replace `type` with `category_id` and `field_values`, handle category change clearing field_values:
```go
func (r *AssetRepository) Update(ctx context.Context, id pgtype.UUID, input model.UpdateAssetInput) (model.Asset, error) {
	var a model.Asset

	// If category_id is being changed and is valid, clear field_values
	// (unless new field_values are also provided)
	if input.CategoryID.Valid && input.FieldValues == nil {
		input.FieldValues = json.RawMessage("{}")
	}

	err := r.pool.QueryRow(ctx,
		`UPDATE assets SET
			category_id  = COALESCE($2, category_id),
			name         = COALESCE($3, name),
			description  = COALESCE($4, description),
			metadata     = COALESCE($5, metadata),
			field_values = COALESCE($6, field_values)
		 WHERE id = $1
		 RETURNING id, customer_id, category_id, name, description, metadata, field_values, created_at, updated_at`,
		id, input.CategoryID, input.Name, input.Description, input.Metadata, input.FieldValues,
	).Scan(&a.ID, &a.CustomerID, &a.CategoryID, &a.Name, &a.Description, &a.Metadata, &a.FieldValues, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return a, fmt.Errorf("updating asset: %w", err)
	}
	return a, nil
}
```

**Note:** The `Delete` method has no column references — it stays unchanged.

- [ ] **Step 2: Verify compilation**

Run: `go vet ./internal/repository/...`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add internal/repository/asset.go
git commit -m "feat: update asset repository for category_id and field_values"
```

---

### Task 6: Hardware Category Handler

**Files:**
- Create: `internal/handler/hardware_category.go`

- [ ] **Step 1: Write the handler**

```go
package handler

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/xan-com/xan-argus/internal/model"
	"github.com/xan-com/xan-argus/internal/repository"
)

type HardwareCategoryHandler struct {
	repo *repository.HardwareCategoryRepository
}

func NewHardwareCategoryHandler(repo *repository.HardwareCategoryRepository) *HardwareCategoryHandler {
	return &HardwareCategoryHandler{repo: repo}
}

func (h *HardwareCategoryHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/hardware-categories", h.list)
	mux.HandleFunc("POST /api/v1/hardware-categories", h.create)
	mux.HandleFunc("GET /api/v1/hardware-categories/{id}", h.get)
	mux.HandleFunc("PUT /api/v1/hardware-categories/{id}", h.update)
	mux.HandleFunc("DELETE /api/v1/hardware-categories/{id}", h.delete)
	mux.HandleFunc("POST /api/v1/hardware-categories/{id}/fields", h.createField)
	mux.HandleFunc("PUT /api/v1/hardware-categories/{id}/fields/{fieldId}", h.updateField)
	mux.HandleFunc("DELETE /api/v1/hardware-categories/{id}/fields/{fieldId}", h.deleteField)
}

// --- Category CRUD ---

func (h *HardwareCategoryHandler) list(w http.ResponseWriter, r *http.Request) {
	categories, err := h.repo.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list categories")
		return
	}
	writeJSON(w, http.StatusOK, categories)
}

func (h *HardwareCategoryHandler) create(w http.ResponseWriter, r *http.Request) {
	var input model.CreateHardwareCategoryInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if input.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	category, err := h.repo.Create(r.Context(), input)
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "category name already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create category")
		return
	}
	writeJSON(w, http.StatusCreated, category)
}

func (h *HardwareCategoryHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category ID")
		return
	}
	category, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "category not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get category")
		return
	}
	writeJSON(w, http.StatusOK, category)
}

func (h *HardwareCategoryHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category ID")
		return
	}
	var input model.UpdateHardwareCategoryInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	category, err := h.repo.Update(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "category not found")
			return
		}
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "category name already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update category")
		return
	}
	writeJSON(w, http.StatusOK, category)
}

func (h *HardwareCategoryHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category ID")
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "category not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete category")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Field Definition CRUD ---

func (h *HardwareCategoryHandler) createField(w http.ResponseWriter, r *http.Request) {
	categoryID, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid category ID")
		return
	}
	var input model.CreateFieldDefinitionInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input.CategoryID = categoryID
	if input.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	validTypes := map[string]bool{"text": true, "number": true, "date": true, "boolean": true}
	if !validTypes[input.FieldType] {
		writeError(w, http.StatusBadRequest, "field_type must be 'text', 'number', 'date', or 'boolean'")
		return
	}
	field, err := h.repo.CreateField(r.Context(), input)
	if err != nil {
		if isFKViolation(err) {
			writeError(w, http.StatusNotFound, "category not found")
			return
		}
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "field name already exists in this category")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create field definition")
		return
	}
	writeJSON(w, http.StatusCreated, field)
}

func (h *HardwareCategoryHandler) updateField(w http.ResponseWriter, r *http.Request) {
	fieldID, err := parseUUID(r.PathValue("fieldId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid field ID")
		return
	}
	var input model.UpdateFieldDefinitionInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	field, err := h.repo.UpdateField(r.Context(), fieldID, input)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "field definition not found")
			return
		}
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "field name already exists in this category")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update field definition")
		return
	}
	writeJSON(w, http.StatusOK, field)
}

func (h *HardwareCategoryHandler) deleteField(w http.ResponseWriter, r *http.Request) {
	fieldID, err := parseUUID(r.PathValue("fieldId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid field ID")
		return
	}
	if err := h.repo.DeleteField(r.Context(), fieldID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "field definition not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to delete field definition")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 2: Verify compilation**

Run: `go vet ./internal/handler/...`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add internal/handler/hardware_category.go
git commit -m "feat: add hardware category handler with category and field CRUD"
```

---

### Task 7: Update Asset Handler

**Files:**
- Modify: `internal/handler/asset.go`

- [ ] **Step 1: Update `AssetHandler` struct to include `catRepo`**

The asset handler needs access to the hardware category repository for field_values validation and populating the inline category in GET responses.

```go
type AssetHandler struct {
	repo    *repository.AssetRepository
	catRepo *repository.HardwareCategoryRepository
}

func NewAssetHandler(repo *repository.AssetRepository, catRepo *repository.HardwareCategoryRepository) *AssetHandler {
	return &AssetHandler{repo: repo, catRepo: catRepo}
}
```

- [ ] **Step 2: Remove type validation from `create`, add field_values validation**

In `create` handler, remove the old type check:
```go
	if input.Type != "hardware" && input.Type != "software" {
		writeError(w, http.StatusBadRequest, "type must be 'hardware' or 'software'")
		return
	}
```

Replace with field_values validation (after name check, before `repo.Create`):
```go
	// Validate field_values against category's field definitions
	if input.CategoryID.Valid && len(input.FieldValues) > 0 && string(input.FieldValues) != "{}" {
		cat, err := h.catRepo.GetByID(r.Context(), input.CategoryID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid category_id")
			return
		}
		if msg := validateFieldValues(input.FieldValues, cat.Fields); msg != "" {
			writeError(w, http.StatusBadRequest, msg)
			return
		}
	}
```

Add the same validation block to the `update` handler before `repo.Update`.

- [ ] **Step 3: Update `get` handler to return `AssetResponse` with inline category**

Per spec, `GET /api/v1/assets/{id}` must return the category object with field definitions inline. Replace the `get` handler:

```go
func (h *AssetHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset ID")
		return
	}
	asset, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "asset not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get asset")
		return
	}
	resp := model.AssetResponse{Asset: asset}
	if asset.CategoryID.Valid {
		cat, err := h.catRepo.GetByID(r.Context(), asset.CategoryID)
		if err == nil {
			resp.Category = &cat
		}
		// If category fetch fails (e.g., deleted), return asset without category — not an error
	}
	writeJSON(w, http.StatusOK, resp)
}
```

- [ ] **Step 4: Override filter param in `listByCustomer`**

Do NOT change the shared `paginationParams` in `json.go` — other handlers may use `?type=`. Instead, override the filter in the asset handler only:

```go
func (h *AssetHandler) listByCustomer(w http.ResponseWriter, r *http.Request) {
	customerID, err := parseUUID(r.PathValue("customerId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer ID")
		return
	}
	params := paginationParams(r)
	params.Filter = r.URL.Query().Get("category_id")
	assets, err := h.repo.ListByCustomer(r.Context(), customerID, params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list assets")
		return
	}
	writeJSON(w, http.StatusOK, assets)
}
```

- [ ] **Step 5: Verify compilation**

Run: `go vet ./internal/handler/...`
Expected: No errors

- [ ] **Step 6: Commit**

```bash
git add internal/handler/asset.go
git commit -m "feat: update asset handler with category response, field validation, category_id filter"
```

---

### Task 8: Wire Up in main.go

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Add repository and handler wiring**

After existing repository declarations, add:
```go
	hardwareCategoryRepo := repository.NewHardwareCategoryRepository(pool)
```

Update the existing `NewAssetHandler` call to pass `hardwareCategoryRepo`:
```go
	handler.NewAssetHandler(assetRepo, hardwareCategoryRepo).RegisterRoutes(mux)
```

After existing handler registrations, add:
```go
	handler.NewHardwareCategoryHandler(hardwareCategoryRepo).RegisterRoutes(mux)
```

Also update `NewPageHandler` if it passes the asset handler or repo — check `page.go` and update accordingly.

- [ ] **Step 2: Verify compilation and startup**

Run: `go vet ./...`
Expected: No errors

Run: `docker compose up --build`
Expected: Application starts successfully

- [ ] **Step 3: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat: wire up hardware category repository and handler"
```

---

### Task 9: Hardware Category Repository Tests

**Files:**
- Create: `internal/repository/hardware_category_test.go`

- [ ] **Step 1: Write the test**

```go
package repository_test

import (
	"context"
	"testing"

	"github.com/xan-com/xan-argus/internal/model"
	"github.com/xan-com/xan-argus/internal/repository"
)

func TestHardwareCategoryCRUD(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	repo := repository.NewHardwareCategoryRepository(pool)

	// Create
	input := model.CreateHardwareCategoryInput{Name: "Test Category"}
	cat, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if cat.Name != "Test Category" {
		t.Errorf("Name = %q, want %q", cat.Name, "Test Category")
	}
	if !cat.ID.Valid {
		t.Error("ID should be valid")
	}
	t.Cleanup(func() { repo.Delete(ctx, cat.ID) }) //nolint:errcheck

	// GetByID (no fields yet)
	got, err := repo.GetByID(ctx, cat.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "Test Category" {
		t.Errorf("GetByID Name = %q, want %q", got.Name, "Test Category")
	}
	if len(got.Fields) != 0 {
		t.Errorf("Fields count = %d, want 0", len(got.Fields))
	}

	// Update
	newName := "Updated Category"
	updated, err := repo.Update(ctx, cat.ID, model.UpdateHardwareCategoryInput{Name: &newName})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "Updated Category" {
		t.Errorf("Updated Name = %q, want %q", updated.Name, "Updated Category")
	}

	// List (should include seeded + test category)
	categories, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(categories) == 0 {
		t.Error("List returned no results")
	}

	// Delete
	if err := repo.Delete(ctx, cat.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = repo.GetByID(ctx, cat.ID)
	if err == nil {
		t.Error("GetByID after Delete should return error")
	}
}

func TestFieldDefinitionCRUD(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	repo := repository.NewHardwareCategoryRepository(pool)

	// Prereq: category
	cat, err := repo.Create(ctx, model.CreateHardwareCategoryInput{Name: "Field Test Category"})
	if err != nil {
		t.Fatalf("Create category: %v", err)
	}
	t.Cleanup(func() { repo.Delete(ctx, cat.ID) }) //nolint:errcheck

	// Create field
	fieldInput := model.CreateFieldDefinitionInput{
		CategoryID: cat.ID,
		Name:       "RAM",
		FieldType:  "number",
	}
	field, err := repo.CreateField(ctx, fieldInput)
	if err != nil {
		t.Fatalf("CreateField: %v", err)
	}
	if field.Name != "RAM" {
		t.Errorf("Name = %q, want %q", field.Name, "RAM")
	}
	if field.FieldType != "number" {
		t.Errorf("FieldType = %q, want %q", field.FieldType, "number")
	}
	if field.Required {
		t.Error("Required should be false by default")
	}

	// GetByID should include field
	got, err := repo.GetByID(ctx, cat.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if len(got.Fields) != 1 {
		t.Fatalf("Fields count = %d, want 1", len(got.Fields))
	}
	if got.Fields[0].Name != "RAM" {
		t.Errorf("Field Name = %q, want %q", got.Fields[0].Name, "RAM")
	}

	// Update field
	newFieldName := "Memory (GB)"
	updatedField, err := repo.UpdateField(ctx, field.ID, model.UpdateFieldDefinitionInput{Name: &newFieldName})
	if err != nil {
		t.Fatalf("UpdateField: %v", err)
	}
	if updatedField.Name != "Memory (GB)" {
		t.Errorf("Updated field Name = %q, want %q", updatedField.Name, "Memory (GB)")
	}

	// Delete field
	if err := repo.DeleteField(ctx, field.ID); err != nil {
		t.Fatalf("DeleteField: %v", err)
	}
	got2, err := repo.GetByID(ctx, cat.ID)
	if err != nil {
		t.Fatalf("GetByID after field delete: %v", err)
	}
	if len(got2.Fields) != 0 {
		t.Errorf("Fields count after delete = %d, want 0", len(got2.Fields))
	}
}
```

- [ ] **Step 2: Run tests**

Run: `TEST_DATABASE_URL="$DATABASE_URL" go test ./internal/repository/ -run TestHardwareCategory -v`
Expected: PASS

Run: `TEST_DATABASE_URL="$DATABASE_URL" go test ./internal/repository/ -run TestFieldDefinition -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/repository/hardware_category_test.go
git commit -m "test: add hardware category and field definition repository tests"
```

---

### Task 10: Update Asset Repository Tests

**Files:**
- Modify: `internal/repository/asset_test.go`

- [ ] **Step 1: Update TestAssetCRUD to use category_id instead of type**

Replace the test to work with the new schema:

```go
package repository_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/xan-com/xan-argus/internal/model"
	"github.com/xan-com/xan-argus/internal/repository"
)

func TestAssetCRUD(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	// Prereqs: customer and category
	customerRepo := repository.NewCustomerRepository(pool)
	customer, err := customerRepo.Create(ctx, model.CreateCustomerInput{Name: "Asset Test Customer"})
	if err != nil {
		t.Fatalf("Create customer: %v", err)
	}
	t.Cleanup(func() { customerRepo.Delete(ctx, customer.ID) }) //nolint:errcheck

	catRepo := repository.NewHardwareCategoryRepository(pool)
	cat, err := catRepo.Create(ctx, model.CreateHardwareCategoryInput{Name: "Test Asset Category"})
	if err != nil {
		t.Fatalf("Create category: %v", err)
	}
	t.Cleanup(func() { catRepo.Delete(ctx, cat.ID) }) //nolint:errcheck

	repo := repository.NewAssetRepository(pool)

	// Create with category
	input := model.CreateAssetInput{
		CustomerID:  customer.ID,
		CategoryID:  cat.ID,
		Name:        "Test Laptop",
		FieldValues: json.RawMessage(`{"some-field": "some-value"}`),
	}
	asset, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if asset.Name != "Test Laptop" {
		t.Errorf("Name = %q, want %q", asset.Name, "Test Laptop")
	}
	if !asset.CategoryID.Valid {
		t.Error("CategoryID should be valid")
	}
	if !asset.ID.Valid {
		t.Error("ID should be valid")
	}
	if string(asset.Metadata) == "" {
		t.Error("Metadata should not be empty")
	}
	if string(asset.FieldValues) == "" {
		t.Error("FieldValues should not be empty")
	}
	t.Cleanup(func() { repo.Delete(ctx, asset.ID) }) //nolint:errcheck

	// Create without category (nullable)
	inputNoCat := model.CreateAssetInput{
		CustomerID: customer.ID,
		Name:       "Uncategorized Device",
	}
	assetNoCat, err := repo.Create(ctx, inputNoCat)
	if err != nil {
		t.Fatalf("Create without category: %v", err)
	}
	if assetNoCat.CategoryID.Valid {
		t.Error("CategoryID should be null for uncategorized asset")
	}
	t.Cleanup(func() { repo.Delete(ctx, assetNoCat.ID) }) //nolint:errcheck

	// GetByID
	got, err := repo.GetByID(ctx, asset.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "Test Laptop" {
		t.Errorf("GetByID Name = %q, want %q", got.Name, "Test Laptop")
	}

	// Update
	newName := "Updated Laptop"
	updated, err := repo.Update(ctx, asset.ID, model.UpdateAssetInput{Name: &newName})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "Updated Laptop" {
		t.Errorf("Updated Name = %q, want %q", updated.Name, "Updated Laptop")
	}

	// ListByCustomer
	assets, err := repo.ListByCustomer(ctx, customer.ID, model.ListParams{Limit: 10})
	if err != nil {
		t.Fatalf("ListByCustomer: %v", err)
	}
	if len(assets) < 2 {
		t.Errorf("ListByCustomer returned %d results, want at least 2", len(assets))
	}

	// ListByCustomer with search
	searched, err := repo.ListByCustomer(ctx, customer.ID, model.ListParams{Limit: 10, Search: "laptop"})
	if err != nil {
		t.Fatalf("ListByCustomer with search: %v", err)
	}
	if len(searched) == 0 {
		t.Error("ListByCustomer with search returned no results")
	}

	// Delete
	if err := repo.Delete(ctx, asset.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = repo.GetByID(ctx, asset.ID)
	if err == nil {
		t.Error("GetByID after Delete should return error")
	}
}
```

- [ ] **Step 2: Run tests**

Run: `TEST_DATABASE_URL="$DATABASE_URL" go test ./internal/repository/ -run TestAssetCRUD -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/repository/asset_test.go
git commit -m "test: update asset repository tests for category_id and field_values"
```

---

### Task 11: Field Values Validation Helper

**Files:**
- Create: `internal/handler/field_validation.go`
- Create: `internal/handler/field_validation_test.go`

**Note:** The `AssetHandler` struct update and handler integration were already done in Task 7. This task only creates the validation function and its unit test.

- [ ] **Step 1: Write the field values validator**

```go
package handler

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/xan-com/xan-argus/internal/model"
)

// validateFieldValues checks that field_values keys match valid field definition IDs
// for the given category, and that values match expected types.
// Returns an error message string if invalid, empty string if valid.
func validateFieldValues(rawValues json.RawMessage, fields []model.FieldDefinition) string {
	if len(rawValues) == 0 || string(rawValues) == "{}" || string(rawValues) == "null" {
		return ""
	}

	var values map[string]json.RawMessage
	if err := json.Unmarshal(rawValues, &values); err != nil {
		return "field_values must be a JSON object"
	}

	// Build lookup of valid field IDs
	fieldMap := make(map[string]model.FieldDefinition, len(fields))
	for _, f := range fields {
		idBytes, _ := f.ID.MarshalJSON()
		// MarshalJSON wraps in quotes, strip them
		idStr := string(idBytes)
		if len(idStr) >= 2 && idStr[0] == '"' {
			idStr = idStr[1 : len(idStr)-1]
		}
		fieldMap[idStr] = f
	}

	for key, rawVal := range values {
		fd, ok := fieldMap[key]
		if !ok {
			return fmt.Sprintf("unknown field_values key: %s", key)
		}

		switch fd.FieldType {
		case "text":
			var s string
			if err := json.Unmarshal(rawVal, &s); err != nil {
				return fmt.Sprintf("field %q (%s) must be a string", fd.Name, key)
			}
		case "number":
			var n float64
			if err := json.Unmarshal(rawVal, &n); err != nil {
				return fmt.Sprintf("field %q (%s) must be a number", fd.Name, key)
			}
		case "boolean":
			var b bool
			if err := json.Unmarshal(rawVal, &b); err != nil {
				return fmt.Sprintf("field %q (%s) must be a boolean", fd.Name, key)
			}
		case "date":
			var s string
			if err := json.Unmarshal(rawVal, &s); err != nil {
				return fmt.Sprintf("field %q (%s) must be a date string (YYYY-MM-DD)", fd.Name, key)
			}
			if _, err := time.Parse("2006-01-02", s); err != nil {
				return fmt.Sprintf("field %q (%s) must be in YYYY-MM-DD format", fd.Name, key)
			}
		}
	}

	return ""
}
```

- [ ] **Step 2: Write unit tests for validateFieldValues**

```go
package handler

import (
	"encoding/json"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/xan-com/xan-argus/internal/model"
)

func makeFieldDef(id string, name string, fieldType string) model.FieldDefinition {
	var uid pgtype.UUID
	uid.Scan(id)
	return model.FieldDefinition{ID: uid, Name: name, FieldType: fieldType}
}

func TestValidateFieldValues(t *testing.T) {
	fields := []model.FieldDefinition{
		makeFieldDef("00000000-0000-0000-0000-000000000001", "RAM", "number"),
		makeFieldDef("00000000-0000-0000-0000-000000000002", "Serial", "text"),
		makeFieldDef("00000000-0000-0000-0000-000000000003", "Active", "boolean"),
		makeFieldDef("00000000-0000-0000-0000-000000000004", "PurchaseDate", "date"),
	}

	tests := []struct {
		name    string
		values  string
		wantErr bool
	}{
		{"empty object", `{}`, false},
		{"null", `null`, false},
		{"valid number", `{"00000000-0000-0000-0000-000000000001": 16}`, false},
		{"valid text", `{"00000000-0000-0000-0000-000000000002": "SN-123"}`, false},
		{"valid boolean", `{"00000000-0000-0000-0000-000000000003": true}`, false},
		{"valid date", `{"00000000-0000-0000-0000-000000000004": "2026-01-15"}`, false},
		{"unknown key", `{"00000000-0000-0000-0000-999999999999": "x"}`, true},
		{"wrong type for number", `{"00000000-0000-0000-0000-000000000001": "not-a-number"}`, true},
		{"wrong type for boolean", `{"00000000-0000-0000-0000-000000000003": "yes"}`, true},
		{"bad date format", `{"00000000-0000-0000-0000-000000000004": "15/01/2026"}`, true},
		{"invalid json", `not-json`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := validateFieldValues(json.RawMessage(tt.values), fields)
			if tt.wantErr && msg == "" {
				t.Error("expected error, got empty string")
			}
			if !tt.wantErr && msg != "" {
				t.Errorf("expected no error, got %q", msg)
			}
		})
	}
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/handler/ -run TestValidateFieldValues -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/handler/field_validation.go internal/handler/field_validation_test.go
git commit -m "feat: add field_values validation helper with unit tests"
```

---

### Task 12: Update Data Model Diagram

**Files:**
- Modify: `docs/data-model.mmd`

- [ ] **Step 1: Update the Mermaid diagram**

Add `hardware_categories` and `category_field_definitions` entities. Update `assets` to remove `type` and add `category_id` and `field_values`.

In the `assets` entity, replace:
```
    assets {
        UUID id PK
        UUID customer_id FK
        enum type "hardware | software"
        string name
        text description
        jsonb metadata
        timestamp created_at
        timestamp updated_at
    }
```

With:
```
    assets {
        UUID id PK
        UUID customer_id FK
        UUID category_id FK "nullable"
        string name
        text description
        jsonb metadata
        jsonb field_values
        timestamp created_at
        timestamp updated_at
    }
```

Add new entities:
```
    hardware_categories {
        UUID id PK
        string name
        text description
        timestamp created_at
        timestamp updated_at
    }

    category_field_definitions {
        UUID id PK
        UUID category_id FK
        string name
        enum field_type "text | number | date | boolean"
        boolean required
        int sort_order
        timestamp created_at
        timestamp updated_at
    }
```

Add relationships:
```
    hardware_categories ||--o{ category_field_definitions : "defines fields"
    hardware_categories ||--o{ assets : "categorizes"
```

- [ ] **Step 2: Commit**

```bash
git add docs/data-model.mmd
git commit -m "docs: update data model diagram with hardware categories"
```

---

### Task 13: Full Integration Verification

- [ ] **Step 1: Run all tests**

Run: `TEST_DATABASE_URL="$DATABASE_URL" go test ./... -v`
Expected: All tests pass

- [ ] **Step 2: Run linter**

Run: `go vet ./...`
Expected: No errors

- [ ] **Step 3: Build and start the application**

Run: `docker compose up --build`
Expected: Application starts, migrations complete

- [ ] **Step 4: Smoke test the API**

Test category list (should include seeds):
```
GET http://localhost:8080/api/v1/hardware-categories
```
Expected: 200 with 5 seeded categories

Test create category:
```
POST http://localhost:8080/api/v1/hardware-categories
{"name": "Tablet", "description": "Tablet devices"}
```
Expected: 201

Test add field:
```
POST http://localhost:8080/api/v1/hardware-categories/{id}/fields
{"name": "Screen Size", "field_type": "number"}
```
Expected: 201

Test get category with fields:
```
GET http://localhost:8080/api/v1/hardware-categories/{id}
```
Expected: 200 with `fields` array

Test create asset with category:
```
POST http://localhost:8080/api/v1/customers/{customerId}/assets
{"name": "iPad Pro", "category_id": "{id}", "field_values": {"{field_id}": 12.9}}
```
Expected: 201

- [ ] **Step 5: Commit any fixes**

If any issues found during smoke testing, fix and commit.
