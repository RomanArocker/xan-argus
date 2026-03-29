# Template Conventions

Go HTML templates in `web/templates/`. Two-tier system: page templates (with layout) and partials (standalone for HTMX).

## Directory Structure

```
web/templates/
├── layout.html              # Base layout — shared by all pages
├── customers/
│   ├── list.html            # Full page (list view)
│   ├── list_rows.html       # Partial (table rows for HTMX)
│   ├── form.html            # Create/edit form
│   ├── detail.html          # Detail view
│   └── asset_detail.html    # Nested detail view
├── users/
│   ├── list.html
│   ├── list_rows.html
│   └── form.html
├── services/
│   └── ...                  # Same pattern
├── categories/
│   └── ...                  # Same pattern
└── import/
    └── page.html            # CSV import/export page (standalone, no partials)
```

Each entity directory contains:
- `list.html` — full page with search, table, and "New" button
- `list_rows.html` — partial with just `<tr>` rows (for HTMX swaps)
- `form.html` — create/edit form (shared template, uses `{{.IsNew}}` to toggle)
- `detail.html` — detail view (optional, when entity has related data)

## Layout Template

`layout.html` defines the HTML shell. All page templates inject content via `{{define "content"}}`:

```html
{{define "layout"}}
<!DOCTYPE html>
<html lang="en">
<head>
    <title>{{if .Title}}{{.Title}} — {{end}}XAN-Argus</title>
    <link rel="stylesheet" href="/static/css/style.css">
    <script src="/static/js/htmx.min.js"></script>
    <script src="/static/js/json-enc.js"></script>
</head>
<body>
    <nav>...</nav>
    <main>
        <div class="container">
            {{template "content" .}}
        </div>
    </main>
</body>
</html>
{{end}}
```

## Page Templates

Every page template wraps its content in `{{define "content"}}...{{end}}`:

```html
{{define "content"}}
<header class="page-header">
    <h1>Customers</h1>
    <a href="/customers/new" class="btn btn-primary">+ New Customer</a>
</header>
<!-- page content -->
{{end}}
```

### Template Data

Page handlers pass `map[string]any` to templates. Standard keys:

```go
h.tmpl.RenderPage(w, "customers/list", map[string]any{
    "Title":     "Customers",       // Always present — used in <title>
    "Customers": customers,         // Entity data
    "Search":    params.Search,     // Current search query (for list pages)
    "Filter":    params.Filter,     // Current filter value (optional)
})
```

Form pages include:

```go
map[string]any{
    "Title":    "New Customer",         // or "Edit Customer"
    "Customer": customer,               // Entity (empty struct for new)
    "IsNew":    true,                   // Toggle create vs edit mode
}
```

### Page Key Naming

Templates are referenced by `"entity/action"`:
- `"customers/list"`, `"customers/form"`, `"customers/detail"`
- `"users/list"`, `"users/form"`
- `"categories/list"`, `"categories/form"`

## Partial Templates

Partials define a named template for row content:

```html
{{define "customer_rows"}}
{{range .}}
<tr>
    <td><a href="/customers/{{uuidStr .ID}}">{{.Name}}</a></td>
    <td>{{pgText .ContactEmail}}</td>
    <td>{{formatDate .CreatedAt}}</td>
    <td><a href="/customers/{{uuidStr .ID}}/edit" class="btn btn-sm">Edit</a></td>
</tr>
{{end}}
{{end}}
```

Partial naming: `entity_rows` (underscore, not slash) — e.g., `customer_rows`, `user_rows`, `service_rows`.

List pages embed partials: `{{template "customer_rows" .Customers}}`

HTMX row endpoints render partials directly: `h.tmpl.RenderPartial(w, "customer_rows", customers)`

## Template Functions

Available via `newFuncMap()` in `template.go`:

| Function | Signature | Output |
|---|---|---|
| `formatDate` | `time.Time → string` | `"02.01.2006"` (DD.MM.YYYY) |
| `formatDateTime` | `time.Time → string` | `"02.01.2006 15:04"` |
| `formatPgDate` | `pgtype.Date → string` | `"02.01.2006"` or `"—"` |
| `pgText` | `pgtype.Text → string` | Text value or `""` |
| `uuidStr` | `pgtype.UUID → string` | UUID as lowercase hex |
| `map` | `map[string]string, key → string` | Map lookup with `""` fallback |
| `lower` | `string → string` | Lowercase |
| `default` | `def, val → string` | `val` if non-empty, else `def` |

## Registering New Templates

When adding a new page template, register it in `NewTemplateEngine()` in `template.go`:

1. Add the page file path to `pageFiles` slice
2. If it has a partial (rows), add to `partialFiles` slice
3. The page key is auto-derived: `dir/filename` (without `.html`)

## Do / Don't

- **Do** use `{{define "content"}}` for all page templates
- **Do** use template functions (`pgText`, `uuidStr`, `formatDate`) — never raw field access for nullable types
- **Do** keep templates minimal — no complex logic, no loops-within-loops
- **Don't** add `<html>`, `<head>`, or `<body>` tags in page templates — layout handles that
- **Don't** create standalone pages without the layout wrapper
- **Don't** add inline `<style>` blocks — use existing CSS classes
