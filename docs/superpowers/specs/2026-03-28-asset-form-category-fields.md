# Asset Form with Dynamic Category Fields

## Overview

Add an asset create/edit form to the web UI, accessible only via the customer detail page. The form includes dynamic category fields that load via HTMX when the user selects a hardware category.

## Scope

- Asset create and edit forms (name, category, description, category fields)
- Dynamic category field loading via HTMX
- Field type mapping to HTML input types
- Pre-filling existing values on edit
- No metadata (free JSONB) editing — out of scope

## Routes

### Page Routes (PageHandler)

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| `GET` | `/customers/{customerId}/assets/new` | `assetForm` | New asset form |
| `GET` | `/customers/{customerId}/assets/{assetId}/edit` | `assetEditForm` | Edit asset form |

### HTMX Partial Route

| Method | Path | Handler | Purpose |
|--------|------|---------|---------|
| `GET` | `/categories/{id}/fields` | `categoryFieldsPartial` | Returns form inputs for a category's field definitions |

This partial lives on the `PageHandler` since it renders HTML, not JSON.

## Templates

### `web/templates/assets/form.html`

Page template for create and edit. Structure:

```html
{{define "content"}}
<header class="page-header">
    <h1>{{if .IsNew}}New Asset{{else}}Edit Asset{{end}} — {{.Customer.Name}}</h1>
</header>

<div class="card">
    <form id="asset-form"
          {{if .IsNew}}
            hx-post="/api/v1/customers/{{uuidStr .Customer.ID}}/assets"
          {{else}}
            hx-put="/api/v1/assets/{{uuidStr .Asset.ID}}"
          {{end}}
          hx-ext="json-enc"
          hx-target="#form-message"
          hx-swap="innerHTML">

        <div id="form-message" aria-live="polite"></div>

        <!-- Name (required) -->
        <div class="form-group">
            <label for="name">Name *</label>
            <input type="text" id="name" name="name" value="..." required>
        </div>

        <!-- Category (optional) -->
        <div class="form-group">
            <label for="category_id">Category</label>
            <select id="category_id" name="category_id"
                    hx-get="..."
                    hx-target="#category-fields"
                    hx-swap="innerHTML"
                    hx-trigger="change">
                <option value="">— No category —</option>
                {{range .Categories}}
                <option value="{{uuidStr .ID}}">{{.Name}}</option>
                {{end}}
            </select>
        </div>

        <!-- Dynamic category fields container -->
        <div id="category-fields">
            {{if .CategoryFields}}
                {{template "category_fields" .CategoryFields}}
            {{end}}
        </div>

        <!-- Description (optional) -->
        <div class="form-group">
            <label for="description">Description</label>
            <textarea id="description" name="description">...</textarea>
        </div>

        <div class="flex gap-1">
            <button type="submit" class="btn btn-primary">
                {{if .IsNew}}Create{{else}}Save{{end}}
            </button>
            <a href="/customers/{{uuidStr .Customer.ID}}" class="btn">Cancel</a>
        </div>
    </form>
</div>

<script>
    // Build field_values from category field inputs before submit
    document.getElementById('asset-form').addEventListener('htmx:configRequest', function(evt) {
        var fieldInputs = document.querySelectorAll('[data-field-id]');
        if (fieldInputs.length > 0) {
            var fieldValues = {};
            fieldInputs.forEach(function(input) {
                var id = input.dataset.fieldId;
                var type = input.dataset.fieldType;
                if (type === 'boolean') {
                    fieldValues[id] = input.checked;
                } else if (type === 'number') {
                    fieldValues[id] = input.value ? parseFloat(input.value) : null;
                } else {
                    fieldValues[id] = input.value || null;
                }
            });
            evt.detail.parameters.field_values = fieldValues;
        }
    });

    // Success: navigate to asset detail page
    document.body.addEventListener('htmx:afterRequest', function(evt) {
        if (evt.detail.elt.id === 'asset-form' && evt.detail.successful) {
            var response = JSON.parse(evt.detail.xhr.responseText);
            if (response.id) {
                var customerId = '{{uuidStr .Customer.ID}}';
                window.location.href = '/customers/' + customerId + '/assets/' + response.id;
            }
        }
    });

    // Error: show error message
    document.body.addEventListener('htmx:responseError', function(evt) {
        if (evt.detail.elt.id === 'asset-form') {
            var msg = 'Error saving asset';
            try {
                var r = JSON.parse(evt.detail.xhr.responseText);
                if (r.error) msg = r.error;
            } catch(e) {}
            document.getElementById('form-message').innerHTML =
                '<div class="alert alert-error" role="alert">' + msg + '</div>';
        }
    });
</script>
{{end}}
```

### `web/templates/assets/fields_partial.html`

Partial template rendered by the HTMX endpoint. Defines `category_fields`:

```html
{{define "category_fields"}}
{{range .}}
<div class="form-group">
    <label for="field-{{uuidStr .ID}}">{{.Name}}{{if .Required}} *{{end}}</label>
    {{if eq .FieldType "text"}}
        <input type="text" id="field-{{uuidStr .ID}}"
               data-field-id="{{uuidStr .ID}}" data-field-type="text"
               value="{{.Value}}"
               {{if .Required}}required{{end}}>
    {{else if eq .FieldType "number"}}
        <input type="number" id="field-{{uuidStr .ID}}"
               data-field-id="{{uuidStr .ID}}" data-field-type="number"
               value="{{.Value}}" step="any"
               {{if .Required}}required{{end}}>
    {{else if eq .FieldType "date"}}
        <input type="date" id="field-{{uuidStr .ID}}"
               data-field-id="{{uuidStr .ID}}" data-field-type="date"
               value="{{.Value}}"
               {{if .Required}}required{{end}}>
    {{else if eq .FieldType "boolean"}}
        <input type="checkbox" id="field-{{uuidStr .ID}}"
               data-field-id="{{uuidStr .ID}}" data-field-type="boolean"
               {{if eq .Value "true"}}checked{{end}}>
    {{end}}
</div>
{{end}}
{{end}}
```

## Data Flow

### New Asset

1. `GET /customers/{customerId}/assets/new` → renders form with empty fields, all categories in dropdown
2. User selects category → `GET /categories/{categoryId}/fields` → server returns `category_fields` partial with empty values
3. User fills in form + category fields → submit
4. `htmx:configRequest` event: JavaScript collects all `[data-field-id]` inputs, builds `field_values` object (keys = UUIDs), injects into request parameters
5. `POST /api/v1/customers/{customerId}/assets` → existing API handler validates and creates
6. Success → redirect to `/customers/{customerId}/assets/{assetId}`

### Edit Asset

1. `GET /customers/{customerId}/assets/{assetId}/edit` → renders form with existing values
2. If asset has a category: page handler pre-resolves field definitions and merges with existing `field_values` to produce pre-filled `CategoryFieldData` list
3. Category fields render pre-filled inside `#category-fields` on page load
4. User can change category → HTMX reloads fields (existing values for the new category are lost, which is expected)
5. Submit → `PUT /api/v1/assets/{assetId}` → existing API handler validates and updates

### Category Fields Partial

The HTMX endpoint `GET /categories/{id}/fields` accepts an optional `?asset_id=` query parameter. When provided (edit mode), the handler looks up the asset's existing `field_values` and pre-fills the field inputs. When absent (new mode), fields render empty.

## Model Addition

A new display struct for template rendering (in handler, not model package):

```go
// CategoryFieldData pairs a field definition with its current value for form rendering.
type CategoryFieldData struct {
    model.FieldDefinition
    Value string // pre-filled value or empty string
}
```

## Handler Changes

### PageHandler additions

```go
func (h *PageHandler) assetForm(w http.ResponseWriter, r *http.Request)       // new asset form
func (h *PageHandler) assetEditForm(w http.ResponseWriter, r *http.Request)   // edit asset form
func (h *PageHandler) categoryFieldsPartial(w http.ResponseWriter, r *http.Request) // HTMX partial
```

Route registrations:

```go
mux.HandleFunc("GET /customers/{customerId}/assets/new", h.assetForm)
mux.HandleFunc("GET /customers/{customerId}/assets/{assetId}/edit", h.assetEditForm)
mux.HandleFunc("GET /categories/{id}/fields", h.categoryFieldsPartial)
```

### assetForm handler

1. Parse `customerId` from path
2. Load customer (404 if not found)
3. Load all categories via `h.hardwareCategoryRepo.List()`
4. Render `assets/form` with `Customer`, `Categories`, `Asset: model.Asset{}`, `IsNew: true`

### assetEditForm handler

1. Parse `customerId` and `assetId` from path
2. Load customer and asset (404 if not found)
3. Load all categories
4. If asset has a category: load field definitions, merge with `field_values` to produce `[]CategoryFieldData`
5. Render `assets/form` with `Customer`, `Categories`, `Asset`, `CategoryFields`, `IsNew: false`

### categoryFieldsPartial handler

1. Parse category `id` from path
2. Load category with fields via `GetByID`
3. If `?asset_id=` is present: load asset, unmarshal `field_values`, merge with field definitions
4. Build `[]CategoryFieldData` with pre-filled values (or empty)
5. Render partial `category_fields`

## Template Registration

Add to `NewTemplateEngine()` in `template.go`:
- Page: `web/templates/assets/form.html`
- Partial: `web/templates/assets/fields_partial.html`

## Customer Detail Page

Add a "+ New Asset" link/button on the customer detail page that navigates to `/customers/{customerId}/assets/new`.

## Edge Cases

- **No category selected**: No category fields shown, `field_values` submitted as `{}` or omitted
- **Category changed during edit**: Old field values are discarded, new empty fields appear — this is expected behavior since field definitions differ between categories
- **Category with no field definitions**: Empty `#category-fields` container, no extra inputs shown
- **Required fields**: HTML `required` attribute enforces client-side; server-side validation not yet implemented for required category fields (future enhancement)
- **Category deleted after asset creation**: Edit form shows no category fields, dropdown shows no selection
