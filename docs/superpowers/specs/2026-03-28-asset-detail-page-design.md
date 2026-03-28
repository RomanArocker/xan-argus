# Asset Detail Page Design

**Date:** 2026-03-28
**Status:** Approved

## Overview

Add a read-only detail page for assets, accessible from the customer detail page. The primary goal is to display category-specific field values (from `category_field_definitions`) that are currently invisible in the UI.

## Route & Navigation

- **New route:** `GET /customers/{customerId}/assets/{assetId}` — renders an HTML page
- **Customer detail table:** Asset name becomes a clickable `<a>` link to the detail page
- **Back link:** Detail page header includes `← Back to {CustomerName}` linking to `/customers/{customerId}`

## Page Layout

### 1. Header

- `<h1>` with the asset name
- Back link to customer detail page

### 2. Base Information (Card with `<dl>`)

| Field       | Source              |
|-------------|---------------------|
| Name        | `asset.name`        |
| Category    | `category.name` or "—" if none |
| Description | `asset.description` or "—" |
| Created     | `asset.created_at`  |
| Updated     | `asset.updated_at`  |

### 3. Category Fields (Card, conditional)

Only rendered when the asset has a `category_id`. Displays `field_values` with labels from `category_field_definitions`, sorted by `sort_order`. Empty values show "—".

Field types are rendered as plain text (no special formatting for date/boolean/number in this read-only view).

## Backend

### Page Handler

Add a new method to `CustomerHandler` (since this is a sub-page of customer):

```go
func (h *CustomerHandler) assetDetail(w http.ResponseWriter, r *http.Request)
```

**Data loading:**
1. Parse `customerId` and `assetId` from path
2. Load customer via `CustomerRepository.GetByID` (for back-link name)
3. Load asset via `AssetRepository.GetByID`
4. Validate `asset.CustomerID == customerId` (404 if mismatch)
5. If `asset.CategoryID` is valid, load category via `HardwareCategoryRepository.GetByID` (provides field definitions with names and sort order)

**Route registration:**
```go
mux.HandleFunc("GET /customers/{customerId}/assets/{assetId}", h.assetDetail)
```

### Template Data

```go
type AssetDetailData struct {
    Customer model.Customer
    Asset    model.Asset
    Category *model.HardwareCategory  // nil if no category
    Fields   []AssetFieldDisplay       // sorted, label + value pairs
}

type AssetFieldDisplay struct {
    Name  string
    Value string
}
```

The handler prepares `Fields` by iterating `category.Fields` (sorted by `sort_order`), looking up each field name in `asset.FieldValues` JSON, and converting to display strings.

## Template

New file: `web/templates/asset_detail.html`

Follows existing patterns (extends base layout, uses `<dl>` for fields, semantic HTML with `<header>`, `<section>`).

```html
{{define "content"}}
<header class="page-header">
    <a href="/customers/{{uuidStr .Customer.ID}}">← Back to {{.Customer.Name}}</a>
    <h1>{{.Asset.Name}}</h1>
</header>

<div class="card mb-1">
    <dl>
        <dt>Category</dt><dd>{{if .Category}}{{.Category.Name}}{{else}}—{{end}}</dd>
        <dt>Description</dt><dd>{{pgText .Asset.Description | default "—"}}</dd>
        <dt>Created</dt><dd>{{formatDateTime .Asset.CreatedAt}}</dd>
        <dt>Updated</dt><dd>{{formatDateTime .Asset.UpdatedAt}}</dd>
    </dl>
</div>

{{if .Fields}}
<section class="section">
    <header class="page-header"><h2>Category Fields</h2></header>
    <div class="card">
        <dl>
            {{range .Fields}}
            <dt>{{.Name}}</dt><dd>{{.Value | default "—"}}</dd>
            {{end}}
        </dl>
    </div>
</section>
{{end}}
{{end}}
```

## Changes to Existing Code

### Customer Detail Template (`customer_detail.html`)

Change asset name `<td>` from plain text to a link:

```html
<!-- Before -->
<td>{{.Name}}</td>

<!-- After -->
<td><a href="/customers/{{uuidStr $.Customer.ID}}/assets/{{uuidStr .ID}}">{{.Name}}</a></td>
```

## Scope Boundaries

- **Read-only** — no edit or delete functionality on this page
- **No new API endpoints** — uses existing repositories directly
- **No new CSS** — uses existing `style.css` classes
- **Future extensions:** Edit button, inline editing, modal view can be added later
