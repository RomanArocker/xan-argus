# XAN-Pythia — Design Specification

## Overview

XAN-Pythia is a multi-tenant asset, user, license, and service management system for an IT services business. It serves as a central database and lookup tool to track what each customer has, who works there, what licenses they hold, and which services they subscribe to.

**MVP Scope:** CRUD operations, search, and overview. No active features (notifications, reports, dashboards) in the initial version.

## Goals

- Single source of truth for all customer-related data (assets, users, licenses, services)
- API-first design enabling future integrations and frontend replacements
- Leverage PostgreSQL capabilities directly — constraints, JSONB, triggers — instead of reimplementing in Go
- Simplicity first: no unnecessary abstractions, no premature optimization

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Backend | Go (standard library + pgx) |
| Database | PostgreSQL 18.3 |
| Frontend | HTMX + Go HTML Templates |
| Deployment | Docker Compose (Go app + PostgreSQL) |
| Migrations | goose (pure SQL) |
| Auth | Deferred to post-MVP |

## Data Model

### Entity Relationship Diagram

See `docs/data-model.mmd` for the full Mermaid ER diagram.

### Tables

#### customers

The customer company/organization.

| Column | Type | Constraints |
|--------|------|-------------|
| id | UUID | PK, DEFAULT gen_random_uuid() |
| name | TEXT | NOT NULL, UNIQUE |
| contact_email | TEXT | |
| notes | TEXT | |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() |
| updated_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() |

#### users

Core identity of a person — either a customer's employee or an internal staff member.

| Column | Type | Constraints |
|--------|------|-------------|
| id | UUID | PK, DEFAULT gen_random_uuid() |
| type | TEXT | NOT NULL, CHECK (type IN ('customer_staff', 'internal_staff')) |
| first_name | TEXT | NOT NULL |
| last_name | TEXT | NOT NULL |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() |
| updated_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() |

#### user_assignments

Links a user to a customer with per-company role, contact details. A user can be assigned to multiple customers.

| Column | Type | Constraints |
|--------|------|-------------|
| id | UUID | PK, DEFAULT gen_random_uuid() |
| user_id | UUID | NOT NULL, FK → users(id) ON DELETE RESTRICT |
| customer_id | UUID | NOT NULL, FK → customers(id) ON DELETE RESTRICT |
| role | TEXT | |
| email | TEXT | |
| phone | TEXT | |
| notes | TEXT | |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() |
| updated_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() |

**Constraints:**
- UNIQUE(user_id, customer_id) — one assignment per user per customer

#### assets

Hardware or software assets owned by a customer.

| Column | Type | Constraints |
|--------|------|-------------|
| id | UUID | PK, DEFAULT gen_random_uuid() |
| customer_id | UUID | NOT NULL, FK → customers(id) ON DELETE RESTRICT |
| type | TEXT | NOT NULL, CHECK (type IN ('hardware', 'software')) |
| name | TEXT | NOT NULL |
| description | TEXT | |
| metadata | JSONB | DEFAULT '{}' |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() |
| updated_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() |

**Notes:**
- `metadata` stores flexible key-value data: serial numbers, IP addresses, software versions, etc.
- GIN index on `metadata` for efficient JSONB queries

#### licenses

Third-party software licenses managed on behalf of a customer.

| Column | Type | Constraints |
|--------|------|-------------|
| id | UUID | PK, DEFAULT gen_random_uuid() |
| customer_id | UUID | NOT NULL, FK → customers(id) ON DELETE RESTRICT |
| user_assignment_id | UUID | FK → user_assignments(id) ON DELETE SET NULL, nullable |
| product_name | TEXT | NOT NULL |
| license_key | TEXT | |
| quantity | INTEGER | DEFAULT 1 |
| valid_from | DATE | |
| valid_until | DATE | |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() |
| updated_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() |

**Constraints:**
- Trigger or CHECK function to ensure `user_assignment.customer_id = licenses.customer_id` when `user_assignment_id` is set
- Licenses without `user_assignment_id` are company-level licenses (e.g., certain M365 licenses)

#### services

Catalog of services offered by the business. These are templates, not customer-specific.

| Column | Type | Constraints |
|--------|------|-------------|
| id | UUID | PK, DEFAULT gen_random_uuid() |
| name | TEXT | NOT NULL |
| description | TEXT | |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() |
| updated_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() |

#### customer_services

Instance of a service for a specific customer, with optional customizations.

| Column | Type | Constraints |
|--------|------|-------------|
| id | UUID | PK, DEFAULT gen_random_uuid() |
| customer_id | UUID | NOT NULL, FK → customers(id) ON DELETE RESTRICT |
| service_id | UUID | NOT NULL, FK → services(id) ON DELETE RESTRICT |
| customizations | JSONB | DEFAULT '{}' |
| notes | TEXT | |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() |
| updated_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() |

