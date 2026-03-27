# XAN-Pythia HTMX Frontend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an HTMX-powered web frontend for browsing, searching, and managing customers, users, services, and their dependent entities.

**Architecture:** Go HTML templates with a shared base layout. A single `PageHandler` struct holds all repositories and renders server-side HTML. HTMX handles search-as-you-type, inline delete with confirmation, and form submissions without full page reloads. All pages are server-rendered — no client-side JavaScript beyond HTMX.

**Tech Stack:** Go html/template, HTMX 2.0, minimal custom CSS (classless/simple)

**Existing codebase:** REST API is fully functional (`/api/v1/*`). Models, repositories, and JSON handlers are complete. This plan adds HTML rendering routes alongside the existing API.

---

## File Structure

```
web/
├── static/
│   ├── css/
│   │   └── style.css              # Minimal app styling
│   └── js/
│       └── htmx.min.js            # HTMX 2.0 (vendored)
└── templates/
    ├── layout.html                 # Base layout (nav, head, footer)
    ├── home.html                   # Landing page — customer list
    ├── customers/
    │   ├── list.html               # Customer list with search
    │   ├── list_rows.html          # HTMX partial: table rows only
    │   ├── detail.html             # Customer detail with sub-entity tabs
    │   └── form.html               # Create/edit customer form
    ├── users/
    │   ├── list.html               # User list with search + type filter
    │   ├── list_rows.html          # HTMX partial: table rows only
    │   └── form.html               # Create/edit user form
    └── services/
        ├── list.html               # Service catalog list
        ├── list_rows.html          # HTMX partial: table rows only
        └── form.html               # Create/edit service form

internal/
├── handler/
│   ├── page.go                     # PageHandler — all web routes + template rendering
│   └── template.go                 # Template engine: parse, helpers, render
```

**Design decisions:**
- One `PageHandler` struct with all repos (web pages often need multiple repos — e.g., customer detail shows assets, licenses, assignments, services)
- `list_rows.html` partials for HTMX swap on search — only the table body is re-rendered
- Forms post to API endpoints, HTMX intercepts the response to show success/error inline
- Customer detail page is the main "overview" — shows all sub-entities in sections
- No separate pages for assets/licenses/assignments/customer_services — they live on the customer detail page

---

## Task 1: Template Engine

**Files:**
- Create: `internal/handler/template.go`

- [ ] **Step 1: Create template engine**

```go
// internal/handler/template.go
package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type TemplateEngine struct {
	templates *template.Template
}

func NewTemplateEngine(templateDir string) (*TemplateEngine, error) {
	funcMap := template.FuncMap{
		"formatDate": func(t time.Time) string {
			return t.Format("02.01.2006")
		},
		"formatDateTime": func(t time.Time) string {
			return t.Format("02.01.2006 15:04")
		},
		"formatPgDate": func(d pgtype.Date) string {
			if !d.Valid {
				return "—"
			}
			return d.Time.Format("02.01.2006")
		},
		"pgText": func(t pgtype.Text) string {
			if !t.Valid {
				return ""
			}
			return t.String
		},
		"uuidStr": func(u pgtype.UUID) string {
			if !u.Valid {
				return ""
			}
			b := u.Bytes
			return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
		},
		"lower": strings.ToLower,
		"default": func(def string, val string) string {
			if val == "" {
				return def
			}
			return val
		},
	}

	patterns := []string{
		filepath.Join(templateDir, "*.html"),
		filepath.Join(templateDir, "customers", "*.html"),
		filepath.Join(templateDir, "users", "*.html"),
		filepath.Join(templateDir, "services", "*.html"),
	}

	tmpl := template.New("").Funcs(funcMap)
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("globbing templates %s: %w", pattern, err)
		}
		if len(matches) > 0 {
			if _, err := tmpl.ParseFiles(matches...); err != nil {
				return nil, fmt.Errorf("parsing templates: %w", err)
			}
		}
	}

	return &TemplateEngine{templates: tmpl}, nil
}

func (e *TemplateEngine) Render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := e.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalalServerError)
	}
}
```

- [ ] **Step 2: Verify compilation**

```bash
go vet ./internal/handler/...
```

- [ ] **Step 3: Commit**

```bash
git add internal/handler/template.go
git commit -m "feat: add template engine with helper functions"
```

---

## Task 2: Static Assets (CSS + HTMX)

**Files:**
- Create: `web/static/css/style.css`
- Create: `web/static/js/htmx.min.js` (vendored from CDN)

- [ ] **Step 1: Download HTMX**

```bash
curl -o web/static/js/htmx.min.js https://unpkg.com/htmx.org@2.0.4/dist/htmx.min.js
```

- [ ] **Step 2: Create minimal CSS**

