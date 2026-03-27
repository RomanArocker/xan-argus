# Web UI Hardware Categories Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Update the HTMX web UI to support hardware categories: category management pages, and updated asset display/creation on customer detail.

**Architecture:** Follow existing Go template + HTMX patterns. New templates under `web/templates/categories/`, page handler routes in `page.go`, template engine registration in `template.go`. All data comes from existing API endpoints — no backend changes needed.

**Tech Stack:** Go HTML templates, HTMX, existing CSS classes

**Spec:** `docs/superpowers/specs/2026-03-27-asset-hardware-categories-design.md`

---

### Task 1: Add Categories Nav Link

**Files:**
- Modify: `web/templates/layout.html`

- [ ] **Step 1: Add "Categories" link to navigation**

In `layout.html`, after the existing nav links (Customers, Users, Services), add:

```html
            <a href="/categories">Categories</a>
```

The full nav section becomes:
```html
    <nav>
        <div class="container">
            <a href="/" class="brand">XAN-Argus</a>
            <a href="/customers">Customers</a>
            <a href="/users">Users</a>
            <a href="/services">Services</a>
            <a href="/categories">Categories</a>
        </div>
    </nav>
```

- [ ] **Step 2: Commit**

```bash
git add web/templates/layout.html
git commit -m "feat: add Categories link to navigation"
```

---

### Task 2: Hardware Categories List Page

**Files:**
- Create: `web/templates/categories/list.html`
- Create: `web/templates/categories/list_rows.html`

- [ ] **Step 1: Create the list page template**

Create `web/templates/categories/list.html`:

```html
{{define "content"}}
<div class="page-header">
    <h1>Hardware Categories</h1>
    <a href="/categories/new" class="btn btn-primary">+ New Category</a>
</div>
<div class="card">
    <table>
        <thead><tr><th>Name</th><th>Description</th><th>Fields</th><th>Created</th><th></th></tr></thead>
        <tbody id="category-rows">
            {{template "category_rows" .Categories}}
        </tbody>
    </table>
</div>
{{end}}
```

- [ ] **Step 2: Create the list rows partial**

Create `web/templates/categories/list_rows.html`:

```html
{{define "category_rows"}}
{{range .}}
<tr>
    <td><a href="/categories/{{uuidStr .ID}}">{{.Name}}</a></td>
    <td>{{pgText .Description}}</td>
    <td>{{len .Fields}}</td>
    <td>{{formatDateTime .CreatedAt}}</td>
    <td class="text-right">
        <a href="/categories/{{uuidStr .ID}}/edit" class="btn btn-primary btn-sm">Edit</a>
        <button class="btn btn-danger btn-sm"
                hx-delete="/api/v1/hardware-categories/{{uuidStr .ID}}"
                hx-confirm="Delete category '{{.Name}}'? Assets using it will become uncategorized."
                hx-target="closest tr" hx-swap="outerHTML swap:0.3s">Delete</button>
    </td>
</tr>
{{else}}<tr><td colspan="5" class="text-muted">No categories defined.</td></tr>{{end}}
{{end}}
```

- [ ] **Step 3: Commit**

```bash
git add web/templates/categories/list.html web/templates/categories/list_rows.html
git commit -m "feat: add hardware categories list page template"
```

---

### Task 3: Hardware Categories Form Page (Create/Edit with Field Definitions)

**Files:**
- Create: `web/templates/categories/form.html`

- [ ] **Step 1: Create the form template**

This page handles both create and edit. For edit, it also displays existing field definitions with inline add/delete.

Create `web/templates/categories/form.html`:

