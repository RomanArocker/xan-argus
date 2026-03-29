# Handler Conventions

Handlers live in `internal/handler/`. They parse HTTP requests, call repositories directly (no service layer), and return JSON or render templates.

## Structure

Each entity gets two handler types:

1. **API Handler** (`customer.go`, `user.go`, etc.) — JSON CRUD under `/api/v1/`
2. **Page Handler** (`page.go`) — HTML pages, all routes in one `PageHandler` struct

### API Handler Pattern

```go
type CustomerHandler struct {
    repo *repository.CustomerRepository
}

func NewCustomerHandler(repo *repository.CustomerRepository) *CustomerHandler {
    return &CustomerHandler{repo: repo}
}

func (h *CustomerHandler) RegisterRoutes(mux *http.ServeMux) {
    mux.HandleFunc("GET /api/v1/customers", h.list)
    mux.HandleFunc("POST /api/v1/customers", h.create)
    mux.HandleFunc("GET /api/v1/customers/{id}", h.get)
    mux.HandleFunc("PUT /api/v1/customers/{id}", h.update)
    mux.HandleFunc("DELETE /api/v1/customers/{id}", h.delete)
}
```

- One struct per entity with a `repo` field
- Constructor: `NewEntityHandler(repo)`
- `RegisterRoutes(mux *http.ServeMux)` registers all routes
- Methods are lowercase (unexported): `list`, `create`, `get`, `update`, `delete`
- Route pattern: `"METHOD /api/v1/{entity}"` using Go 1.22+ ServeMux syntax

### Page Handler Pattern

All page routes live in a single `PageHandler` struct in `page.go`:

```go
func (h *PageHandler) RegisterRoutes(mux *http.ServeMux) {
    mux.HandleFunc("GET /{$}", h.home)
    mux.HandleFunc("GET /customers", h.customerList)
    mux.HandleFunc("GET /customers/rows", h.customerListRows)
    mux.HandleFunc("GET /customers/new", h.customerForm)
    mux.HandleFunc("GET /customers/{id}", h.customerDetail)
    mux.HandleFunc("GET /customers/{id}/edit", h.customerEditForm)
}
```

Page route naming:
- List page: `GET /entity`
- HTMX row partial: `GET /entity/rows`
- New form: `GET /entity/new`
- Detail: `GET /entity/{id}`
- Edit form: `GET /entity/{id}/edit`
- Nested resources: `GET /entity/{entityId}/sub/{subId}`

Page handler method naming: `entityAction` (e.g., `customerList`, `customerListRows`, `customerForm`, `customerEditForm`, `customerDetail`)

## Handler Flow

Every handler follows the same sequence:

```
Parse input → Validate → Call repo → Handle errors → Respond
```

### API Handler Example (create)

```go
func (h *CustomerHandler) create(w http.ResponseWriter, r *http.Request) {
    var input model.CreateCustomerInput
    if err := decodeJSON(r, &input); err != nil {
        writeError(w, http.StatusBadRequest, "invalid request body")
        return
    }
    if input.Name == "" {
        writeError(w, http.StatusBadRequest, "name is required")
        return
    }
    customer, err := h.repo.Create(r.Context(), input)
    if err != nil {
        if isUniqueViolation(err) {
            writeError(w, http.StatusConflict, "customer with this name already exists")
            return
        }
        writeError(w, http.StatusInternalServerError, "failed to create customer")
        return
    }
    writeJSON(w, http.StatusCreated, customer)
}
```

### Page Handler Example (list with HTMX rows)

```go
func (h *PageHandler) customerList(w http.ResponseWriter, r *http.Request) {
    params := paginationParams(r)
    customers, err := h.customerRepo.List(r.Context(), params)
    if err != nil {
        http.Error(w, "failed to load customers", http.StatusInternalServerError)
        return
    }
    h.tmpl.RenderPage(w, "customers/list", map[string]any{
        "Title":     "Customers",
        "Customers": customers,
        "Search":    params.Search,
    })
}

func (h *PageHandler) customerListRows(w http.ResponseWriter, r *http.Request) {
    params := paginationParams(r)
    customers, err := h.customerRepo.List(r.Context(), params)
    if err != nil {
        http.Error(w, "failed to load customers", http.StatusInternalServerError)
        return
    }
    h.tmpl.RenderPartial(w, "customer_rows", customers)
}
```

## Error Handling

### API Handlers

Use shared helpers from `json.go`:

- `writeJSON(w, status, data)` — success response
- `writeError(w, status, "message")` — error as `{"error": "message"}`
- `decodeJSON(r, &dst)` — decode with `DisallowUnknownFields()`

Status codes:
- `200 OK` — list, get, update
- `201 Created` — create
- `204 No Content` — delete / soft delete (no body)
- `400 Bad Request` — invalid JSON, missing required fields
- `404 Not Found` — `pgx.ErrNoRows`
- `409 Conflict` — FK violation (`isFKViolation`), unique violation (`isUniqueViolation`)
- `500 Internal Server Error` — unexpected errors

PostgreSQL error detection:

```go
if isUniqueViolation(err) {
    writeError(w, http.StatusConflict, "entity with this name already exists")
    return
}
if isFKViolation(err) {
    writeError(w, http.StatusConflict, "entity has dependent records")
    return
}
```

**Note:** `isFKViolation()` catches both hard FK violations and soft-delete guard trigger errors (both use PG error code `23503`). DELETE handlers work unchanged — the repository handles the soft delete internally.

### Page Handlers

Use `http.Error(w, "message", statusCode)` — no JSON wrapper.

## UUID Parsing

Always use `parseUUID()` from `params.go`:

```go
id, err := parseUUID(r.PathValue("id"))
if err != nil {
    writeError(w, http.StatusBadRequest, "invalid entity ID")
    return
}
```

## Pagination

Use `paginationParams(r)` to extract `?limit=&offset=&q=&type=` query params. Returns `model.ListParams` with defaults applied via `Normalize()`.

## Validation

- Required fields: inline checks in handler (`if input.Name == ""`)
- JSONB field values: use `validateFieldValues(rawValues, fields)` from `field_validation.go` — validates keys against field definition UUIDs and type-checks values (text, number, boolean, date as YYYY-MM-DD); empty string `""` is valid for optional date fields and skips format validation
- Database constraints: caught via `isFKViolation()` / `isUniqueViolation()` after repo call

## Do / Don't

- **Do** use `r.Context()` for all repo calls
- **Do** return early on errors (guard clause pattern)
- **Do** provide human-readable error messages to the client
- **Don't** create a service layer — handlers call repos directly
- **Don't** log errors in handlers — the logging middleware handles it
- **Don't** use `any` for JSON responses — always use typed structs or `map[string]any`
- **Don't** add new shared helpers unless the pattern appears in 3+ handlers