```css
/* web/static/css/style.css */
:root {
  --bg: #f8f9fa;
  --surface: #ffffff;
  --text: #212529;
  --muted: #6c757d;
  --primary: #0d6efd;
  --danger: #dc3545;
  --success: #198754;
  --border: #dee2e6;
  --radius: 6px;
}

* { box-sizing: border-box; margin: 0; padding: 0; }

body {
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
  background: var(--bg);
  color: var(--text);
  line-height: 1.5;
}

/* Layout */
.container { max-width: 1200px; margin: 0 auto; padding: 0 1rem; }

nav {
  background: var(--text);
  color: white;
  padding: 0.75rem 0;
}
nav .container { display: flex; align-items: center; gap: 2rem; }
nav a { color: rgba(255,255,255,0.8); text-decoration: none; }
nav a:hover { color: white; }
nav .brand { font-weight: 700; font-size: 1.1rem; color: white; }

main { padding: 1.5rem 0; }

/* Cards */
.card {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 1.5rem;
  margin-bottom: 1rem;
}

/* Tables */
table { width: 100%; border-collapse: collapse; }
th, td { text-align: left; padding: 0.5rem 0.75rem; border-bottom: 1px solid var(--border); }
th { font-weight: 600; color: var(--muted); font-size: 0.85rem; text-transform: uppercase; }
tr:hover { background: var(--bg); }

/* Forms */
.form-group { margin-bottom: 1rem; }
label { display: block; font-weight: 500; margin-bottom: 0.25rem; font-size: 0.9rem; }
input, select, textarea {
  width: 100%;
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  font-size: 0.95rem;
}
textarea { min-height: 80px; resize: vertical; }

/* Buttons */
.btn {
  display: inline-block;
  padding: 0.5rem 1rem;
  border: none;
  border-radius: var(--radius);
  cursor: pointer;
  font-size: 0.9rem;
  text-decoration: none;
  transition: opacity 0.15s;
}
.btn:hover { opacity: 0.85; }
.btn-primary { background: var(--primary); color: white; }
.btn-danger { background: var(--danger); color: white; }
.btn-sm { padding: 0.25rem 0.5rem; font-size: 0.8rem; }

/* Search */
.search-bar {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 1rem;
}
.search-bar input { flex: 1; }
.search-bar select { width: auto; min-width: 150px; }

/* Page header */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}
.page-header h1 { font-size: 1.5rem; }

/* Tabs */
.tabs { display: flex; gap: 0; border-bottom: 2px solid var(--border); margin-bottom: 1rem; }
.tab {
  padding: 0.5rem 1rem;
  cursor: pointer;
  border-bottom: 2px solid transparent;
  margin-bottom: -2px;
  color: var(--muted);
  text-decoration: none;
}
.tab.active, .tab:hover { color: var(--primary); border-bottom-color: var(--primary); }

/* Detail sections */
.section { margin-bottom: 2rem; }
.section h2 { font-size: 1.1rem; margin-bottom: 0.75rem; }

/* Flash messages */
.alert {
  padding: 0.75rem 1rem;
  border-radius: var(--radius);
  margin-bottom: 1rem;
}
.alert-success { background: #d1e7dd; color: #0f5132; }
.alert-error { background: #f8d7da; color: #842029; }

/* HTMX indicator */
.htmx-indicator { display: none; }
.htmx-request .htmx-indicator { display: inline-block; }

/* Utilities */
.text-muted { color: var(--muted); }
.text-right { text-align: right; }
.mt-1 { margin-top: 0.5rem; }
.mb-1 { margin-bottom: 0.5rem; }
.gap-1 { gap: 0.5rem; }
.flex { display: flex; }
```

- [ ] **Step 3: Commit**

```bash
git add web/static/
git commit -m "feat: add static assets (HTMX 2.0, CSS)"
```

---

## Task 3: Base Layout Template

**Files:**
- Create: `web/templates/layout.html`

- [ ] **Step 1: Create base layout**

```html
{{/* web/templates/layout.html */}}
{{define "layout"}}
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{if .Title}}{{.Title}} — {{end}}XAN-Pythia</title>
    <link rel="stylesheet" href="/static/css/style.css">
    <script src="/static/js/htmx.min.js"></script>
</head>
<body>
    <nav>
        <div class="container">
            <a href="/" class="brand">XAN-Pythia</a>
            <a href="/customers">Customers</a>
            <a href="/users">Users</a>
            <a href="/services">Services</a>
        </div>
    </nav>
    <main>
        <div class="container">
            {{template "content" .}}
        </div>
    </main>
</body>
</html>
{{end}}
```

- [ ] **Step 2: Commit**

```bash
git add web/templates/layout.html
git commit -m "feat: add base layout template"
```

---

## Task 4: Customer List Page with Search

**Files:**
- Create: `web/templates/customers/list.html`
- Create: `web/templates/customers/list_rows.html`
- Create: `internal/handler/page.go` (PageHandler with customer list route)

- [ ] **Step 1: Create customer list template**

```html
{{/* web/templates/customers/list.html */}}
{{define "content"}}
<div class="page-header">
    <h1>Customers</h1>
    <a href="/customers/new" class="btn btn-primary">+ New Customer</a>
</div>

<div class="search-bar">
    <input type="search" name="q" placeholder="Search customers..."
           hx-get="/customers/rows"
           hx-trigger="keyup changed delay:300ms"
           hx-target="#customer-rows"
           value="{{.Search}}">
</div>

<div class="card">
    <table>
        <thead>
            <tr>
                <th>Name</th>
                <th>Email</th>
                <th>Created</th>
                <th></th>
            </tr>
        </thead>
        <tbody id="customer-rows">
            {{template "customer_rows" .Customers}}
        </tbody>
    </table>
</div>
{{end}}
```

- [ ] **Step 2: Create list rows partial**

```html
{{/* web/templates/customers/list_rows.html */}}
{{define "customer_rows"}}
{{range .}}
<tr>
    <td><a href="/customers/{{uuidStr .ID}}">{{.Name}}</a></td>
    <td>{{pgText .ContactEmail}}</td>
    <td>{{formatDateTime .CreatedAt}}</td>
    <td class="text-right">
        <button class="btn btn-danger btn-sm"
                hx-delete="/api/v1/customers/{{uuidStr .ID}}"
                hx-confirm="Really delete customer '{{.Name}}'?"
                hx-target="closest tr"
                hx-swap="outerHTML swap:0.3s">Delete</button>
    </td>
</tr>
{{else}}
<tr><td colspan="4" class="text-muted">No customers found.</td></tr>
{{end}}
{{end}}
```

- [ ] **Step 3: Create PageHandler with customer list**

