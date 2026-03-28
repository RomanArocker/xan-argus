# XAN-Argus — Complete Project Description for AI Reconstruction

This document describes XAN-Argus comprehensively enough that another AI can rebuild the same product from scratch.

---

## 1. What is XAN-Argus?

XAN-Argus is a **multi-tenant asset, user, license, and service management system** for an IT services business. It serves as a central database and lookup tool to track:
- What each customer owns (assets / hardware)
- Who works there (users / assignments)
- Which licenses they hold
- Which services they subscribe to

**MVP Scope:** CRUD, search, and overview. No active features (notifications, reports, dashboards). No authentication.

---

## 2. Tech Stack

| Component | Technology |
|---|---|
| Backend | Go (stdlib + `github.com/jackc/pgx/v5`) |
| Database | PostgreSQL 18.3 |
| Frontend | HTMX + Go HTML Templates |
| Migrations | goose (pure SQL, run on app startup) |
| Deployment | Docker Compose (multi-stage build) |
| Auth | Explicitly deferred to post-MVP |

Go module: `module xan-argus`, `go 1.23`

Dependencies:
- `github.com/jackc/pgx/v5` — PostgreSQL client
- `github.com/pressly/goose/v3` — migrations
- `github.com/google/uuid` — UUID utilities

---

## 3. Project Structure

```
xan-argus/
├── cmd/server/main.go              # Entry point — wires everything together
├── internal/
│   ├── handler/                    # HTTP handlers (API + page)
│   │   ├── asset.go
│   │   ├── customer.go
│   │   ├── customer_service.go
│   │   ├── field_validation.go     # JSONB field_values validation
│   │   ├── hardware_category.go
│   │   ├── json.go                 # writeJSON, writeError, decodeJSON
│   │   ├── license.go
│   │   ├── page.go                 # All HTML page routes in one struct
│   │   ├── params.go               # parseUUID, paginationParams
│   │   ├── service.go
│   │   ├── template.go             # TemplateEngine
│   │   ├── user.go
│   │   └── user_assignment.go
│   ├── repository/                 # SQL via pgx, one file per entity
│   ├── model/                      # Plain Go structs, no methods
│   ├── database/                   # Connection pool + migration runner
│   └── middleware/                 # HTTP logging
├── db/migrations/                  # goose SQL migrations (001–006)
├── web/
│   ├── templates/                  # Go HTML templates
│   │   ├── layout.html
│   │   ├── customers/
│   │   ├── users/
│   │   ├── assets/
│   │   ├── services/
│   │   ├── licenses/
│   │   ├── user_assignments/
│   │   └── categories/
│   └── static/
│       ├── css/style.css
│       ├── js/htmx.min.js
│       └── js/json-enc.js
├── Dockerfile                      # Multi-stage: golang:1.23 → alpine
├── docker-compose.yml
├── Makefile
└── VERSION
```

**No service layer in MVP** — handlers call repositories directly.

---

## 4. Database Schema (PostgreSQL)