```html
{{define "content"}}
<div class="page-header">
    <h1>{{if .IsNew}}New Category{{else}}Edit Category: {{.Category.Name}}{{end}}</h1>
</div>
<div class="card">
    <form id="category-form"
          {{if .IsNew}}hx-post="/api/v1/hardware-categories"{{else}}hx-put="/api/v1/hardware-categories/{{uuidStr .Category.ID}}"{{end}}
          hx-ext="json-enc" hx-target="#form-message" hx-swap="innerHTML">
        <div id="form-message"></div>
        <div class="form-group">
            <label for="name">Name *</label>
            <input type="text" id="name" name="name" value="{{if not .IsNew}}{{.Category.Name}}{{end}}" required>
        </div>
        <div class="form-group">
            <label for="description">Description</label>
            <textarea id="description" name="description">{{if not .IsNew}}{{pgText .Category.Description}}{{end}}</textarea>
        </div>
        <div class="flex gap-1">
            <button type="submit" class="btn btn-primary">{{if .IsNew}}Create{{else}}Save{{end}}</button>
            <a href="/categories" class="btn">Cancel</a>
        </div>
    </form>
</div>

{{if not .IsNew}}
<div class="section mt-1">
    <div class="page-header"><h2>Field Definitions</h2></div>
    <div class="card">
        <table>
            <thead><tr><th>Name</th><th>Type</th><th>Sort Order</th><th></th></tr></thead>
            <tbody id="field-rows">
                {{range .Category.Fields}}
                <tr>
                    <td>{{.Name}}</td>
                    <td>{{.FieldType}}</td>
                    <td>{{.SortOrder}}</td>
                    <td class="text-right">
                        <button class="btn btn-danger btn-sm"
                                hx-delete="/api/v1/hardware-categories/{{uuidStr .CategoryID}}/fields/{{uuidStr .ID}}"
                                hx-confirm="Delete field '{{.Name}}'? Values in existing assets will be removed."
                                hx-target="closest tr" hx-swap="outerHTML swap:0.3s">Delete</button>
                    </td>
                </tr>
                {{else}}<tr><td colspan="4" class="text-muted">No fields defined yet.</td></tr>{{end}}
            </tbody>
        </table>
    </div>
    <details class="mt-1">
        <summary class="btn btn-sm btn-primary">+ Add Field</summary>
        <form class="card mt-1" id="add-field-form"
              hx-post="/api/v1/hardware-categories/{{uuidStr .Category.ID}}/fields"
              hx-ext="json-enc" hx-target="#form-msg-field" hx-swap="innerHTML">
            <div id="form-msg-field"></div>
            <div class="form-group">
                <label>Name *</label>
                <input type="text" name="name" required>
            </div>
            <div class="form-group">
                <label>Type *</label>
                <select name="field_type" required>
                    <option value="">Please select</option>
                    <option value="text">Text</option>
                    <option value="number">Number</option>
                    <option value="date">Date</option>
                    <option value="boolean">Boolean</option>
                </select>
            </div>
            <div class="form-group">
                <label>Sort Order</label>
                <input type="number" name="sort_order" value="0">
            </div>
            <button type="submit" class="btn btn-primary btn-sm">Add Field</button>
        </form>
    </details>
</div>
{{end}}

<script>
    document.body.addEventListener('htmx:afterRequest', function(evt) {
        if (evt.detail.elt.id === 'category-form' && evt.detail.successful) {
            var response = JSON.parse(evt.detail.xhr.responseText);
            if (response.id) {
                window.location.href = '/categories/' + response.id + '/edit';
            }
        }
        if (evt.detail.elt.id === 'add-field-form' && evt.detail.successful) {
            window.location.reload();
        }
    });
    document.body.addEventListener('htmx:responseError', function(evt) {
        if (evt.detail.elt.id === 'category-form') {
            var msg = 'Error saving';
            try { msg = JSON.parse(evt.detail.xhr.responseText).error || msg; } catch(e) {}
            document.getElementById('form-message').innerHTML = '<div class="alert alert-error">' + msg + '</div>';
        }
        if (evt.detail.elt.id === 'add-field-form') {
            var msg = 'Error saving';
            try { msg = JSON.parse(evt.detail.xhr.responseText).error || msg; } catch(e) {}
            document.getElementById('form-msg-field').innerHTML = '<div class="alert alert-error">' + msg + '</div>';
        }
    });
</script>
{{end}}
```

- [ ] **Step 2: Commit**

```bash
git add web/templates/categories/form.html
git commit -m "feat: add hardware categories form template with field definitions"
```

---

### Task 4: Update Customer Detail — Assets Section

**Files:**
- Modify: `web/templates/customers/detail.html`

- [ ] **Step 1: Replace "Type" column with "Category" in assets table**

In the Assets section, change the table header from:
```html
            <thead><tr><th>Name</th><th>Type</th><th>Description</th><th>Created</th><th></th></tr></thead>
```
To:
```html
            <thead><tr><th>Name</th><th>Category</th><th>Description</th><th>Created</th><th></th></tr></thead>
```