```go
// internal/handler/page.go
package handler

import (
	"net/http"

	"github.com/xan-com/xan-pythia/internal/model"
	"github.com/xan-com/xan-pythia/internal/repository"
)

type PageHandler struct {
	tmpl                *TemplateEngine
	customerRepo        *repository.CustomerRepository
	userRepo            *repository.UserRepository
	serviceRepo         *repository.ServiceRepository
	userAssignmentRepo  *repository.UserAssignmentRepository
	assetRepo           *repository.AssetRepository
	licenseRepo         *repository.LicenseRepository
	customerServiceRepo *repository.CustomerServiceRepository
}

func NewPageHandler(
	tmpl *TemplateEngine,
	customerRepo *repository.CustomerRepository,
	userRepo *repository.UserRepository,
	serviceRepo *repository.ServiceRepository,
	userAssignmentRepo *repository.UserAssignmentRepository,
	assetRepo *repository.AssetRepository,
	licenseRepo *repository.LicenseRepository,
	customerServiceRepo *repository.CustomerServiceRepository,
) *PageHandler {
	return &PageHandler{
		tmpl:                tmpl,
		customerRepo:        customerRepo,
		userRepo:            userRepo,
		serviceRepo:         serviceRepo,
		userAssignmentRepo:  userAssignmentRepo,
		assetRepo:           assetRepo,
		licenseRepo:         licenseRepo,
		customerServiceRepo: customerServiceRepo,
	}
}

func (h *PageHandler) RegisterRoutes(mux *http.ServeMux) {
	// Pages
	mux.HandleFunc("GET /{$}", h.home)
	mux.HandleFunc("GET /customers", h.customerList)
	mux.HandleFunc("GET /customers/rows", h.customerListRows)
	mux.HandleFunc("GET /customers/new", h.customerForm)
	mux.HandleFunc("GET /customers/{id}", h.customerDetail)
	mux.HandleFunc("GET /customers/{id}/edit", h.customerEditForm)
}

func (h *PageHandler) home(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/customers", http.StatusFound)
}

func (h *PageHandler) customerList(w http.ResponseWriter, r *http.Request) {
	params := paginationParams(r)
	customers, err := h.customerRepo.List(r.Context(), params)
	if err != nil {
		http.Error(w, "failed to load customers", http.StatusInternalalServerError)
		return
	}

	data := map[string]any{
		"Title":     "Customers",
		"Customers": customers,
		"Search":    params.Search,
	}
	h.tmpl.Render(w, "layout", data)
}

func (h *PageHandler) customerListRows(w http.ResponseWriter, r *http.Request) {
	params := paginationParams(r)
	customers, err := h.customerRepo.List(r.Context(), params)
	if err != nil {
		http.Error(w, "failed to load customers", http.StatusInternalalServerError)
		return
	}
	h.tmpl.Render(w, "customer_rows", customers)
}

func (h *PageHandler) customerForm(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{
		"Title":    "New Customer",
		"Customer": model.Customer{},
		"IsNew":    true,
	}
	h.tmpl.Render(w, "layout", data)
}

func (h *PageHandler) customerDetail(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid ID", http.StatusBadRequest)
		return
	}

	customer, err := h.customerRepo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "customer not found", http.StatusNotFound)
		return
	}

	listParams := model.ListParams{Limit: 100}
	assets, _ := h.assetRepo.ListByCustomer(r.Context(), id, listParams)
	licenses, _ := h.licenseRepo.ListByCustomer(r.Context(), id, listParams)
	assignments, _ := h.userAssignmentRepo.ListByCustomer(r.Context(), id, listParams)
	customerServices, _ := h.customerServiceRepo.ListByCustomer(r.Context(), id, listParams)

	data := map[string]any{
		"Title":            customer.Name,
		"Customer":         customer,
		"Assets":           assets,
		"Licenses":         licenses,
		"Assignments":      assignments,
		"CustomerServices": customerServices,
	}
	h.tmpl.Render(w, "layout", data)
}

func (h *PageHandler) customerEditForm(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid ID", http.StatusBadRequest)
		return
	}

	customer, err := h.customerRepo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "customer not found", http.StatusNotFound)
		return
	}

	data := map[string]any{
		"Title":    "Edit Customer",
		"Customer": customer,
		"IsNew":    false,
	}
	h.tmpl.Render(w, "layout", data)
}
```

- [ ] **Step 4: Verify compilation**

```bash
go vet ./internal/handler/...
```

- [ ] **Step 5: Commit**

```bash
git add web/templates/customers/ internal/handler/page.go
git commit -m "feat: add customer list page with HTMX search"
```

---

## Task 5: Customer Detail Page

**Files:**
- Create: `web/templates/customers/detail.html`

- [ ] **Step 1: Create customer detail template**