### General Conventions
- Primary key: `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`
- Timestamps: `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`, `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- Shared trigger function `set_updated_at()` applied to every table
- Foreign keys: **always `ON DELETE RESTRICT`** — no cascading deletes
- Enum validation via `CHECK` constraints

### Tables

#### `customers`
```sql
id UUID PK, name TEXT NOT NULL UNIQUE, contact_email TEXT, notes TEXT, created_at, updated_at
```

#### `users`
```sql
id UUID PK,
type TEXT NOT NULL CHECK (type IN ('customer_staff', 'internal_staff')),
first_name TEXT NOT NULL, last_name TEXT NOT NULL, created_at, updated_at
```

#### `user_assignments`
```sql
id UUID PK,
user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
role TEXT NOT NULL, email TEXT, phone TEXT, notes TEXT,
created_at, updated_at,
UNIQUE(user_id, customer_id)
```
**Trigger:** `check_user_assignment_type` — only `customer_staff` users may be assigned to customers.

#### `services` (catalog)
```sql
id UUID PK, name TEXT NOT NULL, description TEXT, created_at, updated_at
```

#### `assets`
```sql
id UUID PK,
customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
category_id UUID REFERENCES hardware_categories(id) ON DELETE SET NULL,
name TEXT NOT NULL, description TEXT,
metadata JSONB DEFAULT '{}',           -- free-form metadata (serial, warranty_until, etc.)
field_values JSONB NOT NULL DEFAULT '{}',  -- typed fields from the hardware category
created_at, updated_at
```
GIN indexes on `metadata` and `field_values`.

#### `licenses`
```sql
id UUID PK,
customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
user_assignment_id UUID REFERENCES user_assignments(id) ON DELETE RESTRICT,  -- optional
product_name TEXT NOT NULL, license_key TEXT, quantity INTEGER,
valid_from DATE, valid_until DATE,
created_at, updated_at
```
**Trigger:** `check_license_customer_consistency` — when `user_assignment_id` is set, its `customer_id` must match `licenses.customer_id`.

#### `customer_services`
```sql
id UUID PK,
customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
service_id UUID NOT NULL REFERENCES services(id) ON DELETE RESTRICT,
customizations JSONB DEFAULT '{}',  -- e.g. {"sla": "8x5", "response_time_hours": 4}
notes TEXT,
created_at, updated_at,
UNIQUE(customer_id, service_id)
```
GIN index on `customizations`.

#### `hardware_categories`
```sql
id UUID PK, name TEXT NOT NULL UNIQUE, description TEXT, created_at, updated_at
```

#### `category_field_definitions`
```sql
id UUID PK,
category_id UUID NOT NULL REFERENCES hardware_categories(id) ON DELETE CASCADE,
name TEXT NOT NULL,
field_type TEXT NOT NULL CHECK (field_type IN ('text', 'number', 'date', 'boolean')),
required BOOLEAN NOT NULL DEFAULT false,
sort_order INTEGER NOT NULL DEFAULT 0,
created_at, updated_at,
UNIQUE(category_id, name)
```
**Trigger:** `strip_deleted_field_values` — when a field definition is deleted, removes its key from all `assets.field_values`.

### Migration Order
1. `001_base_schema.sql` — customers, users, services + `set_updated_at()`
2. `002_dependent_tables.sql` — user_assignments, assets, licenses, customer_services
3. `003_hardware_categories.sql` — hardware_categories, category_field_definitions; assets get category_id + field_values
4. `004_user_assignment_type_check.sql` — trigger: only customer_staff may be assigned
5. `005_seed_data.sql` — 3 customers, users, assignments, assets, licenses, customer-services
6. `006_asset_user_assignment.sql` — optional link from asset to user_assignment

---

## 5. API Design

REST JSON API under `/api/v1/`. No auth in MVP.

**Error format:** `{"error": "human-readable message"}` with appropriate HTTP status code.

**Pagination:** `?limit=&offset=&q=`

| Method | Path | Description |
|---|---|---|
| GET/POST | `/api/v1/customers` | List / Create customers |
| GET/PUT/DELETE | `/api/v1/customers/{id}` | Get / Update / Delete customer |
| GET/POST | `/api/v1/customers/{id}/assets` | List / Create assets for customer |
| GET/PUT/DELETE | `/api/v1/assets/{id}` | Asset CRUD |
| GET/POST | `/api/v1/users` | List / Create users |
| GET/PUT/DELETE | `/api/v1/users/{id}` | User CRUD |
| GET/POST | `/api/v1/customers/{id}/user-assignments` | List / Create assignments |
| GET/PUT/DELETE | `/api/v1/user-assignments/{id}` | Assignment CRUD |
| GET/POST | `/api/v1/customers/{id}/licenses` | List / Create licenses |
| GET/PUT/DELETE | `/api/v1/licenses/{id}` | License CRUD |
| GET/POST | `/api/v1/services` | List / Create services (catalog) |
| GET/PUT/DELETE | `/api/v1/services/{id}` | Service CRUD |
| GET/POST | `/api/v1/customers/{id}/services` | List / Create subscriptions |
| GET/PUT/DELETE | `/api/v1/customer-services/{id}` | Subscription CRUD |
| GET/POST | `/api/v1/hardware-categories` | List / Create categories |
| GET/PUT/DELETE | `/api/v1/hardware-categories/{id}` | Category CRUD |
| GET | `/api/v1/hardware-categories/{id}/fields` | List fields of a category |
| GET | `/health` | Health check (returns version + git hash) |

**HTTP Status Codes:**
- `200` — GET, UPDATE
- `201` — CREATE
- `204` — DELETE
- `400` — Invalid input
- `404` — Not found (`pgx.ErrNoRows`)
- `409` — Unique violation (`23505`) or FK violation (`23503`)
- `500` — Unexpected error

---

## 6. Frontend (HTMX + Go Templates)

### Template System
- `layout.html` — HTML shell; all pages inject content via `{{define "content"}}`
- Page templates: `entity/list.html`, `entity/form.html`, `entity/detail.html`
- Partials: `entity/list_rows.html` — defines named template `entity_rows` for HTMX swaps

### HTMX Patterns

**Live search with debounce:**
```html
<input type="search" name="q"
       hx-get="/customers/rows"
       hx-trigger="keyup changed delay:300ms"
       hx-target="#customer-rows">
