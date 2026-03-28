# Asset Form with Dynamic Category Fields — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add asset create/edit forms to the web UI with dynamic category fields loaded via HTMX.

**Architecture:** New page handler methods serve asset forms. When the user selects a category, an HTMX partial endpoint returns the matching field inputs. JavaScript in the form collects field values (keyed by field definition UUIDs) and injects them into the JSON payload before submit. The existing asset API handlers and validation are reused unchanged.

**Tech Stack:** Go templates, HTMX (`hx-get` for dynamic fields), `json-enc` extension for form submission, existing `validateFieldValues` for server-side validation.

**Spec:** `docs/superpowers/specs/2026-03-28-asset-form-category-fields.md`

---

## File Map

| Action | File | Responsibility |
|--------|------|----------------|
| Create | `web/templates/assets/form.html` | Asset create/edit page template |
| Create | `web/templates/assets/fields_partial.html` | HTMX partial — category field inputs |
| Modify | `internal/handler/template.go` | Register new page + partial templates |
| Modify | `internal/handler/page.go` | Add `assetForm`, `assetEditForm`, `categoryFieldsPartial` handlers + routes |
| Modify | `web/templates/customers/detail.html` | Add "+ New Asset" button |

---

### Task 1: Create the category fields partial template

**Files:**
- Create: `web/templates/assets/fields_partial.html`

- [ ] **Step 1: Create the partial template**

This template renders form inputs for each field definition. Each input carries `data-field-id` (the field definition UUID) and `data-field-type` for the JavaScript collector.

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

The `.Value` field comes from a `CategoryFieldData` struct (defined in Task 3). For new assets it's `""`, for edits it's the pre-filled value from `field_values`.

- [ ] **Step 2: Commit**

```bash
git add web/templates/assets/fields_partial.html
git commit -m "feat: add category fields partial template for HTMX"
```

---

### Task 2: Create the asset form page template

**Files:**
- Create: `web/templates/assets/form.html`

- [ ] **Step 1: Create the form template**

Follow the existing form pattern (see `web/templates/services/form.html`). Key differences:
- Form ID: `asset-form`
- `hx-post` targets `/api/v1/customers/{{uuidStr .Customer.ID}}/assets` (create) or `hx-put` targets `/api/v1/assets/{{uuidStr .Asset.ID}}` (edit)
- Category `<select>` with `hx-get` to load fields dynamically
- `#category-fields` container for the HTMX partial
- JavaScript `htmx:configRequest` handler to build `field_values` from `[data-field-id]` inputs