```html
{{/* web/templates/customers/detail.html */}}
{{define "content"}}
<div class="page-header">
    <h1>{{.Customer.Name}}</h1>
    <div class="flex gap-1">
        <a href="/customers/{{uuidStr .Customer.ID}}/edit" class="btn btn-primary">Edit</a>
    </div>
</div>

<div class="card mb-1">
    <dl>
        <dt>Email</dt>
        <dd>{{pgText .Customer.ContactEmail | default "—"}}</dd>
        <dt>Notes</dt>
        <dd>{{pgText .Customer.Notes | default "—"}}</dd>
        <dt>Created</dt>
        <dd>{{formatDateTime .Customer.CreatedAt}}</dd>
    </dl>
</div>

{{/* Assets Section */}}
<div class="section">
    <div class="page-header">
        <h2>Assets</h2>
    </div>
    <div class="card">
        <table>
            <thead>
                <tr>
                    <th>Name</th>
                    <th>Type</th>
                    <th>Description</th>
                    <th>Created</th>
                    <th></th>
                </tr>
            </thead>
            <tbody>
                {{range .Assets}}
                <tr>
                    <td>{{.Name}}</td>
                    <td>{{.Type}}</td>
                    <td>{{pgText .Description}}</td>
                    <td>{{formatDateTime .CreatedAt}}</td>
                    <td class="text-right">
                        <button class="btn btn-danger btn-sm"
                                hx-delete="/api/v1/assets/{{uuidStr .ID}}"
                                hx-confirm="Really delete asset '{{.Name}}'?"
                                hx-target="closest tr"
                                hx-swap="outerHTML swap:0.3s">Delete</button>
                    </td>
                </tr>
                {{else}}
                <tr><td colspan="5" class="text-muted">No assets.</td></tr>
                {{end}}
            </tbody>
        </table>
    </div>
</div>

{{/* Licenses Section */}}
<div class="section">
    <div class="page-header">
        <h2>Licenses</h2>
    </div>
    <div class="card">
        <table>
            <thead>
                <tr>
                    <th>Product</th>
                    <th>Quantity</th>
                    <th>Valid From</th>
                    <th>Valid Until</th>
                    <th></th>
                </tr>
            </thead>
            <tbody>
                {{range .Licenses}}
                <tr>
                    <td>{{.ProductName}}</td>
                    <td>{{.Quantity}}</td>
                    <td>{{formatPgDate .ValidFrom}}</td>
                    <td>{{formatPgDate .ValidUntil}}</td>
                    <td class="text-right">
                        <button class="btn btn-danger btn-sm"
                                hx-delete="/api/v1/licenses/{{uuidStr .ID}}"
                                hx-confirm="Really delete license '{{.ProductName}}'?"
                                hx-target="closest tr"
                                hx-swap="outerHTML swap:0.3s">Delete</button>
                    </td>
                </tr>
                {{else}}
                <tr><td colspan="5" class="text-muted">No licenses.</td></tr>
                {{end}}
            </tbody>
        </table>
    </div>
</div>

{{/* User Assignments Section */}}
<div class="section">
    <div class="page-header">
        <h2>Assigned Users</h2>
    </div>
    <div class="card">
        <table>
            <thead>
                <tr>
                    <th>Role</th>
                    <th>Email</th>
                    <th>Phone</th>
                    <th></th>
                </tr>
            </thead>
            <tbody>
                {{range .Assignments}}
                <tr>
                    <td>{{.Role}}</td>
                    <td>{{pgText .Email}}</td>
                    <td>{{pgText .Phone}}</td>
                    <td class="text-right">
                        <button class="btn btn-danger btn-sm"
                                hx-delete="/api/v1/user-assignments/{{uuidStr .ID}}"
                                hx-confirm="Really delete assignment?"
                                hx-target="closest tr"
                                hx-swap="outerHTML swap:0.3s">Delete</button>
                    </td>
                </tr>
                {{else}}
                <tr><td colspan="4" class="text-muted">No users assigned.</td></tr>
                {{end}}
            </tbody>
        </table>
    </div>
</div>

{{/* Customer Services Section */}}
<div class="section">
    <div class="page-header">
        <h2>Subscribed Services</h2>
    </div>
    <div class="card">
        <table>
            <thead>
                <tr>
                    <th>Notes</th>
                    <th>Created</th>
                    <th></th>
                </tr>
            </thead>
            <tbody>
                {{range .CustomerServices}}
                <tr>
                    <td>{{pgText .Notes}}</td>
                    <td>{{formatDateTime .CreatedAt}}</td>
                    <td class="text-right">
                        <button class="btn btn-danger btn-sm"
                                hx-delete="/api/v1/customer-services/{{uuidStr .ID}}"
                                hx-confirm="Really delete service subscription?"
                                hx-target="closest tr"
                                hx-swap="outerHTML swap:0.3s">Delete</button>
                    </td>
                </tr>
                {{else}}
                <tr><td colspan="3" class="text-muted">No services subscribed.</td></tr>
                {{end}}
            </tbody>
        </table>
    </div>
</div>
{{end}}
```

Note: The `default` pipe function needs to be added to the template FuncMap in template.go:

```go
"default": func(def string, val string) string {
    if val == "" {
        return def
    }
    return val
},
```

- [ ] **Step 2: Commit**

```bash
git add web/templates/customers/detail.html
git commit -m "feat: add customer detail page with sub-entity sections"
```

---

## Task 6: Customer Create/Edit Form

**Files:**
- Create: `web/templates/customers/form.html`

- [ ] **Step 1: Create customer form template**

```html
{{/* web/templates/customers/form.html */}}
{{define "content"}}
<div class="page-header">
    <h1>{{if .IsNew}}New Customer{{else}}Edit Customer{{end}}</h1>
</div>

<div class="card">
    <form id="customer-form"
          {{if .IsNew}}
          hx-post="/api/v1/customers"
          {{else}}
          hx-put="/api/v1/customers/{{uuidStr .Customer.ID}}"
          {{end}}
          hx-ext="json-enc"
          hx-target="#form-message"
          hx-swap="innerHTML">

        <div id="form-message"></div>

        <div class="form-group">
            <label for="name">Name *</label>
            <input type="text" id="name" name="name" value="{{.Customer.Name}}" required>
        </div>

        <div class="form-group">
            <label for="contact_email">Email</label>
            <input type="email" id="contact_email" name="contact_email" value="{{pgText .Customer.ContactEmail}}">
        </div>

        <div class="form-group">
            <label for="notes">Notes</label>
            <textarea id="notes" name="notes">{{pgText .Customer.Notes}}</textarea>
        </div>

        <div class="flex gap-1">
            <button type="submit" class="btn btn-primary">
                {{if .IsNew}}Create{{else}}Save{{end}}
            </button>
            <a href="/customers" class="btn">Cancel</a>
        </div>
    </form>
</div>

<script>
    // Handle successful form submission — redirect to customer list
    document.body.addEventListener('htmx:afterRequest', function(evt) {
        if (evt.detail.elt.id === 'customer-form' && evt.detail.successful) {
            var response = JSON.parse(evt.detail.xhr.responseText);
            if (response.id) {
                window.location.href = '/customers/' + response.id;
            }
        }
    });
    // Handle error response — show error message
    document.body.addEventListener('htmx:responseError', function(evt) {
        if (evt.detail.elt.id === 'customer-form') {
            var msg = 'Error saving';
            try {
                var response = JSON.parse(evt.detail.xhr.responseText);
                if (response.error) msg = response.error;
            } catch(e) {}
            document.getElementById('form-message').innerHTML =
                '<div class="alert alert-error">' + msg + '</div>';
        }
    });
</script>
{{end}}
```

