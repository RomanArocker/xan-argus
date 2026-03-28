# License User Assignment UI — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add user assignment UI to licenses — dropdown on forms, display on detail/list pages — following the existing asset pattern exactly.

**Architecture:** Pure UI work. The backend (DB column `user_assignment_id`, consistency trigger, model, repository, API handler) already fully supports license user assignment. Changes touch only templates, page handler, and template engine registration.

**Tech Stack:** Go HTML templates, HTMX, existing `style.css`

**Spec:** `docs/superpowers/specs/2026-03-28-license-user-assignment-ui-design.md`

**Task dependencies:** Tasks are strictly sequential — each task depends on the previous one. Do not reorder.

---

### Task 1: Add `buildAssignmentMap` helper to `page.go`

**Files:**
- Modify: `internal/handler/page.go` (after `buildUserAssignmentDisplayList` at line ~341)

This helper builds a `map[string]string` of assignment UUID → display name, reusable for the license table column and any future entity tables.

- [ ] **Step 1: Add the helper function**

After `buildUserAssignmentDisplayList` (line ~341), add:

```go
// buildAssignmentMap builds a map of assignment UUID → "LastName, FirstName (Role)" for template table display.
func buildAssignmentMap(assignments []model.UserAssignment, users []model.User) map[string]string {
	userMap := make(map[string]string, len(users))
	for _, u := range users {
		userMap[uuidToStr(u.ID)] = u.LastName + ", " + u.FirstName
	}
	result := make(map[string]string, len(assignments))
	for _, a := range assignments {
		name := userMap[uuidToStr(a.UserID)]
		result[uuidToStr(a.ID)] = name + " (" + a.Role + ")"
	}
	return result
}
```

- [ ] **Step 2: Verify compilation**

```bash
go vet ./...
```

- [ ] **Step 3: Commit**

```bash
git add internal/handler/page.go
git commit -m "feat: add buildAssignmentMap helper for template table display"
```

---

### Task 2: Update `customerDetail` handler — add `LicenseAssignmentMap` and `UserAssignmentDisplays`

**Files:**
- Modify: `internal/handler/page.go` — `customerDetail` method (line ~109-149)

The `customerDetail` handler already loads `assignments` and `users`. We add two new template variables using the existing data.

- [ ] **Step 1: Add the two new template variables**

In `customerDetail`, after the `categoryMap` loop (line ~135) and before `h.tmpl.RenderPage`, add:

```go
	licenseAssignmentMap := buildAssignmentMap(assignments, users)
	userAssignmentDisplays := buildUserAssignmentDisplayList(assignments, users)
```

Then update the `RenderPage` call to include both:

```go
	h.tmpl.RenderPage(w, "customers/detail", map[string]any{
		"Title":                  customer.Name,
		"Customer":               customer,
		"Assets":                 assets,
		"Licenses":               licenses,
		"Assignments":            assignments,
		"CustomerServices":       customerServices,
		"Users":                  users,
		"AllServices":            allServices,
		"AllCategories":          categories,
		"CategoryMap":            categoryMap,
		"LicenseAssignmentMap":   licenseAssignmentMap,
		"UserAssignmentDisplays": userAssignmentDisplays,
	})
```

- [ ] **Step 2: Verify compilation**

```bash
go vet ./...
```

- [ ] **Step 3: Commit**

```bash
git add internal/handler/page.go
git commit -m "feat: pass LicenseAssignmentMap and UserAssignmentDisplays to customer detail"
```

---

### Task 3: Update customer detail template — license table and add-license form

**Files:**
- Modify: `web/templates/customers/detail.html`

Two changes: (a) add "Assigned to" column in the license table with product name as link, (b) add user assignment dropdown to the "Add License" inline form.

- [ ] **Step 1: Update the license table header**

Replace:
```html
<thead><tr><th>Product</th><th>Quantity</th><th>Valid From</th><th>Valid Until</th><th></th></tr></thead>
```

With:
```html
<thead><tr><th>Product</th><th>Assigned to</th><th>Quantity</th><th>Valid From</th><th>Valid Until</th><th></th></tr></thead>
```

- [ ] **Step 2: Update the license table rows**

Replace each license row block. The product name becomes a link, and a new "Assigned to" column is added:

```html
{{range .Licenses}}
<tr>
    <td><a href="/customers/{{uuidStr $.Customer.ID}}/licenses/{{uuidStr .ID}}">{{.ProductName}}</a></td>
    <td>{{if .UserAssignmentID.Valid}}{{mapGet $.LicenseAssignmentMap (uuidStr .UserAssignmentID)}}{{else}}<span class="text-muted">—</span>{{end}}</td>
    <td>{{.Quantity}}</td>
    <td>{{formatPgDate .ValidFrom}}</td>
    <td>{{formatPgDate .ValidUntil}}</td>
    <td class="text-right"><button class="btn btn-danger btn-sm" hx-delete="/api/v1/licenses/{{uuidStr .ID}}" hx-confirm="Delete license '{{.ProductName}}'?" hx-target="closest tr" hx-swap="outerHTML swap:0.3s">Delete</button></td>
</tr>
{{else}}<tr><td colspan="6" class="text-muted">No licenses.</td></tr>{{end}}
```

Note: `colspan` changed from `5` to `6`.

- [ ] **Step 3: Add user assignment dropdown to the "Add License" form**

In the `add-license-form`, between the License Key field and the Quantity field, add:

```html
<div class="form-group"><label>Assigned to</label>
    <select name="user_assignment_id">
        <option value="">— Not assigned —</option>
        {{range .UserAssignmentDisplays}}<option value="{{uuidStr .ID}}">{{.DisplayName}}</option>{{end}}
    </select>
</div>
```

- [ ] **Step 4: Verify by building and checking the page**

```bash
docker compose up --build -d
```

Open `http://localhost:8080/customers/{id}` and verify:
- License table has "Assigned to" column
- Product names are links
- "Add License" form has the assignment dropdown

- [ ] **Step 5: Commit**

```bash
git add web/templates/customers/detail.html
git commit -m "feat: add assigned-to column and dropdown to license section on customer detail"
```

---

### Task 4: Create license detail template and page handler

**Files:**
- Create: `web/templates/licenses/detail.html`
- Modify: `internal/handler/page.go` — add `licenseDetail` method and route
- Modify: `internal/handler/template.go` — register `licenses/detail`

- [ ] **Step 1: Create the template directory**

```bash
mkdir -p web/templates/licenses
```

- [ ] **Step 2: Create `web/templates/licenses/detail.html`**

```html
{{define "content"}}
<header class="page-header">
    <div>
        <a href="/customers/{{uuidStr .Customer.ID}}">&larr; Back to {{.Customer.Name}}</a>
        <h1>{{.License.ProductName}}</h1>
    </div>
    <a href="/customers/{{uuidStr .Customer.ID}}/licenses/{{uuidStr .License.ID}}/edit" class="btn btn-primary">Edit</a>
</header>

<div class="card mb-1">
    <dl>
        <dt>Product Name</dt><dd>{{.License.ProductName}}</dd>
        <dt>License Key</dt><dd>{{pgText .License.LicenseKey | default "—"}}</dd>
        <dt>Quantity</dt><dd>{{.License.Quantity}}</dd>
        <dt>Valid From</dt><dd>{{formatPgDate .License.ValidFrom}}</dd>
        <dt>Valid Until</dt><dd>{{formatPgDate .License.ValidUntil}}</dd>
        <dt>Assigned to</dt><dd>{{if .AssignedUser}}{{.AssignedUser}}{{else}}—{{end}}</dd>
        <dt>Created</dt><dd>{{formatDateTime .License.CreatedAt}}</dd>
        <dt>Updated</dt><dd>{{formatDateTime .License.UpdatedAt}}</dd>
    </dl>
</div>
{{end}}
```

- [ ] **Step 3: Add all three license routes and `licenseDetail` handler to `page.go`**

Add all three license routes in `RegisterRoutes` (after the asset routes, line ~58). **Order matters:** `/new` must come before `/{licenseId}` so Go's ServeMux doesn't match "new" as a licenseId:

```go
mux.HandleFunc("GET /customers/{customerId}/licenses/new", h.licenseForm)
mux.HandleFunc("GET /customers/{customerId}/licenses/{licenseId}", h.licenseDetail)
mux.HandleFunc("GET /customers/{customerId}/licenses/{licenseId}/edit", h.licenseEditForm)
```

Note: `licenseForm` and `licenseEditForm` handler methods don't exist yet — they'll be added in Task 5. This will cause a compile error until Task 5 is complete. If you prefer to keep each task independently compilable, add empty stub methods:

```go
func (h *PageHandler) licenseForm(w http.ResponseWriter, r *http.Request) {}
func (h *PageHandler) licenseEditForm(w http.ResponseWriter, r *http.Request) {}
```

Add the `licenseDetail` handler method (after `assetEditForm`, at end of asset section):