```html
{{define "content"}}
<header class="page-header">
    <h1>{{if .IsNew}}New Asset{{else}}Edit Asset{{end}} — {{.Customer.Name}}</h1>
</header>

<div class="card">
    <form id="asset-form"
          {{if .IsNew}}hx-post="/api/v1/customers/{{uuidStr .Customer.ID}}/assets"{{else}}hx-put="/api/v1/assets/{{uuidStr .Asset.ID}}"{{end}}
          hx-ext="json-enc"
          hx-target="#form-message"
          hx-swap="innerHTML">

        <div id="form-message" aria-live="polite"></div>

        <div class="form-group">
            <label for="name">Name *</label>
            <input type="text" id="name" name="name"
                   value="{{if not .IsNew}}{{.Asset.Name}}{{end}}" required>
        </div>

        <div class="form-group">
            <label for="category_id">Category</label>
            <select id="category_id" name="category_id"
                    hx-get=""
                    hx-target="#category-fields"
                    hx-swap="innerHTML"
                    hx-trigger="change"
                    hx-include="[name='category_id']">
                <option value="">— No category —</option>
                {{range .Categories}}
                <option value="{{uuidStr .ID}}" {{if eq (uuidStr .ID) (uuidStr $.Asset.CategoryID)}}selected{{end}}>{{.Name}}</option>
                {{end}}
            </select>
        </div>

        <div id="category-fields">
            {{if .CategoryFields}}
                {{template "category_fields" .CategoryFields}}
            {{end}}
        </div>

        <div class="form-group">
            <label for="description">Description</label>
            <textarea id="description" name="description">{{if not .IsNew}}{{pgText .Asset.Description}}{{end}}</textarea>
        </div>

        <div class="flex gap-1">
            <button type="submit" class="btn btn-primary">{{if .IsNew}}Create{{else}}Save{{end}}</button>
            <a href="/customers/{{uuidStr .Customer.ID}}" class="btn">Cancel</a>
        </div>
    </form>
</div>

<script>
    // Set hx-get URL dynamically based on selected category
    var categorySelect = document.getElementById('category_id');
    function updateFieldsUrl() {
        var catId = categorySelect.value;
        if (catId) {
            var assetId = '{{if not .IsNew}}{{uuidStr .Asset.ID}}{{end}}';
            var url = '/categories/' + catId + '/fields';
            if (assetId) url += '?asset_id=' + assetId;
            categorySelect.setAttribute('hx-get', url);
        } else {
            categorySelect.setAttribute('hx-get', '');
            document.getElementById('category-fields').innerHTML = '';
        }
    }
    categorySelect.addEventListener('change', updateFieldsUrl);
    // Initialize on page load (for edit mode)
    updateFieldsUrl();

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
        // Remove empty category_id so it doesn't send as ""
        if (!categorySelect.value) {
            delete evt.detail.parameters.category_id;
        }
    });

    // Success: navigate to asset detail page
    document.body.addEventListener('htmx:afterRequest', function(evt) {
        if (evt.detail.elt.id === 'asset-form' && evt.detail.successful) {
            var response = JSON.parse(evt.detail.xhr.responseText);
            if (response.id) {
                window.location.href = '/customers/{{uuidStr .Customer.ID}}/assets/' + response.id;
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

**Important notes for the implementer:**
- The `hx-get` on the `<select>` starts empty and is set dynamically by JavaScript — this avoids HTMX firing a request to an empty URL on page load.
- `hx-include="[name='category_id']"` is NOT needed since we set the full URL dynamically. The `hx-get` attribute is updated by JavaScript before HTMX processes the change event.
- The `htmx:configRequest` event fires before the request is sent, allowing us to inject `field_values` into the JSON payload.
- `category_id` must be removed from parameters when empty to avoid sending `""` (the API expects a valid UUID or omission).

- [ ] **Step 2: Commit**

```bash
git add web/templates/assets/form.html
git commit -m "feat: add asset form template with dynamic category fields"
```

---

### Task 3: Register templates and add CategoryFieldData type

**Files:**
- Modify: `internal/handler/template.go` — add page + partial to registration lists

- [ ] **Step 1: Add `CategoryFieldData` type to `page.go`**

This struct pairs a field definition with its current value for template rendering. Add it near the existing `AssetFieldDisplay` type in `internal/handler/page.go`:

```go
// CategoryFieldData pairs a field definition with its current value for form rendering.
type CategoryFieldData struct {
	model.FieldDefinition
	Value string // pre-filled value or empty string
}
```

- [ ] **Step 2: Register the new templates in `template.go`**

In `NewTemplateEngine()`, add to `pageFiles`:
```go
filepath.Join(templateDir, "assets", "form.html"),
```

Add to `partialFiles`:
```go
filepath.Join(templateDir, "assets", "fields_partial.html"),
```

The partial defines `category_fields` which is referenced by the form page template. Since all partials are parsed into every page template, this will work automatically.

- [ ] **Step 3: Verify the project compiles**

Run: `go vet ./...`
Expected: no errors (templates are parsed at runtime, so this just checks Go code compiles)

- [ ] **Step 4: Commit**

```bash
git add internal/handler/template.go internal/handler/page.go
git commit -m "feat: register asset form templates and add CategoryFieldData type"
```

---

### Task 4: Add page handler methods and routes

**Files:**
- Modify: `internal/handler/page.go` — add 3 handler methods + 3 routes

- [ ] **Step 1: Add route registrations**

In `PageHandler.RegisterRoutes()`, add these routes (after the existing `assetDetail` route):

```go
mux.HandleFunc("GET /customers/{customerId}/assets/new", h.assetForm)
mux.HandleFunc("GET /customers/{customerId}/assets/{assetId}/edit", h.assetEditForm)
mux.HandleFunc("GET /categories/{id}/fields", h.categoryFieldsPartial)
```

**Important:** The `/assets/new` route must be registered before `/assets/{assetId}` to avoid "new" being captured as an asset ID. Check that the existing `assetDetail` route (`GET /customers/{customerId}/assets/{assetId}`) is registered after this one. Go 1.22+ ServeMux handles this correctly — literal segments like "new" take priority over wildcards.

- [ ] **Step 2: Add `assetForm` handler (new asset)**

```go
func (h *PageHandler) assetForm(w http.ResponseWriter, r *http.Request) {
	customerID, err := parseUUID(r.PathValue("customerId"))
	if err != nil {
		http.Error(w, "invalid customer ID", http.StatusBadRequest)
		return
	}
	customer, err := h.customerRepo.GetByID(r.Context(), customerID)
	if err != nil {
		http.Error(w, "customer not found", http.StatusNotFound)
		return
	}
	categories, err := h.hardwareCategoryRepo.List(r.Context())
	if err != nil {
		http.Error(w, "failed to load categories", http.StatusInternalServerError)
		return
	}
	h.tmpl.RenderPage(w, "assets/form", map[string]any{
		"Title":      "New Asset — " + customer.Name,
		"Customer":   customer,
		"Asset":      model.Asset{},
		"Categories": categories,
		"IsNew":      true,
	})
}
```

- [ ] **Step 3: Add `assetEditForm` handler (edit asset)**

```go
func (h *PageHandler) assetEditForm(w http.ResponseWriter, r *http.Request) {
	customerID, err := parseUUID(r.PathValue("customerId"))
	if err != nil {
		http.Error(w, "invalid customer ID", http.StatusBadRequest)
		return
	}
	assetID, err := parseUUID(r.PathValue("assetId"))
	if err != nil {
		http.Error(w, "invalid asset ID", http.StatusBadRequest)
		return
	}
	customer, err := h.customerRepo.GetByID(r.Context(), customerID)
	if err != nil {
		http.Error(w, "customer not found", http.StatusNotFound)
		return
	}
	asset, err := h.assetRepo.GetByID(r.Context(), assetID)
	if err != nil {
		http.Error(w, "asset not found", http.StatusNotFound)
		return
	}
	categories, err := h.hardwareCategoryRepo.List(r.Context())
	if err != nil {
		http.Error(w, "failed to load categories", http.StatusInternalServerError)
		return
	}

	var categoryFields []CategoryFieldData
	if asset.CategoryID.Valid {
		cat, err := h.hardwareCategoryRepo.GetByID(r.Context(), asset.CategoryID)
		if err == nil {
			categoryFields = buildCategoryFieldData(cat.Fields, asset.FieldValues)
		}
	}

	h.tmpl.RenderPage(w, "assets/form", map[string]any{
		"Title":          "Edit Asset — " + customer.Name,
		"Customer":       customer,
		"Asset":          asset,
		"Categories":     categories,
		"CategoryFields": categoryFields,
		"IsNew":          false,
	})
}
```

- [ ] **Step 4: Add `categoryFieldsPartial` handler (HTMX endpoint)**

```go
func (h *PageHandler) categoryFieldsPartial(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid category ID", http.StatusBadRequest)
		return
	}
	cat, err := h.hardwareCategoryRepo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "category not found", http.StatusNotFound)
		return
	}

	var fieldValues json.RawMessage
	if assetIDStr := r.URL.Query().Get("asset_id"); assetIDStr != "" {
		assetID, err := parseUUID(assetIDStr)
		if err == nil {
			asset, err := h.assetRepo.GetByID(r.Context(), assetID)
			if err == nil {
				fieldValues = asset.FieldValues
			}
		}
	}

	data := buildCategoryFieldData(cat.Fields, fieldValues)
	h.tmpl.RenderPartial(w, "category_fields", data)
}
```

- [ ] **Step 5: Add `buildCategoryFieldData` helper function**

Add this helper in `page.go` (near the existing `AssetFieldDisplay` type):

```go
// buildCategoryFieldData merges field definitions with stored values for form rendering.
func buildCategoryFieldData(fields []model.FieldDefinition, fieldValues json.RawMessage) []CategoryFieldData {
	vals := make(map[string]interface{})
	if len(fieldValues) > 0 {
		json.Unmarshal(fieldValues, &vals)
	}

	data := make([]CategoryFieldData, 0, len(fields))
	for _, fd := range fields {
		v := ""
		if raw, ok := vals[uuidToStr(fd.ID)]; ok && raw != nil {
			v = fmt.Sprintf("%v", raw)
		}
		data = append(data, CategoryFieldData{
			FieldDefinition: fd,
			Value:           v,
		})
	}
	return data
}
```

This reuses the same UUID-key lookup pattern as the existing `assetDetail` handler.

- [ ] **Step 6: Verify the project compiles**

Run: `go vet ./...`
Expected: no errors

- [ ] **Step 7: Commit**

```bash
git add internal/handler/page.go
git commit -m "feat: add asset form and category fields partial handlers"
```

---

### Task 5: Add "+ New Asset" button to customer detail page

**Files:**
- Modify: `web/templates/customers/detail.html`

- [ ] **Step 1: Add the button**

In `web/templates/customers/detail.html`, find the Assets section header:

```html
<header class="page-header"><h2>Assets</h2></header>
```

Replace with:

```html
<header class="page-header">
    <h2>Assets</h2>
    <a href="/customers/{{uuidStr .Customer.ID}}/assets/new" class="btn btn-primary btn-sm">+ New Asset</a>