Note: HTMX `json-enc` extension is needed for sending JSON from forms. We need to include it. Alternative: use a small JS snippet to serialize form data as JSON. For MVP, use the HTMX json-enc extension.

Add to layout.html head:
```html
<script src="https://unpkg.com/htmx-ext-json-enc@2.0.1/json-enc.js"></script>
```

Or vendor it alongside htmx.min.js.

- [ ] **Step 2: Commit**

```bash
git add web/templates/customers/form.html
git commit -m "feat: add customer create/edit form with HTMX"
```

---

## Task 7: User List Page with Search + Filter

**Files:**
- Create: `web/templates/users/list.html`
- Create: `web/templates/users/list_rows.html`
- Modify: `internal/handler/page.go` (add user routes)

- [ ] **Step 1: Create user list template**

```html
{{/* web/templates/users/list.html */}}
{{define "content"}}
<div class="page-header">
    <h1>Users</h1>
    <a href="/users/new" class="btn btn-primary">+ New User</a>
</div>

<div class="search-bar">
    <input type="search" name="q" placeholder="Search users..."
           hx-get="/users/rows"
           hx-trigger="keyup changed delay:300ms"
           hx-target="#user-rows"
           hx-include="[name='type']"
           value="{{.Search}}">
    <select name="type"
            hx-get="/users/rows"
            hx-trigger="change"
            hx-target="#user-rows"
            hx-include="[name='q']">
        <option value="">All Types</option>
        <option value="customer_staff" {{if eq .Filter "customer_staff"}}selected{{end}}>Customer Staff</option>
        <option value="internal_staff" {{if eq .Filter "internal_staff"}}selected{{end}}>Internale Mitarbeiter</option>
    </select>
</div>

<div class="card">
    <table>
        <thead>
            <tr>
                <th>Name</th>
                <th>Type</th>
                <th>Created</th>
                <th></th>
            </tr>
        </thead>
        <tbody id="user-rows">
            {{template "user_rows" .Users}}
        </tbody>
    </table>
</div>
{{end}}
```

- [ ] **Step 2: Create user list rows partial**

```html
{{/* web/templates/users/list_rows.html */}}
{{define "user_rows"}}
{{range .}}
<tr>
    <td>{{.LastName}}, {{.FirstName}}</td>
    <td>{{if eq .Type "customer_staff"}}Customer{{else}}Internal{{end}}</td>
    <td>{{formatDateTime .CreatedAt}}</td>
    <td class="text-right">
        <button class="btn btn-danger btn-sm"
                hx-delete="/api/v1/users/{{uuidStr .ID}}"
                hx-confirm="Delete user '{{.FirstName}} {{.LastName}}'?"
                hx-target="closest tr"
                hx-swap="outerHTML swap:0.3s">Delete</button>
    </td>
</tr>
{{else}}
<tr><td colspan="4" class="text-muted">No users found.</td></tr>
{{end}}
{{end}}
```

- [ ] **Step 3: Add user routes to PageHandler**

Add these methods and routes to `internal/handler/page.go`:

```go
// Add to RegisterRoutes:
mux.HandleFunc("GET /users", h.userList)
mux.HandleFunc("GET /users/rows", h.userListRows)
mux.HandleFunc("GET /users/new", h.userForm)

// Methods:
func (h *PageHandler) userList(w http.ResponseWriter, r *http.Request) {
	params := paginationParams(r)
	users, err := h.userRepo.List(r.Context(), params)
	if err != nil {
		http.Error(w, "failed to load users", http.StatusInternalalServerError)
		return
	}
	data := map[string]any{
		"Title":  "Users",
		"Users":  users,
		"Search": params.Search,
		"Filter": params.Filter,
	}
	h.tmpl.Render(w, "layout", data)
}

func (h *PageHandler) userListRows(w http.ResponseWriter, r *http.Request) {
	params := paginationParams(r)
	users, err := h.userRepo.List(r.Context(), params)
	if err != nil {
		http.Error(w, "failed to load users", http.StatusInternalalServerError)
		return
	}
	h.tmpl.Render(w, "user_rows", users)
}

func (h *PageHandler) userForm(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{
		"Title": "New User",
		"User":  model.User{},
		"IsNew": true,
	}
	h.tmpl.Render(w, "layout", data)
}
```

- [ ] **Step 4: Verify compilation**

```bash
go vet ./internal/handler/...
```

- [ ] **Step 5: Commit**

```bash
git add web/templates/users/ internal/handler/page.go
git commit -m "feat: add user list page with search and type filter"
```

---

## Task 8: User Create/Edit Form

**Files:**
- Create: `web/templates/users/form.html`
- Modify: `internal/handler/page.go` (add user edit route)

- [ ] **Step 1: Create user form template (supports create and edit)**