Change the row rendering from:
```html
                <td>{{.Name}}</td><td>{{.Type}}</td><td>{{pgText .Description}}</td>
```
To:
```html
                <td>{{.Name}}</td><td>{{if .CategoryID.Valid}}{{.CategoryName}}{{else}}<span class="text-muted">—</span>{{end}}</td><td>{{pgText .Description}}</td>
```

**Approach:** Pass a `CategoryMap map[string]string` (uuid→name) from the handler and use a `mapGet` template helper to look up the category name.

Add to `template.go` funcMap:
```go
"mapGet": func(m map[string]string, key string) string {
    if v, ok := m[key]; ok {
        return v
    }
    return ""
},
```

The asset row becomes:
```html
                <td>{{.Name}}</td><td>{{if .CategoryID.Valid}}{{mapGet $.CategoryMap (uuidStr .CategoryID)}}{{else}}<span class="text-muted">—</span>{{end}}</td><td>{{pgText .Description}}</td>
```

- [ ] **Step 2: Replace "Type" select with "Category" dropdown in Add Asset form**

Replace the old type select:
```html
            <div class="form-group"><label>Type *</label>
                <select name="type" required><option value="hardware">Hardware</option><option value="software">Software</option></select>
            </div>
```

With category dropdown:
```html
            <div class="form-group"><label>Category</label>
                <select name="category_id">
                    <option value="">No category</option>
                    {{range $.AllCategories}}<option value="{{uuidStr .ID}}">{{.Name}}</option>{{end}}
                </select>
            </div>
```

Note: category is optional (not required), matching the spec.

- [ ] **Step 3: Commit**

```bash
git add web/templates/customers/detail.html
git commit -m "feat: update customer detail assets section for categories"
```

---

### Task 5: Register Templates in Template Engine

**Files:**
- Modify: `internal/handler/template.go`

- [ ] **Step 1: Add `mapGet` template function**

In `newFuncMap()`, add:
```go
"mapGet": func(m map[string]string, key string) string {
    if v, ok := m[key]; ok {
        return v
    }
    return ""
},
```

- [ ] **Step 2: Register category templates**

In `NewTemplateEngine`, add to `partialFiles`:
```go
filepath.Join(templateDir, "categories", "list_rows.html"),
```

Add to `pageFiles`:
```go
filepath.Join(templateDir, "categories", "list.html"),
filepath.Join(templateDir, "categories", "form.html"),
```

- [ ] **Step 3: Verify compilation**

Run: `go vet ./internal/handler/...`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add internal/handler/template.go
git commit -m "feat: register category templates and add mapGet helper"
```

---

### Task 6: Add Page Handler Routes for Categories

**Files:**
- Modify: `internal/handler/page.go`

- [ ] **Step 1: Add `hardwareCategoryRepo` to PageHandler**

Add field to struct:
```go
type PageHandler struct {
    // ... existing fields ...
    hardwareCategoryRepo *repository.HardwareCategoryRepository
}
```

Update `NewPageHandler` to accept and store the new repo. Update the constructor signature and body.

- [ ] **Step 2: Register category routes**

In `RegisterRoutes`, add:
```go
mux.HandleFunc("GET /categories", h.categoryList)
mux.HandleFunc("GET /categories/new", h.categoryForm)
mux.HandleFunc("GET /categories/{id}/edit", h.categoryEditForm)
```

- [ ] **Step 3: Implement category handlers**

```go
func (h *PageHandler) categoryList(w http.ResponseWriter, r *http.Request) {
    categories, err := h.hardwareCategoryRepo.List(r.Context())
    if err != nil {
        http.Error(w, "failed to load categories", http.StatusInternalServerError)
        return
    }
    // For each category, load fields so we can show field count
    for i, cat := range categories {
        fields, err := h.hardwareCategoryRepo.ListFields(r.Context(), cat.ID)
        if err != nil {
            http.Error(w, "failed to load fields", http.StatusInternalServerError)
            return
        }
        categories[i].Fields = fields
    }
    h.tmpl.RenderPage(w, "categories/list", map[string]any{
        "Title":      "Hardware Categories",
        "Categories": categories,
    })
}

