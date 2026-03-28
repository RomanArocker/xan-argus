# Asset User Assignment UI — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add user assignment dropdown to asset form and display assigned user on asset detail page.

**Architecture:** Handler builds display data via separate lookups (no JOINs) following the existing CategoryMap pattern. Templates use the same dropdown/dl patterns already established.

**Tech Stack:** Go handlers, Go HTML templates, HTMX, vanilla JavaScript

**Spec:** `docs/superpowers/specs/2026-03-28-asset-user-assignment-ui-design.md`

---

## File Map

| Action | File | Responsibility |
|--------|------|----------------|
| Modify | `internal/handler/page.go` | Add `UserAssignmentDisplay` struct, update `assetForm`, `assetEditForm`, `assetDetail` |
| Modify | `web/templates/assets/form.html` | Add assignment dropdown + JS submit line |
| Modify | `web/templates/customers/asset_detail.html` | Add "Assigned to" dt/dd |

---

### Task 1: Add `UserAssignmentDisplay` struct and helper to `page.go`

**Files:**
- Modify: `internal/handler/page.go:315-325`

- [ ] **Step 1: Add display struct and helper function**

Add after the existing `AssetFieldDisplay` struct (around line 316-318):

```go
// UserAssignmentDisplay holds a pre-resolved assignment for template dropdowns.
type UserAssignmentDisplay struct {
	ID          pgtype.UUID
	DisplayName string
}

// buildUserAssignmentDisplayList builds display-friendly assignment list using a user map.
func buildUserAssignmentDisplayList(assignments []model.UserAssignment, users []model.User) []UserAssignmentDisplay {
	userMap := make(map[string]string, len(users))
	for _, u := range users {
		userMap[uuidToStr(u.ID)] = u.LastName + ", " + u.FirstName
	}
	result := make([]UserAssignmentDisplay, 0, len(assignments))
	for _, a := range assignments {
		name := userMap[uuidToStr(a.UserID)]
		display := name + " (" + a.Role + ")"
		result = append(result, UserAssignmentDisplay{ID: a.ID, DisplayName: display})
	}
	return result
}
```

- [ ] **Step 2: Verify compilation**

```bash
go vet ./...
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/handler/page.go
git commit -m "feat: add UserAssignmentDisplay struct and builder helper"
```

---

### Task 2: Update `assetForm` handler to load user assignments

**Files:**
- Modify: `internal/handler/page.go:385-408`

- [ ] **Step 1: Add user assignment loading to `assetForm`**

Replace the `assetForm` method (lines 385-408) with:

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
	assignments, _ := h.userAssignmentRepo.ListByCustomer(r.Context(), customerID, model.ListParams{Limit: 1000})
	users, _ := h.userRepo.List(r.Context(), model.ListParams{Limit: 1000})
	h.tmpl.RenderPage(w, "assets/form", map[string]any{
		"Title":           "New Asset — " + customer.Name,
		"Customer":        customer,
		"Asset":           model.Asset{},
		"Categories":      categories,
		"UserAssignments": buildUserAssignmentDisplayList(assignments, users),
		"IsNew":           true,
	})
}
```

- [ ] **Step 2: Verify compilation**

```bash
go vet ./...
```

- [ ] **Step 3: Commit**

```bash
git add internal/handler/page.go
git commit -m "feat: load user assignments in assetForm handler"
```

---

### Task 3: Update `assetEditForm` handler to load user assignments

**Files:**
- Modify: `internal/handler/page.go:410-453`

- [ ] **Step 1: Add user assignment loading to `assetEditForm`**

Replace the `assetEditForm` method (lines 410-453) with:

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

	assignments, _ := h.userAssignmentRepo.ListByCustomer(r.Context(), customerID, model.ListParams{Limit: 1000})
	users, _ := h.userRepo.List(r.Context(), model.ListParams{Limit: 1000})
	h.tmpl.RenderPage(w, "assets/form", map[string]any{
		"Title":           "Edit Asset — " + customer.Name,
		"Customer":        customer,
		"Asset":           asset,
		"Categories":      categories,
		"CategoryFields":  categoryFields,
		"UserAssignments": buildUserAssignmentDisplayList(assignments, users),
		"IsNew":           false,
	})
}
```

- [ ] **Step 2: Verify compilation**

```bash
go vet ./...
```

- [ ] **Step 3: Commit**

```bash
git add internal/handler/page.go
git commit -m "feat: load user assignments in assetEditForm handler"
```

---

