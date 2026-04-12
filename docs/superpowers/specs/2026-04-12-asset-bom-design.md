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

No timestamp columns (`created_at`, `updated_at`, `deleted_at`), no triggers, no audit — pure append-only reference data. Units are never deleted; the `ON DELETE RESTRICT` FK on `asset_bom_items.unit_id` is a schema safety net only.

Seed data (English per project language convention): `Piece`, `Hour`, `License`, `Month`, `Year`, `kg`, `m`, `m²`

### `asset_bom_items` table

```sql
CREATE TABLE asset_bom_items (
    id         UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id   UUID          NOT NULL REFERENCES assets(id) ON DELETE RESTRICT,
    name       TEXT          NOT NULL,
    quantity   NUMERIC(12,4) NOT NULL,
    unit_id    UUID          NOT NULL REFERENCES units(id) ON DELETE RESTRICT,
    unit_price NUMERIC(12,4) NOT NULL,
    currency   TEXT          NOT NULL CHECK (currency IN ('CHF', 'EUR', 'USD', 'GBP', 'JPY', 'CAD', 'AUD')),
    notes      TEXT,
    sort_order INT           NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ   NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_asset_bom_items_name_active
    ON asset_bom_items(asset_id, name) WHERE deleted_at IS NULL;
```

- Soft delete + audit trigger (consistent with all other tables)
- `sort_order` controls display order; ▲/▼ buttons swap adjacent positions
- `currency`: ISO 4217 code, validated via CHECK constraint; seeded currencies cover typical Swiss IT business use
- `name`: free text for now — future migration will add optional `product_id UUID REFERENCES products(id)` when the product catalog is built
- `quantity` and `unit_price` use `NUMERIC(12,4)` — exact decimal arithmetic for financial values

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
    ID        pgtype.UUID    `json:"id"`
    AssetID   pgtype.UUID    `json:"asset_id"`
    Name      string         `json:"name"`
    Quantity  pgtype.Numeric `json:"quantity"`
    UnitID    pgtype.UUID    `json:"unit_id"`
    UnitName  string         `json:"unit_name"`   // joined from units
    UnitPrice pgtype.Numeric `json:"unit_price"`
    Currency  string         `json:"currency"`
    Notes     pgtype.Text    `json:"notes"`
    SortOrder int            `json:"sort_order"`
    CreatedAt time.Time      `json:"created_at"`
    UpdatedAt time.Time      `json:"updated_at"`
}

type CreateBOMItemInput struct {
    Name      string         `json:"name"`
    Quantity  pgtype.Numeric `json:"quantity"`
    UnitID    pgtype.UUID    `json:"unit_id"`
    UnitPrice pgtype.Numeric `json:"unit_price"`
    Currency  string         `json:"currency"`
    Notes     *string        `json:"notes,omitempty"`
    SortOrder *int           `json:"sort_order,omitempty"`
}

// UpdateBOMItemInput uses *pgtype.Numeric for quantity/unit_price consistent
// with *pgtype.UUID usage elsewhere: nil pointer = "not provided, keep existing".
type UpdateBOMItemInput struct {
    Name      *string         `json:"name,omitempty"`
    Quantity  *pgtype.Numeric `json:"quantity,omitempty"`
    UnitID    *pgtype.UUID    `json:"unit_id,omitempty"`
    UnitPrice *pgtype.Numeric `json:"unit_price,omitempty"`
    Currency  *string         `json:"currency,omitempty"`
    Notes     *string         `json:"notes,omitempty"`
    SortOrder *int            `json:"sort_order,omitempty"`
}

type Unit struct {
    ID   pgtype.UUID `json:"id"`
    Name string      `json:"name"`
}