```html
{{/* web/templates/users/form.html */}}
{{define "content"}}
<div class="page-header">
    <h1>{{if .IsNew}}New User{{else}}Edit User{{end}}</h1>
</div>

<div class="card">
    <form id="user-form"
          {{if .IsNew}}
          hx-post="/api/v1/users"
          {{else}}
          hx-put="/api/v1/users/{{uuidStr .User.ID}}"
          {{end}}
          hx-ext="json-enc"
          hx-target="#form-message"
          hx-swap="innerHTML">

        <div id="form-message"></div>

        <div class="form-group">
            <label for="type">Type *</label>
            <select id="type" name="type" required>
                <option value="">Please select</option>
                <option value="customer_staff" {{if eq .User.Type "customer_staff"}}selected{{end}}>Customer Staff</option>
                <option value="internal_staff" {{if eq .User.Type "internal_staff"}}selected{{end}}>Internalal Staff</option>
            </select>
        </div>

        <div class="form-group">
            <label for="first_name">First Name *</label>
            <input type="text" id="first_name" name="first_name" value="{{.User.FirstName}}" required>
        </div>

        <div class="form-group">
            <label for="last_name">Last Name *</label>
            <input type="text" id="last_name" name="last_name" value="{{.User.LastName}}" required>
        </div>

        <div class="flex gap-1">
            <button type="submit" class="btn btn-primary">
                {{if .IsNew}}Create{{else}}Save{{end}}
            </button>
            <a href="/users" class="btn">Cancel</a>
        </div>
    </form>
</div>

<script>
    document.body.addEventListener('htmx:afterRequest', function(evt) {
        if (evt.detail.elt.id === 'user-form' && evt.detail.successful) {
            window.location.href = '/users';
        }
    });
    document.body.addEventListener('htmx:responseError', function(evt) {
        if (evt.detail.elt.id === 'user-form') {
            var msg = 'Error saving';
            try { msg = JSON.parse(evt.detail.xhr.responseText).error || msg; } catch(e) {}
            document.getElementById('form-message').innerHTML =
                '<div class="alert alert-error">' + msg + '</div>';
        }
    });
</script>
{{end}}
```

- [ ] **Step 2: Add user edit route to PageHandler**

Add to `RegisterRoutes`:
```go
mux.HandleFunc("GET /users/{id}/edit", h.userEditForm)
```

Add method:
```go
func (h *PageHandler) userEditForm(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid ID", http.StatusBadRequest)
		return
	}
	user, err := h.userRepo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	data := map[string]any{
		"Title": "Edit User",
		"User":  user,
		"IsNew": false,
	}
	h.tmpl.Render(w, "layout", data)
}
```

- [ ] **Step 3: Add edit button to user list rows**

In `web/templates/users/list_rows.html`, add before the delete button:
```html
<a href="/users/{{uuidStr .ID}}/edit" class="btn btn-sm">Edit</a>
```

- [ ] **Step 4: Commit**

```bash
git add web/templates/users/form.html internal/handler/page.go
git commit -m "feat: add user create/edit form"
```

---

## Task 9: Service List Page + Form

**Files:**
- Create: `web/templates/services/list.html`
- Create: `web/templates/services/list_rows.html`
- Create: `web/templates/services/form.html`
- Modify: `internal/handler/page.go` (add service routes)

- [ ] **Step 1: Create service list template**

```html
{{/* web/templates/services/list.html */}}
{{define "content"}}
<div class="page-header">
    <h1>Services</h1>
    <a href="/services/new" class="btn btn-primary">+ New Service</a>
</div>

<div class="search-bar">
    <input type="search" name="q" placeholder="Search services..."
           hx-get="/services/rows"
           hx-trigger="keyup changed delay:300ms"
           hx-target="#service-rows"
           value="{{.Search}}">
</div>

<div class="card">
    <table>
        <thead>
            <tr>
                <th>Name</th>
                <th>Description</th>
                <th>Created</th>
                <th></th>
            </tr>
        </thead>
        <tbody id="service-rows">
            {{template "service_rows" .Services}}
        </tbody>
    </table>
</div>
{{end}}
```

- [ ] **Step 2: Create service list rows partial**

```html
{{/* web/templates/services/list_rows.html */}}
{{define "service_rows"}}
{{range .}}
<tr>
    <td>{{.Name}}</td>
    <td>{{pgText .Description}}</td>
    <td>{{formatDateTime .CreatedAt}}</td>
    <td class="text-right">
        <a href="/services/{{uuidStr .ID}}/edit" class="btn btn-sm">Edit</a>
        <button class="btn btn-danger btn-sm"
                hx-delete="/api/v1/services/{{uuidStr .ID}}"
                hx-confirm="Delete service '{{.Name}}'?"
                hx-target="closest tr"
                hx-swap="outerHTML swap:0.3s">Delete</button>
    </td>
</tr>
{{else}}
<tr><td colspan="4" class="text-muted">No services found.</td></tr>
{{end}}
{{end}}
```

- [ ] **Step 3: Create service form template (supports create and edit)**

```html
{{/* web/templates/services/form.html */}}
{{define "content"}}
<div class="page-header">
    <h1>{{if .IsNew}}New Service{{else}}Edit Service{{end}}</h1>
</div>

<div class="card">
    <form id="service-form"
          {{if .IsNew}}
          hx-post="/api/v1/services"
          {{else}}
          hx-put="/api/v1/services/{{uuidStr .Service.ID}}"
          {{end}}
          hx-ext="json-enc"
          hx-target="#form-message"
          hx-swap="innerHTML">

        <div id="form-message"></div>

        <div class="form-group">
            <label for="name">Name *</label>
            <input type="text" id="name" name="name" value="{{if not .IsNew}}{{.Service.Name}}{{end}}" required>
        </div>

        <div class="form-group">
            <label for="description">Description</label>
            <textarea id="description" name="description">{{if not .IsNew}}{{pgText .Service.Description}}{{end}}</textarea>
        </div>

        <div class="flex gap-1">
            <button type="submit" class="btn btn-primary">
                {{if .IsNew}}Create{{else}}Save{{end}}
            </button>
            <a href="/services" class="btn">Cancel</a>
        </div>
    </form>
</div>

<script>
    document.body.addEventListener('htmx:afterRequest', function(evt) {
        if (evt.detail.elt.id === 'service-form' && evt.detail.successful) {
            window.location.href = '/services';
        }
    });
    document.body.addEventListener('htmx:responseError', function(evt) {
        if (evt.detail.elt.id === 'service-form') {
            var msg = 'Error saving';
            try { msg = JSON.parse(evt.detail.xhr.responseText).error || msg; } catch(e) {}
            document.getElementById('form-message').innerHTML =
                '<div class="alert alert-error">' + msg + '</div>';
        }
    });
</script>
{{end}}
```

