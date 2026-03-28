# License User Assignment — UI Design Spec

## Goal

Show and edit the `user_assignment_id` field on licenses in the web UI. The backend already fully supports this field (DB column, consistency trigger, model, repository, API handler) — this spec covers only UI changes.

## Design Decisions

- **Follow the Asset pattern exactly** — same dropdown, same display, same page structure
- **No JOINs in repositories** — separate lookups and maps in page handler (existing project pattern)
- **Dropdown select** for form input — same pattern as the asset user assignment dropdown
- **dl/dt/dd** for detail display — consistent with asset detail page
- **Map-based resolution** for license table on customer detail — avoids N+1 queries

## Changes

### 1. Customer Detail — License Table (`web/templates/customers/detail.html`)

Add "Assigned to" column to the license table and make Product Name a link to the new license detail page:

**Before:**
```html
<thead><tr><th>Product</th><th>Quantity</th><th>Valid From</th><th>Valid Until</th><th></th></tr></thead>
```

**After:**
```html
<thead><tr><th>Product</th><th>Assigned to</th><th>Quantity</th><th>Valid From</th><th>Valid Until</th><th></th></tr></thead>
```

Each row:
```html
<tr>
    <td><a href="/customers/{{uuidStr $.Customer.ID}}/licenses/{{uuidStr .ID}}">{{.ProductName}}</a></td>
    <td>{{if .UserAssignmentID.Valid}}{{mapGet $.LicenseAssignmentMap (uuidStr .UserAssignmentID)}}{{else}}<span class="text-muted">—</span>{{end}}</td>
    <td>{{.Quantity}}</td>
    <td>{{formatPgDate .ValidFrom}}</td>
    <td>{{formatPgDate .ValidUntil}}</td>
    <td class="text-right">...</td>
</tr>
```

Update `colspan` on empty state from `5` to `6`.

The page handler builds a `LicenseAssignmentMap` (`map[string]string` of assignment UUID → "LastName, FirstName (Role)") by:
1. Collecting all `UserAssignmentID` values from licenses
2. Looking up each assignment via `userAssignmentRepo.GetByID()`
3. Looking up user name via the existing `userMap` (already built for the Assigned Users section)
4. Building display string: `"LastName, FirstName (Role)"`

Alternatively, reuse the same map-building approach used for `CategoryMap` — load all assignments for the customer once, build the map.

### 2. Customer Detail — "Add License" Form (`web/templates/customers/detail.html`)

Add user assignment dropdown to the inline "Add License" form, between License Key and Quantity:

```html
<div class="form-group"><label>Assigned to</label>
    <select name="user_assignment_id">
        <option value="">— Not assigned —</option>
        {{range .Assignments}}<option value="{{uuidStr .ID}}">{{.DisplayName}}</option>{{end}}
    </select>
</div>
```

The `.Assignments` data is already loaded for the "Assigned Users" section. We need a display-friendly version with "LastName, FirstName (Role)" — build a `[]UserAssignmentDisplay` slice (reusing the existing `UserAssignmentDisplay` struct from the asset feature) in the page handler and pass it as a new template variable.

### 3. New License Detail Page (`web/templates/licenses/detail.html`)

New template at `web/templates/licenses/detail.html`:

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

`AssignedUser` is a pre-resolved string: `"LastName, FirstName (Role)"` or empty.

### 4. New License Form Page (`web/templates/licenses/form.html`)

New template at `web/templates/licenses/form.html`:

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

### 5. Page Handler Routes (`internal/handler/page.go`)

Add new routes in `RegisterRoutes`:

```go
mux.HandleFunc("GET /customers/{customerId}/licenses/{id}", h.licenseDetail)
mux.HandleFunc("GET /customers/{customerId}/licenses/new", h.licenseForm)
mux.HandleFunc("GET /customers/{customerId}/licenses/{id}/edit", h.licenseEditForm)
```

#### `licenseDetail`

1. Parse `customerId` and `id` from path
2. Load customer via `customerRepo.GetByID()`
3. Load license via `licenseRepo.GetByID()`
4. If `license.UserAssignmentID.Valid`:
   - Call `userAssignmentRepo.GetByID(ctx, license.UserAssignmentID)`
   - Call `userRepo.GetByID(ctx, assignment.UserID)` to get user name
   - Build display string: `"LastName, FirstName (Role)"`
5. Render `"licenses/detail"` with `Customer`, `License`, `AssignedUser`

#### `licenseForm` and `licenseEditForm`

1. Parse `customerId` from path (and `id` for edit)
2. Load customer via `customerRepo.GetByID()`
3. Load user assignments for the customer: `userAssignmentRepo.ListByCustomer()`
4. Load all users: `userRepo.List()`, build `userMap[uuid]string`
5. Build `[]UserAssignmentDisplay` with `DisplayName = "LastName, FirstName (Role)"`
6. For edit: load license via `licenseRepo.GetByID()`
7. Render `"licenses/form"` with `Customer`, `License`, `UserAssignments`, `IsNew`

#### `customerDetail` updates

Add `LicenseAssignmentMap` to template data:
1. After loading licenses, collect unique `UserAssignmentID` values
2. Build map using same user-assignment-to-display-name resolution
3. Pass as `"LicenseAssignmentMap"` alongside existing template data
4. Also build and pass `[]UserAssignmentDisplay` for the "Add License" form dropdown

### 6. Template Engine Registration (`internal/handler/template.go`)

Register the two new templates:
- Page: `web/templates/licenses/detail.html` → key `"licenses/detail"`
- Page: `web/templates/licenses/form.html` → key `"licenses/form"`

## Out of Scope

- License list page (licenses are always scoped to a customer)
- License search/filter
- Bulk assignment of licenses to users
- Any backend changes (already complete)
