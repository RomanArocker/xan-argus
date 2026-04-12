# Asset Bill of Materials (BOM) — Design Spec

**Date:** 2026-04-12  
**Status:** Approved

## Overview

Assets in XAN-Argus represent customer devices (hardware). Each asset can have a Bill of Materials (BOM) — a list of individual component positions (e.g., CPU, RAM, GPU) that make up the asset. These positions may be billed individually to customers. The BOM is designed to be extended later with a product/article catalog as the source for position names.

## Data Model

### `units` table

A seeded lookup table for units of measure. Not editable via UI — extended via migrations.

```sql
CREATE TABLE units (
    id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE
);
```

Seed data: `Stück`, `Stunden`, `Lizenz`, `Monat`, `Jahr`, `kg`, `m`, `m²`

No `deleted_at`, no audit trigger — pure reference data.

### `asset_bom_items` table

```sql
CREATE TABLE asset_bom_items (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id   UUID         NOT NULL REFERENCES assets(id) ON DELETE RESTRICT,
    name       TEXT         NOT NULL,
    quantity   NUMERIC(12,4) NOT NULL,
    unit_id    UUID         NOT NULL REFERENCES units(id) ON DELETE RESTRICT,
    unit_price NUMERIC(12,4) NOT NULL,
    currency   TEXT         NOT NULL CHECK (currency IN ('CHF', 'EUR', 'USD', 'GBP', 'JPY', 'CAD', 'AUD')),
    notes      TEXT,
    sort_order INT          NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);
```

- Soft delete + audit trigger (consistent with all other tables)
- UNIQUE partial index: `(asset_id, name) WHERE deleted_at IS NULL` — prevents duplicate component names per asset
- `sort_order` controls display order; ▲/▼ buttons swap adjacent positions
- `currency`: ISO 4217 code, validated via CHECK constraint; seeded currencies cover typical Swiss IT business use
- `name`: free text for now — future migration will add optional `product_id UUID REFERENCES products(id)` when the product catalog is built

### Future Extension

When the product catalog is implemented, `asset_bom_items` gains:
```sql
ALTER TABLE asset_bom_items ADD COLUMN product_id UUID REFERENCES products(id);
```
`name` remains for display/override; `product_id` becomes the canonical reference.

## Go Layer

### Models (`internal/model/bom.go`)

```go
type BOMItem struct {
    ID        pgtype.UUID
    AssetID   pgtype.UUID
    Name      string
    Quantity  float64
    UnitID    pgtype.UUID
    UnitName  string      // joined from units
    UnitPrice float64
    Currency  string
    Notes     pgtype.Text
    SortOrder int
    CreatedAt time.Time
    UpdatedAt time.Time
}

type CreateBOMItemInput struct {
    Name      string      `json:"name"`
    Quantity  float64     `json:"quantity"`
    UnitID    pgtype.UUID `json:"unit_id"`
    UnitPrice float64     `json:"unit_price"`
    Currency  string      `json:"currency"`
    Notes     *string     `json:"notes,omitempty"`
    SortOrder *int        `json:"sort_order,omitempty"`
}

type UpdateBOMItemInput struct {
    Name      *string      `json:"name,omitempty"`
    Quantity  *float64     `json:"quantity,omitempty"`
    UnitID    *pgtype.UUID `json:"unit_id,omitempty"`
    UnitPrice *float64     `json:"unit_price,omitempty"`
    Currency  *string      `json:"currency,omitempty"`
    Notes     *string      `json:"notes,omitempty"`
    SortOrder *int         `json:"sort_order,omitempty"`
}

type ReorderBOMInput struct {
    IDs []pgtype.UUID `json:"ids"`
}

type Unit struct {
    ID   pgtype.UUID
    Name string
}
```

### Repository (`internal/repository/bom.go`)

