# Asset Detail Page Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a read-only asset detail page accessible from the customer detail view, showing base info and category-specific field values.

**Architecture:** New page handler method on `PageHandler`, new template at `web/templates/customers/asset_detail.html`, link from customer detail table. Data flows through existing repositories.

**Tech Stack:** Go stdlib, HTMX, Go HTML templates, pgx

**Spec:** `docs/superpowers/specs/2026-03-28-asset-detail-page-design.md`

---

### Task 1: Create the asset detail template

**Files:**
- Create: `web/templates/customers/asset_detail.html`

- [ ] **Step 1: Create the template file**

```html
{{define "content"}}
<header class="page-header">
    <a href="/customers/{{uuidStr .Customer.ID}}">&larr; Back to {{.Customer.Name}}</a>
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

- [ ] **Step 2: Commit**

```bash
git add web/templates/customers/asset_detail.html
git commit -m "feat: add asset detail page template"
```

---

### Task 2: Register template and add page handler

**Files:**
- Modify: `internal/handler/template.go:72-81` (add to `pageFiles`)
- Modify: `internal/handler/page.go:46-63` (add route)
- Modify: `internal/handler/page.go` (add `assetDetail` method)

- [ ] **Step 1: Register the template in `pageFiles`**

In `internal/handler/template.go`, add to the `pageFiles` slice (after line 75, the `customers/form.html` entry):

```go
filepath.Join(templateDir, "customers", "asset_detail.html"),
```

- [ ] **Step 2: Add route in `RegisterRoutes`**

In `internal/handler/page.go`, inside `RegisterRoutes`, add after the `customers/{id}/edit` route (line 52):

```go
mux.HandleFunc("GET /customers/{customerId}/assets/{assetId}", h.assetDetail)
```

- [ ] **Step 3: Add `AssetFieldDisplay` type and `assetDetail` handler method**

In `internal/handler/page.go`, add the type and method:

```go
// AssetFieldDisplay holds a pre-resolved label+value pair for template rendering.
type AssetFieldDisplay struct {
	Name  string
	Value string
}

func (h *PageHandler) assetDetail(w http.ResponseWriter, r *http.Request) {
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
	if uuidToStr(asset.CustomerID) != uuidToStr(customerID) {
		http.Error(w, "asset not found", http.StatusNotFound)
		return
	}

	var cat *model.HardwareCategory
	var fields []AssetFieldDisplay
	if asset.CategoryID.Valid {
		// If category was deleted, err != nil → cat stays nil, fields stays empty.
		// This is intentional: show asset without category fields rather than erroring.
		c, err := h.hardwareCategoryRepo.GetByID(r.Context(), asset.CategoryID)
		if err == nil {
			cat = &c
			// c.Fields is pre-sorted by sort_order, name from the repository query
			var vals map[string]interface{}
			if err := json.Unmarshal(asset.FieldValues, &vals); err == nil {
				for _, fd := range c.Fields {
					v := "—"
					if raw, ok := vals[fd.Name]; ok && raw != nil {
						v = fmt.Sprintf("%v", raw)
					}
					fields = append(fields, AssetFieldDisplay{Name: fd.Name, Value: v})
				}
			}
		}
	}

	h.tmpl.RenderPage(w, "customers/asset_detail", map[string]any{
		"Title":    asset.Name + " — " + customer.Name,
		"Customer": customer,
		"Asset":    asset,
		"Category": cat,
		"Fields":   fields,
	})
}
```

- [ ] **Step 4: Add missing imports if needed**

Ensure `internal/handler/page.go` imports `"encoding/json"` and `"fmt"`. Check existing imports and add only if missing.

- [ ] **Step 5: Build and verify**

Run: `go vet ./...`
Expected: no errors

- [ ] **Step 6: Commit**

```bash
git add internal/handler/template.go internal/handler/page.go
git commit -m "feat: add asset detail page handler and route"
```

---

### Task 3: Link asset names in customer detail table

**Files:**
- Modify: `web/templates/customers/detail.html` (asset table row)

- [ ] **Step 1: Change asset name from plain text to link**

In `web/templates/customers/detail.html`, find the asset table row and replace:

```html
<td>{{.Name}}</td>
```

with:

```html
<td><a href="/customers/{{uuidStr $.Customer.ID}}/assets/{{uuidStr .ID}}">{{.Name}}</a></td>
```

- [ ] **Step 2: Commit**

```bash
git add web/templates/customers/detail.html
git commit -m "feat: link asset names to detail page in customer view"
```

---

### Task 4: Build, run, and verify end-to-end

- [ ] **Step 1: Build the full application**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 2: Run with Docker Compose**

Run: `docker compose up --build -d`
Expected: containers start successfully

- [ ] **Step 3: Verify the asset detail page**

1. Open a customer with assets in the browser
2. Verify asset names are clickable links
3. Click an asset — verify the detail page loads with:
   - Back link to customer
   - Base info (category, description, created, updated)
   - Category fields section (if asset has a category with field definitions)
4. Verify an asset without a category shows "—" for category and no fields section

- [ ] **Step 4: Verify edge cases**

1. Navigate to a URL with mismatched customer/asset IDs — should show 404
2. Navigate to a URL with invalid UUIDs — should show 400

- [ ] **Step 5: Commit if any fixes were needed**

```bash
git add -A
git commit -m "fix: address issues found during asset detail page verification"
```
