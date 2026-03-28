# Asset User Assignment — UI Design Spec

## Goal

Show and edit the `user_assignment_id` field on assets in the web UI. The backend already supports this field — this spec covers only UI changes.

## Design Decisions

- **No JOINs in repositories** — follow existing project pattern of separate lookups and maps in handlers
- **Dropdown select** for form input — same pattern as the category dropdown
- **dl/dt/dd** for detail display — consistent with other fields on the asset detail page

## Changes

### 1. Asset Form (`web/templates/assets/form.html`)

Add a `<select>` dropdown for `user_assignment_id` after the category dropdown:

```html
<div class="form-group">
    <label for="user_assignment_id">Assigned to</label>
    <select id="user_assignment_id" name="user_assignment_id">
        <option value="">— Not assigned —</option>
        {{range .UserAssignments}}
        <option value="{{uuidStr .ID}}" {{if eq (uuidStr .ID) (uuidStr $.Asset.UserAssignmentID)}}selected{{end}}>
            {{.DisplayName}}
        </option>
        {{end}}
    </select>
</div>
```

The handler builds a display-friendly list with `DisplayName` = `"LastName, FirstName (Role)"`.

The existing form submit JavaScript (custom `fetch()`, not HTMX) must be updated to include `user_assignment_id` in the JSON body. Add after the existing `category_id` handling:

```javascript
var assignSelect = document.getElementById('user_assignment_id');
if (assignSelect.value) body.user_assignment_id = assignSelect.value;
```

### 2. Asset Detail (`web/templates/customers/asset_detail.html`)

Add a new `<dt>/<dd>` pair in the existing `<dl>`:

```html
<dt>Assigned to</dt>
<dd>{{if .AssignedUser}}{{.AssignedUser}}{{else}}—{{end}}</dd>
```

`AssignedUser` is a pre-resolved string: `"LastName, FirstName (Role)"` or empty.

### 3. Page Handler (`internal/handler/page.go`)

#### `assetForm` and `assetEditForm`

Load user assignments for the customer and build display data:

1. Call `userAssignmentRepo.ListByCustomer(ctx, customerID, params)` to get all assignments
2. Call `userRepo.List(ctx, params)` once to get all users, build a `userMap[uuid]string` of `ID → "LastName, FirstName"` (avoids N+1 queries — same pattern as `CategoryMap`)
3. Iterate assignments, look up user name from map, build `[]UserAssignmentDisplay` with `DisplayName = "LastName, FirstName (Role)"`
4. Pass as `"UserAssignments"` to the template

#### `assetDetail`

Resolve the assigned user for display:

1. If `asset.UserAssignmentID.Valid`, call `userAssignmentRepo.GetByID(ctx, asset.UserAssignmentID)`
2. Then call `userRepo.GetByID(ctx, assignment.UserID)` to get the user name
3. Build display string: `"LastName, FirstName (Role)"`
4. Pass as `"AssignedUser"` (string) to the template

### 4. Handler Types

Add a small display struct in `page.go` (following `AssetFieldDisplay` pattern):

```go
type UserAssignmentDisplay struct {
    ID          pgtype.UUID
    DisplayName string
}
```

### 5. Customer Detail — Asset Table

The asset table on the customer detail page (`web/templates/customers/detail.html`) currently shows: Name, Category, Description, Created.

No change needed — showing the assigned user in the asset table would add clutter. The information is visible on the asset detail page.

## Out of Scope

- Search/filter assets by assigned user
- Bulk assignment
- Assignment history/audit trail