<tbody id="customer-rows">{{template "customer_rows" .Customers}}</tbody>
```

**Forms with JSON submit:**
```html
<form id="customer-form"
      hx-post="/api/v1/customers"  <!-- or hx-put for edit -->
      hx-ext="json-enc"
      hx-target="#form-message"
      hx-swap="innerHTML">
  <div id="form-message" aria-live="polite"></div>
  <!-- fields -->
</form>
<script>
  document.body.addEventListener('htmx:afterRequest', function(evt) {
    if (evt.detail.elt.id === 'customer-form' && evt.detail.successful) {
      var r = JSON.parse(evt.detail.xhr.responseText);
      if (r.id) window.location.href = '/customers/' + r.id;
    }
  });
  document.body.addEventListener('htmx:responseError', function(evt) {
    if (evt.detail.elt.id === 'customer-form') {
      var msg = 'Error saving';
      try { var r = JSON.parse(evt.detail.xhr.responseText); if (r.error) msg = r.error; } catch(e) {}
      document.getElementById('form-message').innerHTML =
        '<div class="alert alert-error" role="alert">' + msg + '</div>';
    }
  });
</script>
```

### Page Routes (HTML)
```
GET /                          → Home
GET /customers                 → Customer list
GET /customers/rows            → HTMX partial: table rows
GET /customers/new             → New form
GET /customers/{id}            → Detail view
GET /customers/{id}/edit       → Edit form
GET /customers/{id}/assets     → Customer's asset list
GET /assets/{id}               → Asset detail
GET /assets/{id}/edit          → Asset edit form
... (same pattern for users, services, licenses, categories)
```

### CSS Design System (`web/static/css/style.css`)

CSS variables:
```css
--bg: #f8f9fa; --surface: #ffffff; --text: #212529; --muted: #6c757d;
--primary: #0d6efd; --danger: #dc3545; --success: #198754;
--border: #dee2e6; --radius: 6px;
```

Classes: `.container`, `.card`, `.section`, `.page-header`, `.btn`, `.btn-primary`, `.btn-danger`, `.btn-sm`, `.form-group`, `.search-bar`, `.alert`, `.alert-success`, `.alert-error`, `.flex`, `.gap-1`, `.text-muted`, `.text-right`, `.mt-1`, `.mb-1`

Semantic HTML: `<header>`, `<search>`, `<nav>`, `<main>`, `<section>`, `<dl>/<dt>/<dd>` for label/value pairs on detail pages.

### Template Functions
| Function | Output |
|---|---|
| `formatDate time.Time` | `"02.01.2006"` (DD.MM.YYYY) |
| `formatDateTime time.Time` | `"02.01.2006 15:04"` |
| `formatPgDate pgtype.Date` | `"02.01.2006"` or `"—"` |
| `pgText pgtype.Text` | String value or `""` |
| `uuidStr pgtype.UUID` | UUID as lowercase hex string |
| `default def val` | val if non-empty, otherwise def |
| `lower string` | Lowercase string |
| `map` | Map lookup with empty string fallback |

---

## 7. Go Code Conventions

### Handler Pattern
```go
type CustomerHandler struct { repo *repository.CustomerRepository }

func NewCustomerHandler(repo *repository.CustomerRepository) *CustomerHandler
func (h *CustomerHandler) RegisterRoutes(mux *http.ServeMux)
// Methods are unexported: list, create, get, update, delete
```

All page routes live in a single `PageHandler` struct in `page.go`.

Method naming: `entityAction` — e.g. `customerList`, `customerForm`, `customerDetail`.

### Repository Pattern
```go
type CustomerRepository struct { pool *pgxpool.Pool }

func (r *CustomerRepository) Create(ctx, input) (model.Customer, error)
func (r *CustomerRepository) GetByID(ctx, id pgtype.UUID) (model.Customer, error)
func (r *CustomerRepository) List(ctx, params model.ListParams) ([]model.Customer, error)
func (r *CustomerRepository) Update(ctx, id, input) (model.Customer, error)
func (r *CustomerRepository) Delete(ctx, id) error
```

SQL rules:
- `INSERT ... RETURNING id, ...` — returns the full entity after write
- `COALESCE($N, column)` for partial updates
- `pgx.CollectRows(rows, pgx.RowToStructByPos[model.Entity])` for result sets
- Positional parameters `$1, $2, $3` — never string interpolation
- Search via `ILIKE '%' || $1 || '%'`

### Model Pattern
```go
type Customer struct {
    ID           pgtype.UUID `json:"id"`
    Name         string      `json:"name"`
    ContactEmail pgtype.Text `json:"contact_email"`
    CreatedAt    time.Time   `json:"created_at"`
}

