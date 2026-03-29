# Naming & Structure Conventions

Consistent naming across the entire project.

## Directory Structure

```
cmd/server/main.go          # Entry point — wires everything together
internal/handler/            # HTTP handlers (API + page)
internal/repository/         # SQL data access via pgx
internal/model/              # Plain Go structs
internal/importer/           # CSV import/export engine, configs, FK resolver
internal/database/           # Connection pool + migration runner
internal/middleware/          # HTTP middleware (logging)
db/migrations/               # goose SQL migration files
web/templates/               # Go HTML templates
web/static/                  # CSS, JS assets
```

## Go Naming

### Packages

- `handler`, `repository`, `model`, `database`, `middleware`, `importer` — singular nouns
- No `pkg/`, `util/`, `common/`, `helpers/` packages

### Files

- One entity per file: `customer.go`, `asset.go`, `user.go`
- Shared utilities named by purpose: `json.go`, `params.go`, `template.go`, `field_validation.go`
- Test files: `field_validation_test.go` (standard `_test.go` suffix)

### Types

| Pattern | Example |
|---|---|
| Entity struct | `Customer`, `Asset`, `HardwareCategory` |
| Create input | `CreateCustomerInput`, `CreateAssetInput` |
| Update input | `UpdateCustomerInput`, `UpdateAssetInput` |
| Response wrapper | `AssetResponse` |
| Repository | `CustomerRepository`, `AssetRepository` |
| API Handler | `CustomerHandler`, `AssetHandler` |
| Page Handler | `PageHandler` (single, shared) |
| Template Engine | `TemplateEngine` (single) |
| Import Engine | `Engine` (in `importer` package) |
| Import Exporter | `Exporter` (in `importer` package) |
| Import Registry | `Registry` (in `importer` package) |
| Import Handler | `ImportHandler` |
| Entity Config | `EntityConfig`, `ColumnConfig`, `FKConfig` |

### Constructors

```go
func NewCustomerRepository(pool *pgxpool.Pool) *CustomerRepository
func NewCustomerHandler(repo *repository.CustomerRepository) *CustomerHandler
func NewPageHandler(tmpl *TemplateEngine, repos...) *PageHandler
func NewTemplateEngine(templateDir string) (*TemplateEngine, error)
func NewImportHandler(engine, exporter, registry) *ImportHandler
func importer.NewEngine(pool, registry) *Engine
func importer.NewExporter(pool, registry) *Exporter
func importer.NewRegistry() *Registry
```

### Methods

- Repository: `Create`, `GetByID`, `List`, `Update`, `Delete`, `ListByCustomer`, `ListFields`
- API Handler (unexported): `list`, `create`, `get`, `update`, `delete`
- Page Handler (unexported): `entityAction` — e.g., `customerList`, `customerListRows`, `customerForm`, `customerEditForm`, `customerDetail`
- Route registration: `RegisterRoutes(mux *http.ServeMux)` on every handler

## SQL Naming

- Tables: `customers`, `users`, `assets` (plural, snake_case)
- Columns: `snake_case` — `contact_email`, `created_at`, `field_values`
- Foreign keys: `entity_id` — `customer_id`, `category_id`
- Triggers: `trg_entity_updated_at` — `trg_customers_updated_at`
- Indexes: `idx_entity_column` — `idx_assets_metadata`
- Constraints: PostgreSQL auto-naming or explicit `chk_entity_rule`

## Template Naming

- Page templates: `entity/action.html` — `customers/list.html`, `users/form.html`
- Partials: `entity/list_rows.html` — defines named template `entity_rows`
- Template keys (Go): `"customers/list"`, `"users/form"` (dir/name without .html)
- Partial names (Go): `"customer_rows"`, `"user_rows"` (underscore, not slash)

## JSON Field Naming

All JSON uses `snake_case` consistently:
- API responses: `{"id": "...", "contact_email": "...", "created_at": "..."}`
- Error responses: `{"error": "message"}`
- HTMX form fields: `name="contact_email"` matches JSON key

## URL Naming

- API: `/api/v1/customers`, `/api/v1/customers/{id}`
- Import: `POST /api/v1/import/{entity}`, `GET /api/v1/import/{entity}/template`
- Export: `GET /api/v1/export/{entity}`
- Pages: `/customers`, `/customers/new`, `/customers/{id}`, `/customers/{id}/edit`, `/import`
- Partials: `/customers/rows` (for HTMX)
- Nested: `/customers/{customerId}/assets/{assetId}`
- Query params: `?limit=&offset=&q=&type=`

## Business Language

All code, comments, commit messages, UI labels, and documentation must be in **English**.