func (h *PageHandler) categoryForm(w http.ResponseWriter, r *http.Request) {
    h.tmpl.RenderPage(w, "categories/form", map[string]any{
        "Title":    "New Category",
        "Category": model.HardwareCategory{},
        "IsNew":    true,
    })
}

func (h *PageHandler) categoryEditForm(w http.ResponseWriter, r *http.Request) {
    id, err := parseUUID(r.PathValue("id"))
    if err != nil {
        http.Error(w, "invalid ID", http.StatusBadRequest)
        return
    }
    cat, err := h.hardwareCategoryRepo.GetByID(r.Context(), id)
    if err != nil {
        http.Error(w, "category not found", http.StatusNotFound)
        return
    }
    h.tmpl.RenderPage(w, "categories/form", map[string]any{
        "Title":    "Edit Category",
        "Category": cat,
        "IsNew":    false,
    })
}
```

- [ ] **Step 4: Update `customerDetail` handler to pass categories**

In the `customerDetail` handler, after fetching existing data, add:
```go
    categories, err := h.hardwareCategoryRepo.List(r.Context())
    if err != nil {
        http.Error(w, "failed to load categories", http.StatusInternalServerError)
        return
    }
    categoryMap := make(map[string]string)
    for _, cat := range categories {
        categoryMap[uuidStr(cat.ID)] = cat.Name
    }
```

The template funcMap already has a `uuidStr` function. Extract it into a shared unexported Go function in `params.go` so both templates and handlers can use it:

```go
// uuidToStr formats a pgtype.UUID as a lowercase hex string.
// Shared by template funcMap and handler code.
func uuidToStr(u pgtype.UUID) string {
    if !u.Valid {
        return ""
    }
    b := u.Bytes
    return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
```

Then update `template.go` to use `"uuidStr": uuidToStr,` instead of the inline closure.

Use it in the handler:
```go
    categoryMap[uuidToStr(cat.ID)] = cat.Name
```

Add `CategoryMap` and `AllCategories` to the template data:
```go
    h.tmpl.RenderPage(w, "customers/detail", map[string]any{
        "Title":            customer.Name,
        "Customer":         customer,
        "Assets":           assets,
        "Licenses":         licenses,
        "Assignments":      assignments,
        "CustomerServices": customerServices,
        "Users":            users,
        "AllServices":      allServices,
        "AllCategories":    categories,
        "CategoryMap":      categoryMap,
    })
```

- [ ] **Step 5: Verify compilation**

Run: `go vet ./...`
Expected: No errors

- [ ] **Step 6: Commit**

```bash
git add internal/handler/page.go internal/handler/params.go
git commit -m "feat: add category page handlers and update customer detail with categories"
```

---

### Task 7: Wire Up in main.go

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Pass `hardwareCategoryRepo` to `NewPageHandler`**

The `NewPageHandler` call currently passes 8 repos. Add `hardwareCategoryRepo` as the 9th argument. Update the call to match the new signature.

- [ ] **Step 2: Verify compilation and startup**

Run: `go vet ./...`
Expected: No errors

Run: `docker compose up --build`
Expected: Application starts, navigate to http://localhost:8080/categories shows seeded categories

- [ ] **Step 3: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat: wire hardwareCategoryRepo into PageHandler"
```

---

### Task 8: Visual Verification

- [ ] **Step 1: Verify categories list page**

Navigate to: `http://localhost:8080/categories`
Expected: Table with 5 seeded categories (Laptop, Server, Printer, Monitor, Network Device), each showing 0 fields

- [ ] **Step 2: Verify create category**

Click "+ New Category", fill in name "Tablet", description "Tablet devices", submit.
Expected: Redirects to edit page for Tablet.

- [ ] **Step 3: Verify add field definitions**

On the Tablet edit page, expand "+ Add Field", add "Screen Size" (number), "Serial Number" (text).
Expected: Fields appear in the table after reload.

- [ ] **Step 4: Verify customer detail assets section**

Navigate to a customer detail page.
Expected: Assets table shows "Category" column instead of "Type". Add Asset form has category dropdown.

- [ ] **Step 5: Verify creating an asset with category**

On customer detail, expand "+ Add Asset", select a category, fill in name, submit.
Expected: New asset appears in table with category name.

- [ ] **Step 6: Fix any issues and commit**

```bash
git add -A
git commit -m "fix: visual verification fixes"
```