Methods:
- `ListByAsset(ctx, assetID) ([]BOMItem, error)` — ordered by `sort_order ASC, created_at ASC`; joins `units.name`
- `Create(ctx, assetID, input) (BOMItem, error)`
- `GetByID(ctx, id) (BOMItem, error)`
- `Update(ctx, id, input) (BOMItem, error)` — COALESCE for all fields except `notes` (clearable via direct assignment)
- `Delete(ctx, id) error` — soft delete
- `SwapSortOrder(ctx, idA, idB pgtype.UUID) error` — swaps `sort_order` values of two items in a transaction
- `ListUnits(ctx) ([]Unit, error)` — for populating the unit dropdown

### Handlers

**`BOMHandler`** (`internal/handler/bom.go`) — API routes:

```
GET    /api/v1/assets/{assetId}/bom           → list
POST   /api/v1/assets/{assetId}/bom           → create
GET    /api/v1/assets/{assetId}/bom/{id}      → get
PUT    /api/v1/assets/{assetId}/bom/{id}      → update
DELETE /api/v1/assets/{assetId}/bom/{id}      → delete
PUT    /api/v1/assets/{assetId}/bom/{id}/up   → move up (swap with predecessor)
PUT    /api/v1/assets/{assetId}/bom/{id}/down → move down (swap with successor)
```

**`PageHandler`** additions (`internal/handler/page.go`):

```
GET /assets/{assetId}/bom          → bomList (full page)
GET /assets/{assetId}/bom/rows     → bomListRows (HTMX partial)
GET /assets/{assetId}/bom/new      → bomForm (create)
GET /assets/{assetId}/bom/{id}/edit → bomEditForm (edit)
```

## UI

### BOM Page (`/assets/{assetId}/bom`)

- Page header: asset name as back-link to `/assets/{assetId}`, "+ New Position" button
- Table columns: Order (▲▼) | Name | Quantity | Unit | Unit Price | Currency | Notes | Actions (Edit, Delete)
- **Totals row** at the bottom: grouped by currency (e.g., `Total CHF: 749.00 | Total EUR: 120.00`)
- ▲/▼ buttons call `PUT /api/v1/assets/{assetId}/bom/{id}/up` or `/down` — HTMX reloads `#bom-rows` partial
- Delete uses `hx-delete` with `hx-confirm`

### Asset Detail Page

Add a link/button section: `Bill of Materials (N items)` → `/assets/{assetId}/bom`  
N = count of active BOM items, shown inline.

### BOM Form (`/assets/{assetId}/bom/new`, `/assets/{assetId}/bom/{id}/edit`)

Fields:
- Name (text, required)
- Quantity (number, required, step=0.0001)
- Unit (select from `units` table, required)
- Unit Price (number, required, step=0.0001)
- Currency (select: CHF, EUR, USD, GBP, JPY, CAD, AUD — default CHF)
- Notes (textarea, optional)

Uses custom `fetch()` JS (not `hx-ext="json-enc"`) to handle `null` for optional notes field — consistent with existing asset/license form patterns.

On success: redirect to `/assets/{assetId}/bom`.

## Templates

```
web/templates/bom/
├── list.html       # Full BOM page
├── list_rows.html  # Partial: <tr> rows (defines "bom_rows" template)
└── form.html       # Create/edit form
```

## Error Handling

- FK violation on delete (future: product catalog reference) → 409 Conflict
- Unique violation (duplicate name per asset) → 409 Conflict
- Asset not found → 404
- Invalid currency code → 400 (caught by DB CHECK constraint, surfaced as 500 unless pre-validated in handler)
- Currency pre-validation in handler: check against allowed list before DB call → 400 Bad Request

## Migration

New migration file: `010_asset_bom.sql`

Up:
1. Create `units` table
2. Seed units
3. Create `asset_bom_items` table
4. Add `updated_at` trigger on `asset_bom_items`
5. Add audit trigger on `asset_bom_items`
6. Add soft-delete guard trigger on `assets` (check for active BOM items before soft-deleting asset)
7. Add partial unique index `(asset_id, name) WHERE deleted_at IS NULL`

Down: drop in reverse order.

## Out of Scope

- Product catalog integration (future migration)
- BOM export to PDF/CSV (future feature)
- Price history / versioning
- BOM duplication when cloning an asset