**Constraints:**
- UNIQUE(customer_id, service_id) — one subscription per service per customer

**Notes:**
- `customizations` stores scope modifications, e.g., `{"byod": true, "smartphones": true}`
- GIN index on `customizations` for queries

## API Design

### Principles

- REST JSON API under `/api/v1/`
- Resource-oriented routes
- No authentication in MVP
- Standard HTTP status codes
- Pagination via `?limit=` and `?offset=` query parameters
- Error responses: `{"error": "human-readable message"}` with appropriate HTTP status code

### Routes

| Method | Path | Description |
|--------|------|-------------|
| GET/POST | `/api/v1/customers` | List / Create customers |
| GET/PUT/DELETE | `/api/v1/customers/{id}` | Get / Update / Delete customer |
| GET/POST | `/api/v1/customers/{id}/assets` | List / Create assets for customer |
| GET/PUT/DELETE | `/api/v1/assets/{id}` | Get / Update / Delete asset |
| GET/POST | `/api/v1/users` | List / Create users |
| GET/PUT/DELETE | `/api/v1/users/{id}` | Get / Update / Delete user |
| GET/POST | `/api/v1/customers/{id}/user-assignments` | List / Create assignments for customer |
| GET/PUT/DELETE | `/api/v1/user-assignments/{id}` | Get / Update / Delete assignment |
| GET/POST | `/api/v1/customers/{id}/licenses` | List / Create licenses for customer |
| GET/PUT/DELETE | `/api/v1/licenses/{id}` | Get / Update / Delete license |
| GET/POST | `/api/v1/services` | List / Create services (catalog) |
| GET/PUT/DELETE | `/api/v1/services/{id}` | Get / Update / Delete service |
| GET/POST | `/api/v1/customers/{id}/services` | List / Create customer service subscriptions |
| GET/PUT/DELETE | `/api/v1/customer-services/{id}` | Get / Update / Delete customer service |

## Project Structure

```
xan-pythia/
├── cmd/
│   └── server/
│       └── main.go              # Entry point
├── internal/
│   ├── handler/                  # HTTP handlers (routing, request/response)
│   ├── repository/               # Database access (raw SQL via pgx)
│   ├── model/                    # Go structs for entities
│   └── middleware/               # Logging, CORS, etc. (auth later)
├── db/
│   └── migrations/               # Pure SQL migration files
├── web/
│   ├── templates/                # Go HTML templates
│   └── static/                   # CSS, HTMX, static assets
├── Dockerfile                    # Multi-stage build
├── docker-compose.yml            # App + PostgreSQL
├── go.mod
└── go.sum
```

### Layer Responsibilities

- **handler:** Parse HTTP requests, call repository, return JSON or render templates. No business logic beyond simple validation.
- **repository:** Execute SQL queries via pgx, return model structs. One file per entity.
- **model:** Plain Go structs matching database tables. No methods, no logic.
- **No service layer** in MVP. Handler calls repository directly. A service layer can be introduced when business logic grows beyond simple CRUD.

## Deployment

### Docker Compose

Two containers:
1. **app** — Go binary (multi-stage build: `golang:1.23` build stage → `alpine` runtime)
2. **db** — PostgreSQL 18.3 with persistent volume

### Configuration

All configuration via environment variables (12-Factor):
- `DATABASE_URL` — PostgreSQL connection string
- `PORT` — HTTP listen port (default: 8080)
- `LOG_LEVEL` — Logging verbosity

### Migrations

Run automatically on application startup. Pure SQL files managed by goose.

## PostgreSQL Design Principles

- **Foreign keys with ON DELETE RESTRICT** — prevent accidental cascading deletes
- **CHECK constraints** for enums (user type, asset type) — enforced at database level
- **GIN indexes** on JSONB columns (assets.metadata, customer_services.customizations)
- **License consistency trigger** — on INSERT/UPDATE of `licenses`, when `user_assignment_id` is NOT NULL, verify that the referenced `user_assignments.customer_id` matches `licenses.customer_id`. Reject with error if mismatched.
- **`set_updated_at()` trigger function** — shared trigger applied to all 7 tables, sets `updated_at = now()` before each UPDATE
- **UUIDs** as primary keys (gen_random_uuid())

## Future Considerations (Post-MVP)

These are explicitly out of scope for the initial build but inform architectural decisions:

- **Authentication & Authorization** — API keys, JWT, or OAuth2/OIDC
- **License expiration notifications** — based on `valid_until` field
- **Reports & Dashboards** — PostgreSQL views and aggregations
- **Search** — full-text search via PostgreSQL `tsvector`
- **Audit log** — track who changed what
- **Frontend upgrade** — swap HTMX for React/Svelte/Vue against the same API