- [ ] **Step 4: Add service routes to PageHandler (including edit)**

Add to `RegisterRoutes`:
```go
mux.HandleFunc("GET /services", h.serviceList)
mux.HandleFunc("GET /services/rows", h.serviceListRows)
mux.HandleFunc("GET /services/new", h.serviceForm)
mux.HandleFunc("GET /services/{id}/edit", h.serviceEditForm)
```

Add methods:
```go
func (h *PageHandler) serviceList(w http.ResponseWriter, r *http.Request) {
	params := paginationParams(r)
	services, err := h.serviceRepo.List(r.Context(), params)
	if err != nil {
		http.Error(w, "failed to load services", http.StatusInternalalServerError)
		return
	}
	data := map[string]any{
		"Title":    "Services",
		"Services": services,
		"Search":   params.Search,
	}
	h.tmpl.Render(w, "layout", data)
}

func (h *PageHandler) serviceListRows(w http.ResponseWriter, r *http.Request) {
	params := paginationParams(r)
	services, err := h.serviceRepo.List(r.Context(), params)
	if err != nil {
		http.Error(w, "failed to load services", http.StatusInternalalServerError)
		return
	}
	h.tmpl.Render(w, "service_rows", services)
}

func (h *PageHandler) serviceForm(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{
		"Title":   "New Service",
		"Service": model.Service{},
		"IsNew":   true,
	}
	h.tmpl.Render(w, "layout", data)
}

func (h *PageHandler) serviceEditForm(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid ID", http.StatusBadRequest)
		return
	}
	svc, err := h.serviceRepo.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "service not found", http.StatusNotFound)
		return
	}
	data := map[string]any{
		"Title":   "Edit Service",
		"Service": svc,
		"IsNew":   false,
	}
	h.tmpl.Render(w, "layout", data)
}
```

- [ ] **Step 5: Verify compilation**

```bash
go vet ./internal/handler/...
```

- [ ] **Step 6: Commit**

```bash
git add web/templates/services/ internal/handler/page.go
git commit -m "feat: add service list page, search, and create form"
```

---

## Task 10: Wire Frontend in main.go + Download json-enc Extension

**Files:**
- Modify: `cmd/server/main.go` (add PageHandler + TemplateEngine)
- Create: `web/static/js/json-enc.js` (vendored HTMX extension)
- Modify: `web/templates/layout.html` (add json-enc script)

- [ ] **Step 1: Download json-enc extension**

```bash
curl -o web/static/js/json-enc.js https://unpkg.com/htmx-ext-json-enc@2.0.1/json-enc.js
```

- [ ] **Step 2: Update layout.html to include json-enc**

Add after the htmx script tag:
```html
<script src="/static/js/json-enc.js"></script>
```

- [ ] **Step 3: Update main.go**

Add after handler registrations:

```go
// Template engine
tmpl, err := handler.NewTemplateEngine("web/templates")
if err != nil {
    log.Fatalf("loading templates: %v", err)
}

// Page handler (web frontend)
pageHandler := handler.NewPageHandler(
    tmpl,
    customerRepo, userRepo, serviceRepo,
    userAssignmentRepo, assetRepo, licenseRepo, customerServiceRepo,
)
pageHandler.RegisterRoutes(mux)
```

- [ ] **Step 4: Verify compilation and build**

```bash
go vet ./...
go build ./cmd/server/
```

- [ ] **Step 5: Commit**

```bash
git add cmd/server/main.go web/static/js/json-enc.js web/templates/layout.html
git commit -m "feat: wire frontend templates and HTMX in main.go"
```

---

## Task 11: Integration Test — Docker Compose

- [ ] **Step 1: Build and start**

```bash
docker compose up --build -d
sleep 5
```

- [ ] **Step 2: Verify pages load**

```bash
# Health check
curl -s http://localhost:8080/health

# Customer list page
curl -s http://localhost:8080/customers | head -20

# Create a customer via API
curl -s -X POST http://localhost:8080/api/v1/customers \
  -H "Content-Type: application/json" \
  -d '{"name":"Test GmbH","contact_email":"info@test.de"}'

# Verify it shows on the page
curl -s http://localhost:8080/customers | grep "Test GmbH"

# Service list page
curl -s http://localhost:8080/services | head -20

# User list page
curl -s http://localhost:8080/users | head -20
```

- [ ] **Step 3: Tear down**

```bash
docker compose down
```

- [ ] **Step 4: Commit (if any fixes were needed)**

```bash
git add -A
git commit -m "fix: frontend integration fixes"
```

---

## Task 12: Inline Add Forms for Sub-Entities on Customer Detail

**Files:**
- Modify: `web/templates/customers/detail.html` (add collapsible inline forms per section)

Each sub-entity section gets an "Add" button that toggles an inline form. The form posts to the existing API endpoint with the customer ID pre-filled. On success, the page reloads to show the new entry.

- [ ] **Step 1: Add inline asset form**

After the Assets `</table>` and before `</div>`, add:

```html
<details class="mt-1">
    <summary class="btn btn-sm btn-primary">+ Add Asset</summary>
    <form class="card mt-1" id="add-asset-form"
          hx-post="/api/v1/customers/{{uuidStr .Customer.ID}}/assets"
          hx-ext="json-enc"
          hx-target="#form-msg-asset"
          hx-swap="innerHTML">
        <div id="form-msg-asset"></div>
        <div class="form-group">
            <label>Name *</label>
            <input type="text" name="name" required>
        </div>
        <div class="form-group">
            <label>Typ *</label>
            <select name="type" required>
                <option value="hardware">Hardware</option>
                <option value="software">Software</option>
            </select>
        </div>
        <div class="form-group">
            <label>Description</label>
            <textarea name="description"></textarea>
        </div>
        <button type="submit" class="btn btn-primary btn-sm">Save</button>
    </form>
</details>
```