// BOMTotals holds the summed line totals for one currency.
type BOMTotals struct {
    Currency string         `json:"currency"`
    Total    pgtype.Numeric `json:"total"`
}
```

`BOMTotals` is produced by a repository aggregate query and passed to the template alongside the item list for the totals row.

### Repository (`internal/repository/bom.go`)

Methods:
- `ListByAsset(ctx, assetID) ([]BOMItem, error)` — ordered by `sort_order ASC, created_at ASC`; joins `units.name`
- `CountByAsset(ctx, assetID) (int, error)` — count of active BOM items; used by `assetDetail` handler
- `TotalsByAsset(ctx, assetID) ([]BOMTotals, error)` — `SELECT currency, SUM(quantity * unit_price) … GROUP BY currency ORDER BY currency`
- `Create(ctx, assetID, input) (BOMItem, error)`
- `GetByID(ctx, id) (BOMItem, error)`
- `Update(ctx, id, input) (BOMItem, error)` — COALESCE for all fields except `notes` (clearable via direct assignment)
- `Delete(ctx, id) error` — soft delete; returns "not found" when `RowsAffected() == 0`
- `SwapSortOrder(ctx, assetID, id pgtype.UUID, direction string) error` — swaps `sort_order` of the item with its neighbour in a transaction; fetches the item and its predecessor/successor by `sort_order` within the same `asset_id`; returns "not found" if the item doesn't exist under that `asset_id`; no-op (returns nil) if already at boundary (first item moved up, or last item moved down)
- `ListUnits(ctx) ([]Unit, error)` — for populating the unit dropdown

**`unit_id` validation:** Relies entirely on the FK constraint (`ON DELETE RESTRICT`). An invalid `unit_id` raises a FK violation caught by `isFKViolation()` in the handler → 409 Conflict. No pre-validation in the handler (consistent with how `category_id` is handled for assets).

### Handlers

**`BOMHandler`** (`internal/handler/bom.go`) — API routes:

The BOM API uses the flat `/api/v1/assets/{assetId}/bom/...` prefix, consistent with existing asset API routes (`GET /api/v1/assets/{id}`). The `assetId` is validated by the DB (asset must exist via FK on `asset_bom_items.asset_id`). The handler returns 404 on `pgx.ErrNoRows` for item lookups.

```
GET    /api/v1/assets/{assetId}/bom           → list
POST   /api/v1/assets/{assetId}/bom           → create
GET    /api/v1/assets/{assetId}/bom/{id}      → get
PUT    /api/v1/assets/{assetId}/bom/{id}      → update
DELETE /api/v1/assets/{assetId}/bom/{id}      → delete
PUT    /api/v1/assets/{assetId}/bom/{id}/up   → move up (no-op at boundary → 200)
PUT    /api/v1/assets/{assetId}/bom/{id}/down → move down (no-op at boundary → 200)
```

Both `/up` and `/down` respond with 200 and reload the rows partial via HTMX. At boundary (first/last), the handler returns 200 with no change.

**`PageHandler`** additions (`internal/handler/page.go`):

Existing asset page routes use prefix `/customers/{customerId}/assets/{assetId}/...`. BOM page routes follow the same convention:

```
GET /customers/{customerId}/assets/{assetId}/bom           → bomList
GET /customers/{customerId}/assets/{assetId}/bom/rows      → bomListRows
GET /customers/{customerId}/assets/{assetId}/bom/new       → bomForm
GET /customers/{customerId}/assets/{assetId}/bom/{id}/edit → bomEditForm
```

**`main.go` wiring:**
- Instantiate `BOMRepository(pool)`
- Instantiate `BOMHandler(bomRepo)` and call `RegisterRoutes(mux)`
- Pass `bomRepo` to `NewPageHandler` (new parameter for BOM page routes and `assetDetail` BOM count)

## UI

### BOM Page (`/customers/{customerId}/assets/{assetId}/bom`)

- Page header: asset name as back-link to `/customers/{customerId}/assets/{assetId}`, "+ New Position" button
- Table columns: Order (▲▼) | Name | Quantity | Unit | Unit Price | Currency | Notes | Actions (Edit, Delete)
- **Totals row** at the bottom: grouped by currency, computed by `TotalsByAsset` (e.g., `Total CHF: 749.00 | Total EUR: 120.00`)
- ▲/▼ buttons: `PUT /api/v1/assets/{assetId}/bom/{id}/up` or `/down` via HTMX — on success reloads `#bom-rows` partial; boundary items show greyed-out button (CSS `disabled` attribute)
- Delete uses `hx-delete` with `hx-confirm`

### Asset Detail Page

Add a "Bill of Materials (N items)" link → `/customers/{customerId}/assets/{assetId}/bom`.  
`N` is provided by `BOMRepository.CountByAsset` called from `assetDetail` handler.

### BOM Form

Fields:
- Name (text, required)
- Quantity (number, required, step=0.0001)
- Unit (select from `units` table, required)
- Unit Price (number, required, step=0.0001)
- Currency (select: CHF, EUR, USD, GBP, JPY, CAD, AUD — default CHF)
- Notes (textarea, optional)

Uses custom `fetch()` JS (not `hx-ext="json-enc"`) to handle `null` for optional notes field — consistent with existing asset/license form patterns.

On success: redirect to `/customers/{customerId}/assets/{assetId}/bom`.

## Templates

```
web/templates/bom/
├── list.html       # Full BOM page (key: "bom/list")
├── list_rows.html  # Partial: <tr> rows (defines template "bom_rows")
└── form.html       # Create/edit form (key: "bom/form")
```

## Error Handling

- BOM item not found (GET, PUT, DELETE) → 404
- Asset not found (FK violation on create) → 409 Conflict
- `unit_id` invalid (FK violation on create/update) → 409 Conflict
- Unique violation (duplicate name per asset) → 409 Conflict
- Invalid currency code → 400 (pre-validated in handler against allowed list before DB call)
- Move up/down at boundary → 200 no-op

## Migration

New migration file: `010_asset_bom.sql`

**Up (in order):**
1. Create `units` table (no triggers, no timestamps)
2. Seed units (`INSERT … ON CONFLICT DO NOTHING`)
3. Create `asset_bom_items` table
4. Add `updated_at` trigger on `asset_bom_items`
5. Add audit trigger on `asset_bom_items`
6. Add soft-delete guard trigger on `assets` (check for active BOM items before soft-deleting asset; raises `ERRCODE '23503'`)
7. Add partial unique index `idx_asset_bom_items_name_active`

**Down (in reverse order):**
1. Drop partial unique index `idx_asset_bom_items_name_active`
2. Drop soft-delete guard trigger and function on `assets`
3. Drop audit trigger on `asset_bom_items`
4. Drop `updated_at` trigger on `asset_bom_items`
5. `DROP TABLE IF EXISTS asset_bom_items`
6. `DROP TABLE IF EXISTS units`

## Out of Scope

- Product catalog integration (future migration)
- BOM export to PDF/CSV (future feature)
- Price history / versioning
- BOM duplication when cloning an asset