type CreateCustomerInput struct {
    Name         string  `json:"name"`
    ContactEmail *string `json:"contact_email,omitempty"`
}

type UpdateCustomerInput struct {
    Name         *string `json:"name,omitempty"`  // all fields are pointers → partial update
    ContactEmail *string `json:"contact_email,omitempty"`
}
```

Type mapping: `UUID` → `pgtype.UUID`, nullable TEXT → `pgtype.Text`, nullable DATE → `pgtype.Date`, JSONB → `json.RawMessage`, TIMESTAMPTZ → `time.Time`.

### Shared Helpers
- `writeJSON(w, status, data)`
- `writeError(w, status, "message")`
- `decodeJSON(r, &dst)` — uses `DisallowUnknownFields()`
- `parseUUID(r.PathValue("id"))` → `pgtype.UUID, error`
- `paginationParams(r)` → `model.ListParams`
- `isUniqueViolation(err)` — PG error code `23505`
- `isFKViolation(err)` — PG error code `23503`
- `validateFieldValues(rawValues, fields)` — validates JSONB field_values against field definitions

---

## 8. Deployment

### docker-compose.yml
```yaml
services:
  app:
    build: .
    ports: ["8080:8080"]
    environment:
      DATABASE_URL: postgres://argus:argus@db:5432/argus?sslmode=disable
    depends_on: [db]
  db:
    image: postgres:18.3-alpine
    environment:
      POSTGRES_USER: argus
      POSTGRES_PASSWORD: argus
      POSTGRES_DB: argus
    volumes: [postgres_data:/var/lib/postgresql/data]
```

### Dockerfile (Multi-Stage)
```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o argus ./cmd/server

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/argus .
COPY --from=builder /app/web ./web
COPY --from=builder /app/db ./db
CMD ["./argus"]
```

### Environment Variables
- `DATABASE_URL` — PostgreSQL connection string (required)
- `PORT` — HTTP listen port (default: 8080)
- `LOG_LEVEL` — logging verbosity

### Migrations
Run **automatically on application startup** via `internal/database/migrate.go` (goose embedded).

---

## 9. Known Gotchas & Learnings

1. **Go html/template + ParseFiles:** When multiple files define the same template name (`"content"`), only the last parsed file's definition survives. Fix: create a separate `*template.Template` instance per page (layout + partials + one page file).

2. **Goose + PL/pgSQL:** `-- +goose StatementBegin` / `-- +goose StatementEnd` are required around any SQL statement containing semicolons in its body (functions, DO blocks). Without these, goose splits on `;` and sends broken SQL.

3. **pgx.RowToStructByPos:** Struct fields must match the SELECT column order exactly.

4. **ON DELETE RESTRICT:** Delete handlers must catch FK violations (`23503`) and return `409 Conflict`, not `500`.

5. **ON CONFLICT DO NOTHING:** Always use explicit `ON CONFLICT (id) DO NOTHING` — bare `ON CONFLICT DO NOTHING` silently swallows any constraint violation, including unexpected ones.

6. **Seed data with FK references:** When referenced row UUIDs are not fixed, use a subquery: `(SELECT id FROM hardware_categories WHERE name = 'Laptop')`.

7. **Docker Compose + DB rename:** Run `docker compose down -v` when DB name/user changes — the old volume is incompatible.

---

## 10. Seed Data (Development Fixtures)

**3 customers:** Müller AG, Schmidt & Partner, TechStart GmbH

**Hardware categories:** Laptop, Server, Printer, Monitor, Network Device — each with typical typed fields

**Category fields example (Laptop):** Hostname (text, required), Operating System (text, required), RAM GB (number), Storage GB (number)

**Assets:** Lenovo ThinkPads, Dell servers, HP printers, Dell monitors, etc. — with `field_values` as JSONB

**Licenses:** Microsoft 365, Adobe Creative Cloud, JetBrains, Slack — some linked via `user_assignment_id`

**Customer services:** with `customizations` JSONB e.g. `{"sla": "8x5", "response_time_hours": 4}`

---

## 11. Post-MVP (explicitly out of scope)

- Authentication & Authorization (API keys, JWT, OAuth2/OIDC)
- License expiration notifications
- Reports & dashboards (PostgreSQL views)
- Full-text search (`tsvector`)
- Audit log
- Frontend upgrade (React/Svelte/Vue)