- [ ] **Step 2: Add inline license form**

```html
<details class="mt-1">
    <summary class="btn btn-sm btn-primary">+ Add License</summary>
    <form class="card mt-1" id="add-license-form"
          hx-post="/api/v1/customers/{{uuidStr .Customer.ID}}/licenses"
          hx-ext="json-enc"
          hx-target="#form-msg-license"
          hx-swap="innerHTML">
        <div id="form-msg-license"></div>
        <div class="form-group">
            <label>Product Name *</label>
            <input type="text" name="product_name" required>
        </div>
        <div class="form-group">
            <label>License Key</label>
            <input type="text" name="license_key">
        </div>
        <div class="form-group">
            <label>Quantity *</label>
            <input type="number" name="quantity" value="1" min="1" required>
        </div>
        <button type="submit" class="btn btn-primary btn-sm">Save</button>
    </form>
</details>
```

- [ ] **Step 3: Add inline user assignment form**

This needs a dropdown of existing users. Pass `Users` from the handler.

Add to customer detail handler data:
```go
users, _ := h.userRepo.List(r.Context(), model.ListParams{Limit: 100})
// add to data map:
"Users": users,
```

```html
<details class="mt-1">
    <summary class="btn btn-sm btn-primary">+ Assign User</summary>
    <form class="card mt-1" id="add-assignment-form"
          hx-post="/api/v1/customers/{{uuidStr .Customer.ID}}/user-assignments"
          hx-ext="json-enc"
          hx-target="#form-msg-assign"
          hx-swap="innerHTML">
        <div id="form-msg-assign"></div>
        <div class="form-group">
            <label>User *</label>
            <select name="user_id" required>
                <option value="">Please select</option>
                {{range .Users}}
                <option value="{{uuidStr .ID}}">{{.LastName}}, {{.FirstName}}</option>
                {{end}}
            </select>
        </div>
        <div class="form-group">
            <label>Role *</label>
            <input type="text" name="role" required>
        </div>
        <div class="form-group">
            <label>Email</label>
            <input type="email" name="email">
        </div>
        <div class="form-group">
            <label>Phone</label>
            <input type="text" name="phone">
        </div>
        <button type="submit" class="btn btn-primary btn-sm">Save</button>
    </form>
</details>
```

- [ ] **Step 4: Add inline customer service form**

Needs `AllServices` from handler. Add to customer detail handler:
```go
allServices, _ := h.serviceRepo.List(r.Context(), model.ListParams{Limit: 100})
// add to data map:
"AllServices": allServices,
```

```html
<details class="mt-1">
    <summary class="btn btn-sm btn-primary">+ Subscribe Service</summary>
    <form class="card mt-1" id="add-cs-form"
          hx-post="/api/v1/customers/{{uuidStr .Customer.ID}}/services"
          hx-ext="json-enc"
          hx-target="#form-msg-cs"
          hx-swap="innerHTML">
        <div id="form-msg-cs"></div>
        <div class="form-group">
            <label>Service *</label>
            <select name="service_id" required>
                <option value="">Please select</option>
                {{range .AllServices}}
                <option value="{{uuidStr .ID}}">{{.Name}}</option>
                {{end}}
            </select>
        </div>
        <div class="form-group">
            <label>Notes</label>
            <textarea name="notes"></textarea>
        </div>
        <button type="submit" class="btn btn-primary btn-sm">Save</button>
    </form>
</details>
```

- [ ] **Step 5: Add JS reload on successful sub-entity creation**

At the bottom of `detail.html`, add:
```html
<script>
    document.body.addEventListener('htmx:afterRequest', function(evt) {
        var formIds = ['add-asset-form', 'add-license-form', 'add-assignment-form', 'add-cs-form'];
        if (formIds.includes(evt.detail.elt.id) && evt.detail.successful) {
            window.location.reload();
        }
    });
    document.body.addEventListener('htmx:responseError', function(evt) {
        var formIds = ['add-asset-form', 'add-license-form', 'add-assignment-form', 'add-cs-form'];
        if (formIds.includes(evt.detail.elt.id)) {
            var msg = 'Error saving';
            try { msg = JSON.parse(evt.detail.xhr.responseText).error || msg; } catch(e) {}
            var target = evt.detail.elt.querySelector('[id^="form-msg"]');
            if (target) target.innerHTML = '<div class="alert alert-error">' + msg + '</div>';
        }
    });
</script>
```

- [ ] **Step 6: Commit**

```bash
git add web/templates/customers/detail.html internal/handler/page.go
git commit -m "feat: add inline CRUD forms for sub-entities on customer detail"
```

---

## Summary

| Task | What's built |
|------|-------------|
| 1 | Template engine with helper functions |
| 2 | Static assets (HTMX 2.0, CSS) |
| 3 | Base layout template (nav, structure) |
| 4 | Customer list page with HTMX live search |
| 5 | Customer detail page with sub-entity sections + delete buttons |
| 6 | Customer create/edit form |
| 7 | User list page with search + type filter |
| 8 | User create/edit form |
| 9 | Service list page + create/edit form |
| 10 | Wire everything in main.go |
| 11 | Integration test with Docker Compose |
| 12 | Inline add forms for assets, licenses, assignments, services on customer detail |

**Total: 12 tasks**

**CRUD coverage:** All 7 entities have Create, Read (list + detail), and Delete from the UI. Top-level entities (customers, users, services) also have Edit. Sub-entities (assets, licenses, assignments, customer_services) are managed inline on the customer detail page.

All pages use English labels.
