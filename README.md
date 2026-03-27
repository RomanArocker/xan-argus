# XAN-Pythia

Multi-tenant asset, user, license, and service management system for IT services businesses.

## Features

- **Customer Management** — track customers with contact details and notes
- **User Management** — manage internal staff and customer employees
- **Asset Tracking** — hardware and software assets per customer with flexible JSONB metadata
- **License Management** — track licenses, keys, quantities, and validity periods with customer-assignment consistency enforcement
- **Service Catalog** — define services and subscribe customers with custom configurations
- **User Assignments** — assign users to customers with roles and contact info

### Frontend

- HTMX-powered web UI with live search, inline delete, and create/edit forms
- Customer detail page with overview of all related entities
- No page reloads — HTMX handles partial updates

### API

- RESTful JSON API under `/api/v1/`
- Standard pagination (`?limit=`, `?offset=`)
- Search (`?q=`) and type filtering (`?type=`)
- Proper error responses: 400 (validation), 404 (not found), 409 (conflict/FK violation)

## Tech Stack

| Component  | Technology                    |
|------------|-------------------------------|
| Backend    | Go (stdlib + pgx)             |
| Database   | PostgreSQL 18.3               |
| Frontend   | HTMX 2.0 + Go HTML Templates |
| Migrations | goose (pure SQL, run on startup) |
| Deployment | Docker Compose                |

## Quick Start

```bash
docker compose up --build
```

Open [http://localhost:8080](http://localhost:8080)

### Environment Variables

| Variable       | Description                    | Default |
|----------------|--------------------------------|---------|
| `DATABASE_URL` | PostgreSQL connection string   | —       |
| `PORT`         | HTTP listen port               | `8080`  |
| `LOG_LEVEL`    | Logging verbosity              | —       |

## Project Structure

```
cmd/server/main.go           # Entry point
internal/
  handler/                    # HTTP handlers (API + web pages)
    customer.go               # REST API: /api/v1/customers
    user.go                   # REST API: /api/v1/users
    service.go                # REST API: /api/v1/services
    asset.go                  # REST API: /api/v1/assets
    license.go                # REST API: /api/v1/licenses
    user_assignment.go        # REST API: /api/v1/user-assignments
    customer_service.go       # REST API: /api/v1/customer-services
    page.go                   # Web frontend routes
    template.go               # Template engine
    json.go                   # JSON helpers
    params.go                 # UUID parsing
  repository/                 # Database access (SQL via pgx)
  model/                      # Go structs for entities
  database/                   # Connection pool + migration runner
  middleware/                  # Logging middleware
db/migrations/                # goose SQL migrations
web/
  templates/                  # Go HTML templates
    layout.html               # Base layout (nav, head)
    customers/                # Customer pages
    users/                    # User pages
    services/                 # Service pages
  static/                     # CSS, HTMX JS
```

## API Endpoints

| Method         | Path                                      | Description                        |
|----------------|-------------------------------------------|------------------------------------|
| GET/POST       | `/api/v1/customers`                       | List / Create customers            |
| GET/PUT/DELETE  | `/api/v1/customers/{id}`                 | Get / Update / Delete customer     |
| GET/POST       | `/api/v1/customers/{id}/assets`           | List / Create assets for customer  |
| GET/PUT/DELETE  | `/api/v1/assets/{id}`                    | Get / Update / Delete asset        |
| GET/POST       | `/api/v1/users`                           | List / Create users                |
| GET/PUT/DELETE  | `/api/v1/users/{id}`                     | Get / Update / Delete user         |
| GET/POST       | `/api/v1/customers/{id}/user-assignments` | List / Create assignments          |
| GET/PUT/DELETE  | `/api/v1/user-assignments/{id}`          | Get / Update / Delete assignment   |
| GET/POST       | `/api/v1/customers/{id}/licenses`         | List / Create licenses             |
| GET/PUT/DELETE  | `/api/v1/licenses/{id}`                  | Get / Update / Delete license      |
| GET/POST       | `/api/v1/services`                        | List / Create services (catalog)   |
| GET/PUT/DELETE  | `/api/v1/services/{id}`                  | Get / Update / Delete service      |
| GET/POST       | `/api/v1/customers/{id}/services`         | List / Create service subscriptions|
| GET/PUT/DELETE  | `/api/v1/customer-services/{id}`         | Get / Update / Delete subscription |

## Database

7 tables with full referential integrity:

- `customers` — root entity (name is unique)
- `users` — internal or customer staff
- `user_assignments` — links users to customers (unique per pair)
- `assets` — hardware/software per customer (JSONB metadata with GIN index)
- `licenses` — per customer, optionally assigned to a user (consistency trigger enforces matching customer)
- `services` — service catalog
- `customer_services` — subscriptions (unique per customer-service pair, JSONB customizations with GIN index)

All tables have `set_updated_at()` triggers and UUID primary keys. Foreign keys use `ON DELETE RESTRICT`.

## Development

```bash
# Prerequisites: Go 1.26+, Docker

# Run locally (needs PostgreSQL)
export DATABASE_URL="postgres://user:pass@localhost:5432/xanpythia?sslmode=disable"
go run ./cmd/server/

# Static analysis
go vet ./...
golangci-lint run ./...

# Tests (needs test database)
export TEST_DATABASE_URL="postgres://user:pass@localhost:5432/xanpythia_test?sslmode=disable"
go test ./...

# Format
gofmt -w .
```

## License

Proprietary — internal use only.