</header>
```

This follows the existing page-header pattern (title + action button on the right).

- [ ] **Step 2: Commit**

```bash
git add web/templates/customers/detail.html
git commit -m "feat: add New Asset button to customer detail page"
```

---

### Task 6: Build, run, and verify end-to-end

- [ ] **Step 1: Rebuild and start the application**

```bash
docker compose up --build -d
```

Wait for the containers to start and migrations to run.

- [ ] **Step 2: Verify new asset form loads**

Open a browser and navigate to a customer detail page. Click "+ New Asset". Verify:
- Form renders with Name, Category dropdown, Description fields
- Category dropdown lists all hardware categories

- [ ] **Step 3: Verify category fields load dynamically**

Select a category (e.g., "Laptop"). Verify:
- Category-specific fields appear (e.g., RAM, CPU, Serial Number)
- Fields have correct input types (text, number, date, checkbox)
- Selecting "— No category —" clears the fields

- [ ] **Step 4: Verify asset creation**

Fill in the form:
- Name: "Test Laptop"
- Category: select one
- Fill in category fields
- Submit

Verify:
- Redirects to asset detail page
- Asset detail shows the entered values including category fields

- [ ] **Step 5: Verify asset editing**

From the customer detail page, navigate to the asset just created. Click "Edit" (if available) or go to `/customers/{id}/assets/{assetId}/edit` manually. Verify:
- Form pre-fills Name, Category, Description
- Category fields pre-fill with saved values
- Changing category loads new fields (old values disappear)
- Saving updates the asset

- [ ] **Step 6: Commit if any fixes were needed**

```bash
git add -A
git commit -m "fix: address issues found during manual testing"
```

Only commit this if fixes were applied. Skip if everything worked on first try.