```go
func (h *PageHandler) licenseDetail(w http.ResponseWriter, r *http.Request) {
	customerID, err := parseUUID(r.PathValue("customerId"))
	if err != nil {
		http.Error(w, "invalid customer ID", http.StatusBadRequest)
		return
	}
	licenseID, err := parseUUID(r.PathValue("licenseId"))
	if err != nil {
		http.Error(w, "invalid license ID", http.StatusBadRequest)
		return
	}
	customer, err := h.customerRepo.GetByID(r.Context(), customerID)
	if err != nil {
		http.Error(w, "customer not found", http.StatusNotFound)
		return
	}
	license, err := h.licenseRepo.GetByID(r.Context(), licenseID)
	if err != nil {
		http.Error(w, "license not found", http.StatusNotFound)
		return
	}

	var assignedUser string
	if license.UserAssignmentID.Valid {
		ua, err := h.userAssignmentRepo.GetByID(r.Context(), license.UserAssignmentID)
		if err == nil {
			u, err := h.userRepo.GetByID(r.Context(), ua.UserID)
			if err == nil {
				assignedUser = u.LastName + ", " + u.FirstName + " (" + ua.Role + ")"
			}
		}
	}

	h.tmpl.RenderPage(w, "licenses/detail", map[string]any{
		"Title":        license.ProductName + " — " + customer.Name,
		"Customer":     customer,
		"License":      license,
		"AssignedUser": assignedUser,
	})
}
```

- [ ] **Step 4: Register the template in `template.go`**

Add to the `pageFiles` slice (after the `assets/form.html` entry, line ~84):

```go
filepath.Join(templateDir, "licenses", "detail.html"),
```

- [ ] **Step 5: Verify compilation and test**

```bash
go vet ./...
docker compose up --build -d
```

Navigate to a license detail page from the customer detail (click a product name link). Verify all fields display correctly.

- [ ] **Step 6: Commit**

```bash
git add web/templates/licenses/detail.html internal/handler/page.go internal/handler/template.go
git commit -m "feat: add license detail page with assigned user display"
```

---

### Task 5: Create license form template and page handlers

**Files:**
- Create: `web/templates/licenses/form.html`
- Modify: `internal/handler/page.go` — add `licenseForm` and `licenseEditForm` methods and routes
- Modify: `internal/handler/template.go` — register `licenses/form`

- [ ] **Step 1: Create `web/templates/licenses/form.html`**