### Task 4: Update `assetDetail` handler to resolve assigned user

**Files:**
- Modify: `internal/handler/page.go:327-383`

- [ ] **Step 1: Add assigned user resolution to `assetDetail`**

After the category/fields block (line 374) and before `h.tmpl.RenderPage`, add:

```go
	var assignedUser string
	if asset.UserAssignmentID.Valid {
		ua, err := h.userAssignmentRepo.GetByID(r.Context(), asset.UserAssignmentID)
		if err == nil {
			u, err := h.userRepo.GetByID(r.Context(), ua.UserID)
			if err == nil {
				assignedUser = u.LastName + ", " + u.FirstName + " (" + ua.Role + ")"
			}
		}
	}
```

Then update the `RenderPage` call to include `"AssignedUser": assignedUser`:

```go
	h.tmpl.RenderPage(w, "customers/asset_detail", map[string]any{
		"Title":        asset.Name + " — " + customer.Name,
		"Customer":     customer,
		"Asset":        asset,
		"Category":     cat,
		"Fields":       fields,
		"AssignedUser": assignedUser,
	})
```

- [ ] **Step 2: Verify compilation**

```bash
go vet ./...
```

- [ ] **Step 3: Commit**

```bash
git add internal/handler/page.go
git commit -m "feat: resolve assigned user in assetDetail handler"
```

---

### Task 5: Add assignment dropdown to asset form template

**Files:**
- Modify: `web/templates/assets/form.html`

- [ ] **Step 1: Add `<select>` dropdown after category dropdown**

After the `<div id="category-fields">` block and before the description `<div class="form-group">`, add:

```html
        <div class="form-group">
            <label for="user_assignment_id">Assigned to</label>
            <select id="user_assignment_id" name="user_assignment_id">
                <option value="">— Not assigned —</option>
                {{range .UserAssignments}}
                <option value="{{uuidStr .ID}}" {{if eq (uuidStr .ID) (uuidStr $.Asset.UserAssignmentID)}}selected{{end}}>{{.DisplayName}}</option>
                {{end}}
            </select>
        </div>
```

- [ ] **Step 2: Add `user_assignment_id` to the JS submit body**

In the `<script>` block, find the line:

```javascript
        if (categorySelect.value) body.category_id = categorySelect.value;
```

Add immediately after it:

```javascript
        var assignSelect = document.getElementById('user_assignment_id');
        if (assignSelect.value) body.user_assignment_id = assignSelect.value;
```

- [ ] **Step 3: Commit**

```bash
git add web/templates/assets/form.html
git commit -m "feat: add user assignment dropdown to asset form template"
```

---

### Task 6: Add "Assigned to" field to asset detail template

**Files:**
- Modify: `web/templates/customers/asset_detail.html`

- [ ] **Step 1: Add dt/dd pair to the existing dl**

In the `<dl>` block, add after the Category line (line 12) and before the Description line:

```html
        <dt>Assigned to</dt><dd>{{if .AssignedUser}}{{.AssignedUser}}{{else}}—{{end}}</dd>
```

- [ ] **Step 2: Commit**

```bash
git add web/templates/customers/asset_detail.html
git commit -m "feat: show assigned user on asset detail page"
```

---

### Task 7: Build and verify end-to-end

- [ ] **Step 1: Rebuild and start**

```bash
docker compose up --build -d
```

- [ ] **Step 2: Verify asset form shows dropdown**

Open browser: navigate to a customer → New Asset. Verify "Assigned to" dropdown appears with user assignments from that customer.

- [ ] **Step 3: Verify create with assignment works**

Create a new asset with a user assignment selected. Verify redirect to detail page shows the assigned user.

- [ ] **Step 4: Verify edit form pre-selects assignment**

Edit the asset. Verify the dropdown has the correct user pre-selected.

- [ ] **Step 5: Verify detail page shows assigned user**

View the asset detail page. Verify "Assigned to" shows "LastName, FirstName (Role)".

- [ ] **Step 6: Verify asset without assignment**

View/create an asset without assignment. Verify "Assigned to" shows "—".

## Verification Checklist

- [ ] `go vet ./...` passes
- [ ] Docker builds without errors
- [ ] Asset form shows "Assigned to" dropdown with customer's user assignments
- [ ] Creating asset with assignment works
- [ ] Editing asset preserves/changes assignment
- [ ] Asset detail shows "Assigned to: LastName, FirstName (Role)"
- [ ] Asset without assignment shows "—"