```html
{{define "content"}}
<header class="page-header">
    <h1>{{if .IsNew}}New License{{else}}Edit License{{end}} — {{.Customer.Name}}</h1>
</header>

<div class="card">
    <form id="license-form"
          {{if .IsNew}}hx-post="/api/v1/customers/{{uuidStr .Customer.ID}}/licenses"{{else}}hx-put="/api/v1/licenses/{{uuidStr .License.ID}}"{{end}}
          hx-ext="json-enc"
          hx-target="#form-message"
          hx-swap="innerHTML">

        <div id="form-message" aria-live="polite"></div>

        <div class="form-group">
            <label for="product_name">Product Name *</label>
            <input type="text" id="product_name" name="product_name"
                   value="{{if not .IsNew}}{{.License.ProductName}}{{end}}" required>
        </div>

        <div class="form-group">
            <label for="license_key">License Key</label>
            <input type="text" id="license_key" name="license_key"
                   value="{{if not .IsNew}}{{pgText .License.LicenseKey}}{{end}}">
        </div>

        <div class="form-group">
            <label for="quantity">Quantity *</label>
            <input type="number" id="quantity" name="quantity" min="1"
                   value="{{if .IsNew}}1{{else}}{{.License.Quantity}}{{end}}" required>
        </div>

        <div class="form-group">
            <label for="valid_from">Valid From</label>
            <input type="date" id="valid_from" name="valid_from"
                   value="{{if not .IsNew}}{{formatPgDate .License.ValidFrom}}{{end}}">
        </div>

        <div class="form-group">
            <label for="valid_until">Valid Until</label>
            <input type="date" id="valid_until" name="valid_until"
                   value="{{if not .IsNew}}{{formatPgDate .License.ValidUntil}}{{end}}">
        </div>

        <div class="form-group">
            <label for="user_assignment_id">Assigned to</label>
            <select id="user_assignment_id" name="user_assignment_id">
                <option value="">— Not assigned —</option>
                {{range .UserAssignments}}
                <option value="{{uuidStr .ID}}" {{if eq (uuidStr .ID) (uuidStr $.License.UserAssignmentID)}}selected{{end}}>{{.DisplayName}}</option>
                {{end}}
            </select>
        </div>

        <div class="flex gap-1">
            <button type="submit" class="btn btn-primary">{{if .IsNew}}Create{{else}}Save{{end}}</button>
            <a href="/customers/{{uuidStr .Customer.ID}}" class="btn">Cancel</a>
        </div>
    </form>
</div>

<script>
    document.body.addEventListener('htmx:afterRequest', function(evt) {
        if (evt.detail.elt.id === 'license-form' && evt.detail.successful) {
            var response = JSON.parse(evt.detail.xhr.responseText);
            if (response.id) {
                window.location.href = '/customers/{{uuidStr .Customer.ID}}/licenses/' + response.id;
            }
        }
    });
    document.body.addEventListener('htmx:responseError', function(evt) {
        if (evt.detail.elt.id === 'license-form') {
            var msg = 'Error saving';
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

- [ ] **Step 2: Add `licenseForm` handler method**

Routes were already registered in Task 4. Replace the stub (if added) with the full implementation:

```go
func (h *PageHandler) licenseForm(w http.ResponseWriter, r *http.Request) {
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
	listParams := model.ListParams{Limit: 100}
	assignments, _ := h.userAssignmentRepo.ListByCustomer(r.Context(), customerID, listParams)
	users, _ := h.userRepo.List(r.Context(), listParams)

	h.tmpl.RenderPage(w, "licenses/form", map[string]any{
		"Title":           "New License — " + customer.Name,
		"Customer":        customer,
		"License":         model.License{},
		"IsNew":           true,
		"UserAssignments": buildUserAssignmentDisplayList(assignments, users),
	})
}
```

- [ ] **Step 3: Add `licenseEditForm` handler method**

```go
func (h *PageHandler) licenseEditForm(w http.ResponseWriter, r *http.Request) {
	customerID, err := parseUUID(r.PathValue("customerId"))
	if err != nil {
		http.Error(w, "invalid customer ID", http.StatusBadRequest)
		return
	}
	licenseID, err := parseUUID(r.PathValue("licenseId"))
	if err != nil {
		http.Error(w, "invalid license ID", http.StatusBadRequest)
		return
	}
	customer, err := h.customerRepo.GetByID(r.Context(), customerID)
	if err != nil {
		http.Error(w, "customer not found", http.StatusNotFound)
		return
	}
	license, err := h.licenseRepo.GetByID(r.Context(), licenseID)
	if err != nil {
		http.Error(w, "license not found", http.StatusNotFound)
		return
	}
	listParams := model.ListParams{Limit: 100}
	assignments, _ := h.userAssignmentRepo.ListByCustomer(r.Context(), customerID, listParams)
	users, _ := h.userRepo.List(r.Context(), listParams)

	h.tmpl.RenderPage(w, "licenses/form", map[string]any{
		"Title":           "Edit License — " + customer.Name,
		"Customer":        customer,
		"License":         license,
		"IsNew":           false,
		"UserAssignments": buildUserAssignmentDisplayList(assignments, users),
	})
}
```

- [ ] **Step 4: Register the template in `template.go`**

Add to the `pageFiles` slice (after `licenses/detail.html`):

```go
filepath.Join(templateDir, "licenses", "form.html"),
```

- [ ] **Step 5: Verify compilation and test**

```bash
go vet ./...
docker compose up --build -d
```

Test the full flow:
1. Go to customer detail → click "+ New License" link (or navigate to `/customers/{id}/licenses/new`)
2. Fill in fields, select a user assignment, submit
3. Verify redirect to license detail page
4. Click "Edit" → verify form pre-fills all values including the assignment dropdown
5. Change the assignment, save, verify it updates

- [ ] **Step 6: Commit**

```bash
git add web/templates/licenses/form.html internal/handler/page.go internal/handler/template.go
git commit -m "feat: add license form page with user assignment dropdown"
```

---

### Task 6: Add "New License" button to customer detail

**Files:**
- Modify: `web/templates/customers/detail.html`

Currently licenses use an inline `<details>` form. Add a "New License" button link (like assets have) alongside the existing inline form.

- [ ] **Step 1: Add the button to the Licenses section header**

Replace:
```html
<header class="page-header"><h2>Licenses</h2></header>
```

With:
```html
<header class="page-header">
    <h2>Licenses</h2>
    <a href="/customers/{{uuidStr .Customer.ID}}/licenses/new" class="btn btn-primary btn-sm">+ New License</a>
</header>
```

This matches the Assets section pattern.

- [ ] **Step 2: Verify the page**

```bash
docker compose up --build -d
```

Open customer detail, verify the "+ New License" button appears and navigates to the form page.

- [ ] **Step 3: Commit**

```bash
git add web/templates/customers/detail.html
git commit -m "feat: add New License button to customer detail page"
```
