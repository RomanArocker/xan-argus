# XAN-Pythia MVP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a multi-tenant asset/user/license/service management system with REST API and HTMX frontend.

**Architecture:** Go stdlib HTTP server with pgx for PostgreSQL. Three-layer structure: handler → repository → model (no service layer). PostgreSQL enforces constraints, triggers, and JSONB indexes. HTMX + Go HTML templates for frontend.

**Tech Stack:** Go 1.23, PostgreSQL 18.3, pgx v5, goose migrations, HTMX, Docker Compose

**Spec:** `docs/superpowers/specs/2026-03-27-xan-pythia-design.md`
**Data Model:** `docs/data-model.mmd`

---

## Phase 1: Foundation

### Task 1: Project Scaffolding

**Files:**
- Create: `go.mod`
- Create: `cmd/server/main.go`
- Create: `internal/model/`, `internal/handler/`, `internal/repository/`, `internal/middleware/`
- Create: `db/migrations/`
- Create: `web/templates/`, `web/static/`

- [ ] **Step 1: Initialize Go module**

```bash
go mod init github.com/xan-com/xan-pythia
```

- [ ] **Step 2: Create directory structure**

```bash
mkdir -p cmd/server internal/model internal/handler internal/repository internal/middleware db/migrations web/templates web/static
```

- [ ] **Step 3: Create minimal main.go**

```go
// cmd/server/main.go
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"ok"}`)
	})

	log.Printf("Starting server on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
```

- [ ] **Step 4: Verify it compiles**

```bash
go build ./cmd/server/
```

Expected: no errors, binary created.

- [ ] **Step 5: Commit**

```bash
git init
git add .
git commit -m "chore: scaffold project structure with health endpoint"
```

---

### Task 2: Docker Setup

**Files:**
- Create: `Dockerfile`
- Create: `docker-compose.yml`
- Create: `.dockerignore`

- [ ] **Step 1: Create Dockerfile**

```dockerfile
# Dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /xan-pythia ./cmd/server/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /xan-pythia /xan-pythia
EXPOSE 8080
CMD ["/xan-pythia"]
```

- [ ] **Step 2: Create .dockerignore**

```
.git
*.md
docs/
.claude/
```

- [ ] **Step 3: Create docker-compose.yml**

```yaml
# docker-compose.yml
services:
  db:
    image: postgres:18-alpine
    environment:
      POSTGRES_DB: xanpythia
      POSTGRES_USER: xanpythia
      POSTGRES_PASSWORD: xanpythia
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U xanpythia"]
      interval: 5s
      timeout: 3s
      retries: 5

  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: postgres://xanpythia:xanpythia@db:5432/xanpythia?sslmode=disable
      PORT: "8080"
    depends_on:
      db:
        condition: service_healthy

volumes:
  pgdata:
```

- [ ] **Step 4: Verify Docker build**

```bash
docker compose build
```

Expected: successful build.

- [ ] **Step 5: Commit**

```bash
git add Dockerfile .dockerignore docker-compose.yml
git commit -m "chore: add Docker and Docker Compose setup"
```

---

### Task 3: Database Connection & Migration Runner

**Files:**
- Create: `internal/database/database.go`
- Modify: `cmd/server/main.go`
- Modify: `go.mod` (new deps: pgx, goose)

- [ ] **Step 1: Add dependencies**

```bash
go get github.com/jackc/pgx/v5/pgxpool
go get github.com/pressly/goose/v3
```

- [ ] **Step 2: Create database connection helper**

```go
// internal/database/database.go
package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return pool, nil
}
```

- [ ] **Step 3: Create migration runner**

```go
// internal/database/migrate.go
package database

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func RunMigrations(databaseURL string, migrationsDir string) error {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("opening database for migrations: %w", err)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("setting goose dialect: %w", err)
	}

	if err := goose.Up(db, migrationsDir); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}

	return nil
}
```

- [ ] **Step 4: Update main.go to connect and migrate**

```go
// cmd/server/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/xan-com/xan-pythia/internal/database"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	// Run migrations
	if err := database.RunMigrations(databaseURL, "db/migrations"); err != nil {
		log.Fatalf("running migrations: %v", err)
	}
	log.Println("Migrations completed")

	// Connect pool
	ctx := context.Background()
	pool, err := database.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer pool.Close()
	log.Println("Database connected")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"ok"}`)
	})

	log.Printf("Starting server on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
```

- [ ] **Step 5: Verify it compiles**

```bash
go build ./cmd/server/
```

- [ ] **Step 6: Commit**

```bash
git add internal/database/ cmd/server/main.go go.mod go.sum
git commit -m "feat: add database connection and migration runner"
```

---

### Task 4: Migration — Base Tables & Triggers

**Files:**
- Create: `db/migrations/001_base_schema.sql`

This migration creates the `set_updated_at()` trigger function, the three independent tables (`customers`, `users`, `services`), and applies the trigger to each.

- [ ] **Step 1: Create migration file**

```sql
-- db/migrations/001_base_schema.sql

-- +goose Up

-- Shared trigger function for updated_at
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- customers
CREATE TABLE customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    contact_email TEXT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER trg_customers_updated_at
    BEFORE UPDATE ON customers
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- users
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type TEXT NOT NULL CHECK (type IN ('customer_staff', 'internal_staff')),
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- services (catalog)
CREATE TABLE services (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER trg_services_updated_at
    BEFORE UPDATE ON services
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE IF EXISTS services;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS customers;
DROP FUNCTION IF EXISTS set_updated_at;
```

- [ ] **Step 2: Verify migration runs**

```bash
docker compose up db -d
export DATABASE_URL="postgres://xanpythia:xanpythia@localhost:5432/xanpythia?sslmode=disable"
go run ./cmd/server/ &
# Check logs for "Migrations completed"
kill %1
```

- [ ] **Step 3: Commit**

```bash
git add db/migrations/001_base_schema.sql
git commit -m "feat: add base schema migration (customers, users, services)"
```

---

### Task 5: Migration — Dependent Tables, License Trigger & Indexes

**Files:**
- Create: `db/migrations/002_dependent_tables.sql`

- [ ] **Step 1: Create migration file**

```sql
-- db/migrations/002_dependent_tables.sql

-- +goose Up

-- user_assignments
CREATE TABLE user_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    role TEXT NOT NULL,
    email TEXT,
    phone TEXT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(user_id, customer_id)
);

CREATE TRIGGER trg_user_assignments_updated_at
    BEFORE UPDATE ON user_assignments
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- assets
CREATE TABLE assets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    type TEXT NOT NULL CHECK (type IN ('hardware', 'software')),
    name TEXT NOT NULL,
    description TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_assets_metadata ON assets USING GIN (metadata);

CREATE TRIGGER trg_assets_updated_at
    BEFORE UPDATE ON assets
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- licenses
CREATE TABLE licenses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    user_assignment_id UUID REFERENCES user_assignments(id) ON DELETE RESTRICT,
    product_name TEXT NOT NULL,
    license_key TEXT,
    quantity INTEGER NOT NULL DEFAULT 1,
    valid_from DATE,
    valid_until DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER trg_licenses_updated_at
    BEFORE UPDATE ON licenses
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- License consistency trigger: ensure user_assignment belongs to same customer
CREATE OR REPLACE FUNCTION check_license_customer_consistency()
RETURNS TRIGGER AS $$
DECLARE
    assignment_customer_id UUID;
BEGIN
    IF NEW.user_assignment_id IS NOT NULL THEN
        SELECT customer_id INTO assignment_customer_id
        FROM user_assignments
        WHERE id = NEW.user_assignment_id;

        IF assignment_customer_id IS NULL THEN
            RAISE EXCEPTION 'user_assignment_id % does not exist', NEW.user_assignment_id;
        END IF;

        IF assignment_customer_id != NEW.customer_id THEN
            RAISE EXCEPTION 'license customer_id (%) does not match user_assignment customer_id (%)',
                NEW.customer_id, assignment_customer_id;
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_licenses_customer_consistency
    BEFORE INSERT OR UPDATE ON licenses
    FOR EACH ROW EXECUTE FUNCTION check_license_customer_consistency();

-- customer_services
CREATE TABLE customer_services (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    service_id UUID NOT NULL REFERENCES services(id) ON DELETE RESTRICT,
    customizations JSONB DEFAULT '{}',
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(customer_id, service_id)
);

CREATE INDEX idx_customer_services_customizations ON customer_services USING GIN (customizations);

CREATE TRIGGER trg_customer_services_updated_at
    BEFORE UPDATE ON customer_services
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE IF EXISTS customer_services;
DROP TABLE IF EXISTS licenses;
DROP TABLE IF EXISTS assets;
DROP TABLE IF EXISTS user_assignments;
DROP FUNCTION IF EXISTS check_license_customer_consistency;
```

- [ ] **Step 2: Verify migration runs**

```bash
export DATABASE_URL="postgres://xanpythia:xanpythia@localhost:5432/xanpythia?sslmode=disable"
go run ./cmd/server/ &
# Check logs for "Migrations completed"
kill %1
```

- [ ] **Step 3: Commit**

```bash
git add db/migrations/002_dependent_tables.sql
git commit -m "feat: add dependent tables, license trigger, GIN indexes"
```

---

## Phase 2: Core Entities (Customers, Users, Services)

These three entities have no foreign-key dependencies on other app tables, so they can be built first.

### Task 6: Models Package

**Files:**
- Create: `internal/model/customer.go`
- Create: `internal/model/user.go`
- Create: `internal/model/service.go`
- Create: `internal/model/pagination.go`

- [ ] **Step 1: Create pagination helper**

```go
// internal/model/pagination.go
package model

const (
	DefaultLimit = 20
	MaxLimit     = 100
)

type ListParams struct {
	Limit  int
	Offset int
	Search string // optional name/text search (?q=)
	Filter string // optional type filter (?type=)
}

func (p *ListParams) Normalize() {
	if p.Limit <= 0 {
		p.Limit = DefaultLimit
	}
	if p.Limit > MaxLimit {
		p.Limit = MaxLimit
	}
	if p.Offset < 0 {
		p.Offset = 0
	}
}
```

- [ ] **Step 2: Create customer model**

```go
// internal/model/customer.go
package model

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type Customer struct {
	ID           pgtype.UUID        `json:"id"`
	Name         string             `json:"name"`
	ContactEmail pgtype.Text        `json:"contact_email"`
	Notes        pgtype.Text        `json:"notes"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
}

type CreateCustomerInput struct {
	Name         string  `json:"name"`
	ContactEmail *string `json:"contact_email,omitempty"`
	Notes        *string `json:"notes,omitempty"`
}

type UpdateCustomerInput struct {
	Name         *string `json:"name,omitempty"`
	ContactEmail *string `json:"contact_email,omitempty"`
	Notes        *string `json:"notes,omitempty"`
}
```

- [ ] **Step 3: Create user model**

```go
// internal/model/user.go
package model

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type User struct {
	ID        pgtype.UUID `json:"id"`
	Type      string      `json:"type"`
	FirstName string      `json:"first_name"`
	LastName  string      `json:"last_name"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

type CreateUserInput struct {
	Type      string `json:"type"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type UpdateUserInput struct {
	Type      *string `json:"type,omitempty"`
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
}
```

- [ ] **Step 4: Create service model**

```go
// internal/model/service.go
package model

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type Service struct {
	ID          pgtype.UUID `json:"id"`
	Name        string      `json:"name"`
	Description pgtype.Text `json:"description"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type CreateServiceInput struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

type UpdateServiceInput struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}
```

- [ ] **Step 5: Verify it compiles**

```bash
go vet ./internal/model/...
```

- [ ] **Step 6: Commit**

```bash
git add internal/model/
git commit -m "feat: add model structs for customers, users, services"
```

---

### Task 7: Customer Repository

**Files:**
- Create: `internal/repository/customer.go`
- Create: `internal/repository/customer_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// internal/repository/customer_test.go
package repository_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xan-com/xan-pythia/internal/model"
	"github.com/xan-com/xan-pythia/internal/repository"
)

func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("connecting to test db: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	return pool
}

func TestCustomerCRUD(t *testing.T) {
	pool := setupTestDB(t)
	repo := repository.NewCustomerRepository(pool)
	ctx := context.Background()

	// Create
	input := model.CreateCustomerInput{Name: "Acme Corp"}
	customer, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if customer.Name != "Acme Corp" {
		t.Errorf("Name = %q, want %q", customer.Name, "Acme Corp")
	}
	if !customer.ID.Valid {
		t.Error("ID should be valid")
	}

	// Get
	got, err := repo.GetByID(ctx, customer.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "Acme Corp" {
		t.Errorf("GetByID Name = %q, want %q", got.Name, "Acme Corp")
	}

	// Update
	newName := "Acme Inc"
	updated, err := repo.Update(ctx, customer.ID, model.UpdateCustomerInput{Name: &newName})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "Acme Inc" {
		t.Errorf("Updated Name = %q, want %q", updated.Name, "Acme Inc")
	}

	// List
	customers, err := repo.List(ctx, model.ListParams{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(customers) == 0 {
		t.Error("List returned no customers")
	}

	// Delete
	if err := repo.Delete(ctx, customer.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = repo.GetByID(ctx, customer.ID)
	if err == nil {
		t.Error("GetByID after Delete should return error")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/repository/... -v -run TestCustomerCRUD
```

Expected: compilation errors (repository package doesn't exist yet).

- [ ] **Step 3: Implement customer repository**

```go
// internal/repository/customer.go
package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xan-com/xan-pythia/internal/model"
)

type CustomerRepository struct {
	pool *pgxpool.Pool
}

func NewCustomerRepository(pool *pgxpool.Pool) *CustomerRepository {
	return &CustomerRepository{pool: pool}
}

func (r *CustomerRepository) Create(ctx context.Context, input model.CreateCustomerInput) (model.Customer, error) {
	var c model.Customer
	err := r.pool.QueryRow(ctx,
		`INSERT INTO customers (name, contact_email, notes)
		 VALUES ($1, $2, $3)
		 RETURNING id, name, contact_email, notes, created_at, updated_at`,
		input.Name, input.ContactEmail, input.Notes,
	).Scan(&c.ID, &c.Name, &c.ContactEmail, &c.Notes, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return c, fmt.Errorf("creating customer: %w", err)
	}
	return c, nil
}

func (r *CustomerRepository) GetByID(ctx context.Context, id pgtype.UUID) (model.Customer, error) {
	var c model.Customer
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, contact_email, notes, created_at, updated_at
		 FROM customers WHERE id = $1`, id,
	).Scan(&c.ID, &c.Name, &c.ContactEmail, &c.Notes, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return c, fmt.Errorf("getting customer: %w", err)
	}
	return c, nil
}

func (r *CustomerRepository) List(ctx context.Context, params model.ListParams) ([]model.Customer, error) {
	params.Normalize()
	query := `SELECT id, name, contact_email, notes, created_at, updated_at FROM customers`
	args := []any{}
	if params.Search != "" {
		query += ` WHERE name ILIKE $1`
		args = append(args, "%"+params.Search+"%")
	}
	query += fmt.Sprintf(` ORDER BY name LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2)
	args = append(args, params.Limit, params.Offset)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing customers: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowToStructByPos[model.Customer])
}

func (r *CustomerRepository) Update(ctx context.Context, id pgtype.UUID, input model.UpdateCustomerInput) (model.Customer, error) {
	var c model.Customer
	err := r.pool.QueryRow(ctx,
		`UPDATE customers SET
			name = COALESCE($2, name),
			contact_email = COALESCE($3, contact_email),
			notes = COALESCE($4, notes)
		 WHERE id = $1
		 RETURNING id, name, contact_email, notes, created_at, updated_at`,
		id, input.Name, input.ContactEmail, input.Notes,
	).Scan(&c.ID, &c.Name, &c.ContactEmail, &c.Notes, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return c, fmt.Errorf("updating customer: %w", err)
	}
	return c, nil
}

func (r *CustomerRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM customers WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting customer: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("customer not found")
	}
	return nil
}
```

- [ ] **Step 4: Run tests with test database**

```bash
export TEST_DATABASE_URL="postgres://xanpythia:xanpythia@localhost:5432/xanpythia?sslmode=disable"
go test ./internal/repository/... -v -run TestCustomerCRUD
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/repository/customer.go internal/repository/customer_test.go
git commit -m "feat: add customer repository with CRUD operations"
```

---

### Task 8: JSON Helper & Customer Handler

**Files:**
- Create: `internal/handler/json.go`
- Create: `internal/handler/customer.go`
- Create: `internal/handler/customer_test.go`

- [ ] **Step 1: Create JSON helpers**

```go
// internal/handler/json.go
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/xan-com/xan-pythia/internal/model"
)

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func isFKViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func paginationParams(r *http.Request) model.ListParams {
	p := model.ListParams{}
	if v := r.URL.Query().Get("limit"); v != "" {
		p.Limit, _ = strconv.Atoi(v)
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		p.Offset, _ = strconv.Atoi(v)
	}
	p.Search = r.URL.Query().Get("q")
	p.Filter = r.URL.Query().Get("type")
	p.Normalize()
	return p
}
```

- [ ] **Step 2: Create UUID parser helper**

```go
// internal/handler/params.go
package handler

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
)

func parseUUID(s string) (pgtype.UUID, error) {
	var id pgtype.UUID
	if err := id.Scan(s); err != nil {
		return id, fmt.Errorf("invalid UUID: %w", err)
	}
	return id, nil
}
```

- [ ] **Step 3: Write failing handler test**

```go
// internal/handler/customer_test.go
package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xan-com/xan-pythia/internal/handler"
	"github.com/xan-com/xan-pythia/internal/model"
)

func TestCustomerHandler_Create(t *testing.T) {
	h := handler.NewCustomerHandler(nil) // will need mock
	_ = h
	// Placeholder — real test after handler exists
	t.Skip("handler not implemented yet")
}

func TestCustomerHandler_CreateValidation(t *testing.T) {
	mux := http.NewServeMux()
	h := handler.NewCustomerHandler(nil)
	h.RegisterRoutes(mux)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/customers", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] == "" {
		t.Error("expected error message in response")
	}
}
```

- [ ] **Step 4: Implement customer handler**

```go
// internal/handler/customer.go
package handler

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/xan-com/xan-pythia/internal/model"
	"github.com/xan-com/xan-pythia/internal/repository"
)

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

func (h *CustomerHandler) list(w http.ResponseWriter, r *http.Request) {
	params := paginationParams(r)
	customers, err := h.repo.List(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list customers")
		return
	}
	writeJSON(w, http.StatusOK, customers)
}

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

func (h *CustomerHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer ID")
		return
	}
	customer, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "customer not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get customer")
		return
	}
	writeJSON(w, http.StatusOK, customer)
}

func (h *CustomerHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer ID")
		return
	}
	var input model.UpdateCustomerInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	customer, err := h.repo.Update(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "customer not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update customer")
		return
	}
	writeJSON(w, http.StatusOK, customer)
}

func (h *CustomerHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer ID")
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		if isFKViolation(err) {
			writeError(w, http.StatusConflict, "customer has dependent records")
			return
		}
		writeError(w, http.StatusNotFound, "customer not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 5: Run tests**

```bash
go test ./internal/handler/... -v
```

Expected: validation test passes, skip on nil repo test.

- [ ] **Step 6: Commit**

```bash
git add internal/handler/
git commit -m "feat: add customer handler with REST endpoints"
```

---

### Task 9: User Repository & Handler

**Files:**
- Create: `internal/repository/user.go`
- Create: `internal/repository/user_test.go`
- Create: `internal/handler/user.go`

- [ ] **Step 1: Write failing user repository test**

```go
// internal/repository/user_test.go
package repository_test

import (
	"context"
	"testing"

	"github.com/xan-com/xan-pythia/internal/model"
	"github.com/xan-com/xan-pythia/internal/repository"
)

func TestUserCRUD(t *testing.T) {
	pool := setupTestDB(t)
	repo := repository.NewUserRepository(pool)
	ctx := context.Background()

	// Create
	input := model.CreateUserInput{
		Type:      "internal_staff",
		FirstName: "Jane",
		LastName:  "Doe",
	}
	user, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if user.FirstName != "Jane" {
		t.Errorf("FirstName = %q, want %q", user.FirstName, "Jane")
	}

	// Get
	got, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.LastName != "Doe" {
		t.Errorf("LastName = %q, want %q", got.LastName, "Doe")
	}

	// Update
	newFirst := "Janet"
	updated, err := repo.Update(ctx, user.ID, model.UpdateUserInput{FirstName: &newFirst})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.FirstName != "Janet" {
		t.Errorf("Updated FirstName = %q, want %q", updated.FirstName, "Janet")
	}

	// List
	users, err := repo.List(ctx, model.ListParams{Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(users) == 0 {
		t.Error("List returned no users")
	}

	// Delete
	if err := repo.Delete(ctx, user.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/repository/... -v -run TestUserCRUD
```

Expected: compilation error.

- [ ] **Step 3: Implement user repository**

```go
// internal/repository/user.go
package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xan-com/xan-pythia/internal/model"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, input model.CreateUserInput) (model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (type, first_name, last_name)
		 VALUES ($1, $2, $3)
		 RETURNING id, type, first_name, last_name, created_at, updated_at`,
		input.Type, input.FirstName, input.LastName,
	).Scan(&u.ID, &u.Type, &u.FirstName, &u.LastName, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return u, fmt.Errorf("creating user: %w", err)
	}
	return u, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id pgtype.UUID) (model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx,
		`SELECT id, type, first_name, last_name, created_at, updated_at
		 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Type, &u.FirstName, &u.LastName, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return u, fmt.Errorf("getting user: %w", err)
	}
	return u, nil
}

func (r *UserRepository) List(ctx context.Context, params model.ListParams) ([]model.User, error) {
	params.Normalize()
	query := `SELECT id, type, first_name, last_name, created_at, updated_at FROM users`
	args := []any{}
	argN := 1
	clauses := []string{}
	if params.Filter != "" {
		clauses = append(clauses, fmt.Sprintf(`type = $%d`, argN))
		args = append(args, params.Filter)
		argN++
	}
	if params.Search != "" {
		clauses = append(clauses, fmt.Sprintf(`(first_name ILIKE $%d OR last_name ILIKE $%d)`, argN, argN))
		args = append(args, "%"+params.Search+"%")
		argN++
	}
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += fmt.Sprintf(` ORDER BY last_name, first_name LIMIT $%d OFFSET $%d`, argN, argN+1)
	args = append(args, params.Limit, params.Offset)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToStructByPos[model.User])
}

func (r *UserRepository) Update(ctx context.Context, id pgtype.UUID, input model.UpdateUserInput) (model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx,
		`UPDATE users SET
			type = COALESCE($2, type),
			first_name = COALESCE($3, first_name),
			last_name = COALESCE($4, last_name)
		 WHERE id = $1
		 RETURNING id, type, first_name, last_name, created_at, updated_at`,
		id, input.Type, input.FirstName, input.LastName,
	).Scan(&u.ID, &u.Type, &u.FirstName, &u.LastName, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return u, fmt.Errorf("updating user: %w", err)
	}
	return u, nil
}

func (r *UserRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}
```

- [ ] **Step 4: Implement user handler**

```go
// internal/handler/user.go
package handler

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/xan-com/xan-pythia/internal/model"
	"github.com/xan-com/xan-pythia/internal/repository"
)

type UserHandler struct {
	repo *repository.UserRepository
}

func NewUserHandler(repo *repository.UserRepository) *UserHandler {
	return &UserHandler{repo: repo}
}

func (h *UserHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/users", h.list)
	mux.HandleFunc("POST /api/v1/users", h.create)
	mux.HandleFunc("GET /api/v1/users/{id}", h.get)
	mux.HandleFunc("PUT /api/v1/users/{id}", h.update)
	mux.HandleFunc("DELETE /api/v1/users/{id}", h.delete)
}

func (h *UserHandler) list(w http.ResponseWriter, r *http.Request) {
	params := paginationParams(r)
	users, err := h.repo.List(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list users")
		return
	}
	writeJSON(w, http.StatusOK, users)
}

func (h *UserHandler) create(w http.ResponseWriter, r *http.Request) {
	var input model.CreateUserInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if input.FirstName == "" || input.LastName == "" {
		writeError(w, http.StatusBadRequest, "first_name and last_name are required")
		return
	}
	if input.Type != "customer_staff" && input.Type != "internal_staff" {
		writeError(w, http.StatusBadRequest, "type must be 'customer_staff' or 'internal_staff'")
		return
	}
	user, err := h.repo.Create(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

func (h *UserHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user ID")
		return
	}
	user, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get user")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *UserHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user ID")
		return
	}
	var input model.UpdateUserInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	user, err := h.repo.Update(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update user")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *UserHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user ID")
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		if isFKViolation(err) {
			writeError(w, http.StatusConflict, "user has dependent records")
			return
		}
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 5: Run tests**

```bash
go vet ./...
go test ./internal/repository/... -v -run TestUserCRUD
```

Expected: PASS (with TEST_DATABASE_URL set)

- [ ] **Step 6: Commit**

```bash
git add internal/repository/user.go internal/repository/user_test.go internal/handler/user.go
git commit -m "feat: add user repository and handler"
```

---

### Task 10: Service Repository & Handler

**Files:**
- Create: `internal/repository/service.go`
- Create: `internal/repository/service_test.go`
- Create: `internal/handler/service.go`

- [ ] **Step 1: Write failing service repository test**

```go
// internal/repository/service_test.go
package repository_test

import (
	"context"
	"testing"

	"github.com/xan-com/xan-pythia/internal/model"
	"github.com/xan-com/xan-pythia/internal/repository"
)

func TestServiceCRUD(t *testing.T) {
	pool := setupTestDB(t)
	repo := repository.NewServiceRepository(pool)
	ctx := context.Background()

	desc := "Managed email"
	input := model.CreateServiceInput{Name: "Email Hosting", Description: &desc}
	svc, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if svc.Name != "Email Hosting" {
		t.Errorf("Name = %q, want %q", svc.Name, "Email Hosting")
	}

	got, err := repo.GetByID(ctx, svc.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "Email Hosting" {
		t.Errorf("GetByID Name = %q, want %q", got.Name, "Email Hosting")
	}

	newName := "Email Premium"
	updated, err := repo.Update(ctx, svc.ID, model.UpdateServiceInput{Name: &newName})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "Email Premium" {
		t.Errorf("Updated Name = %q, want %q", updated.Name, "Email Premium")
	}

	services, err := repo.List(ctx, model.ListParams{Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(services) == 0 {
		t.Error("List returned no services")
	}

	if err := repo.Delete(ctx, svc.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}
```

- [ ] **Step 2: Implement service repository** (follows same pattern as customer repo)

```go
// internal/repository/service.go
package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xan-com/xan-pythia/internal/model"
)

type ServiceRepository struct {
	pool *pgxpool.Pool
}

func NewServiceRepository(pool *pgxpool.Pool) *ServiceRepository {
	return &ServiceRepository{pool: pool}
}

func (r *ServiceRepository) Create(ctx context.Context, input model.CreateServiceInput) (model.Service, error) {
	var s model.Service
	err := r.pool.QueryRow(ctx,
		`INSERT INTO services (name, description) VALUES ($1, $2)
		 RETURNING id, name, description, created_at, updated_at`,
		input.Name, input.Description,
	).Scan(&s.ID, &s.Name, &s.Description, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return s, fmt.Errorf("creating service: %w", err)
	}
	return s, nil
}

func (r *ServiceRepository) GetByID(ctx context.Context, id pgtype.UUID) (model.Service, error) {
	var s model.Service
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, description, created_at, updated_at FROM services WHERE id = $1`, id,
	).Scan(&s.ID, &s.Name, &s.Description, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return s, fmt.Errorf("getting service: %w", err)
	}
	return s, nil
}

func (r *ServiceRepository) List(ctx context.Context, params model.ListParams) ([]model.Service, error) {
	params.Normalize()
	query := `SELECT id, name, description, created_at, updated_at FROM services`
	args := []any{}
	if params.Search != "" {
		query += ` WHERE name ILIKE $1`
		args = append(args, "%"+params.Search+"%")
	}
	query += fmt.Sprintf(` ORDER BY name LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2)
	args = append(args, params.Limit, params.Offset)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing services: %w", err)
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToStructByPos[model.Service])
}

func (r *ServiceRepository) Update(ctx context.Context, id pgtype.UUID, input model.UpdateServiceInput) (model.Service, error) {
	var s model.Service
	err := r.pool.QueryRow(ctx,
		`UPDATE services SET name = COALESCE($2, name), description = COALESCE($3, description)
		 WHERE id = $1
		 RETURNING id, name, description, created_at, updated_at`,
		id, input.Name, input.Description,
	).Scan(&s.ID, &s.Name, &s.Description, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return s, fmt.Errorf("updating service: %w", err)
	}
	return s, nil
}

func (r *ServiceRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM services WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting service: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("service not found")
	}
	return nil
}
```

- [ ] **Step 3: Implement service handler**

```go
// internal/handler/service.go
package handler

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/xan-com/xan-pythia/internal/model"
	"github.com/xan-com/xan-pythia/internal/repository"
)

type ServiceHandler struct {
	repo *repository.ServiceRepository
}

func NewServiceHandler(repo *repository.ServiceRepository) *ServiceHandler {
	return &ServiceHandler{repo: repo}
}

func (h *ServiceHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/services", h.list)
	mux.HandleFunc("POST /api/v1/services", h.create)
	mux.HandleFunc("GET /api/v1/services/{id}", h.get)
	mux.HandleFunc("PUT /api/v1/services/{id}", h.update)
	mux.HandleFunc("DELETE /api/v1/services/{id}", h.delete)
}

func (h *ServiceHandler) list(w http.ResponseWriter, r *http.Request) {
	params := paginationParams(r)
	services, err := h.repo.List(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list services")
		return
	}
	writeJSON(w, http.StatusOK, services)
}

func (h *ServiceHandler) create(w http.ResponseWriter, r *http.Request) {
	var input model.CreateServiceInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if input.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	svc, err := h.repo.Create(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create service")
		return
	}
	writeJSON(w, http.StatusCreated, svc)
}

func (h *ServiceHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid service ID")
		return
	}
	svc, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "service not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get service")
		return
	}
	writeJSON(w, http.StatusOK, svc)
}

func (h *ServiceHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid service ID")
		return
	}
	var input model.UpdateServiceInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	svc, err := h.repo.Update(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "service not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update service")
		return
	}
	writeJSON(w, http.StatusOK, svc)
}

func (h *ServiceHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid service ID")
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		if isFKViolation(err) {
			writeError(w, http.StatusConflict, "service has dependent records")
			return
		}
		writeError(w, http.StatusNotFound, "service not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 4: Run tests**

```bash
go vet ./...
go test ./internal/repository/... -v -run TestServiceCRUD
```

- [ ] **Step 5: Commit**

```bash
git add internal/repository/service.go internal/repository/service_test.go internal/handler/service.go
git commit -m "feat: add service repository and handler"
```

---

## Phase 3: Dependent Entities

### Task 11: Dependent Models

**Files:**
- Create: `internal/model/user_assignment.go`
- Create: `internal/model/asset.go`
- Create: `internal/model/license.go`
- Create: `internal/model/customer_service.go`

- [ ] **Step 1: Create user_assignment model**

```go
// internal/model/user_assignment.go
package model

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type UserAssignment struct {
	ID         pgtype.UUID `json:"id"`
	UserID     pgtype.UUID `json:"user_id"`
	CustomerID pgtype.UUID `json:"customer_id"`
	Role       string      `json:"role"`
	Email      pgtype.Text `json:"email"`
	Phone      pgtype.Text `json:"phone"`
	Notes      pgtype.Text `json:"notes"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
}

type CreateUserAssignmentInput struct {
	UserID     pgtype.UUID `json:"user_id"`
	CustomerID pgtype.UUID `json:"customer_id"`
	Role       string      `json:"role"`
	Email      *string     `json:"email,omitempty"`
	Phone      *string     `json:"phone,omitempty"`
	Notes      *string     `json:"notes,omitempty"`
}

type UpdateUserAssignmentInput struct {
	Role  *string `json:"role,omitempty"`
	Email *string `json:"email,omitempty"`
	Phone *string `json:"phone,omitempty"`
	Notes *string `json:"notes,omitempty"`
}
```

- [ ] **Step 2: Create asset model**

```go
// internal/model/asset.go
package model

import (
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type Asset struct {
	ID          pgtype.UUID     `json:"id"`
	CustomerID  pgtype.UUID     `json:"customer_id"`
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Description pgtype.Text     `json:"description"`
	Metadata    json.RawMessage `json:"metadata"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type CreateAssetInput struct {
	CustomerID  pgtype.UUID     `json:"customer_id"`
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Description *string         `json:"description,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
}

type UpdateAssetInput struct {
	Type        *string         `json:"type,omitempty"`
	Name        *string         `json:"name,omitempty"`
	Description *string         `json:"description,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
}
```

- [ ] **Step 3: Create license model**

```go
// internal/model/license.go
package model

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type License struct {
	ID               pgtype.UUID `json:"id"`
	CustomerID       pgtype.UUID `json:"customer_id"`
	UserAssignmentID pgtype.UUID `json:"user_assignment_id"`
	ProductName      string      `json:"product_name"`
	LicenseKey       pgtype.Text `json:"license_key"`
	Quantity         int         `json:"quantity"`
	ValidFrom        pgtype.Date `json:"valid_from"`
	ValidUntil       pgtype.Date `json:"valid_until"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
}

type CreateLicenseInput struct {
	CustomerID       pgtype.UUID `json:"customer_id"`
	UserAssignmentID *pgtype.UUID `json:"user_assignment_id,omitempty"`
	ProductName      string      `json:"product_name"`
	LicenseKey       *string     `json:"license_key,omitempty"`
	Quantity         int         `json:"quantity"`
	ValidFrom        *pgtype.Date `json:"valid_from,omitempty"`
	ValidUntil       *pgtype.Date `json:"valid_until,omitempty"`
}

type UpdateLicenseInput struct {
	UserAssignmentID *pgtype.UUID `json:"user_assignment_id,omitempty"`
	ProductName      *string     `json:"product_name,omitempty"`
	LicenseKey       *string     `json:"license_key,omitempty"`
	Quantity         *int        `json:"quantity,omitempty"`
	ValidFrom        *pgtype.Date `json:"valid_from,omitempty"`
	ValidUntil       *pgtype.Date `json:"valid_until,omitempty"`
}
```

- [ ] **Step 4: Create customer_service model**

```go
// internal/model/customer_service.go
package model

import (
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type CustomerService struct {
	ID             pgtype.UUID     `json:"id"`
	CustomerID     pgtype.UUID     `json:"customer_id"`
	ServiceID      pgtype.UUID     `json:"service_id"`
	Customizations json.RawMessage `json:"customizations"`
	Notes          pgtype.Text     `json:"notes"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type CreateCustomerServiceInput struct {
	CustomerID     pgtype.UUID     `json:"customer_id"`
	ServiceID      pgtype.UUID     `json:"service_id"`
	Customizations json.RawMessage `json:"customizations,omitempty"`
	Notes          *string         `json:"notes,omitempty"`
}

type UpdateCustomerServiceInput struct {
	Customizations json.RawMessage `json:"customizations,omitempty"`
	Notes          *string         `json:"notes,omitempty"`
}
```

- [ ] **Step 5: Verify compilation**

```bash
go vet ./internal/model/...
```

- [ ] **Step 6: Commit**

```bash
git add internal/model/
git commit -m "feat: add models for user_assignments, assets, licenses, customer_services"
```

---

### Task 12: User Assignment Repository & Handler

**Files:**
- Create: `internal/repository/user_assignment.go`
- Create: `internal/repository/user_assignment_test.go`
- Create: `internal/handler/user_assignment.go`

- [ ] **Step 1: Write failing test**

```go
// internal/repository/user_assignment_test.go
package repository_test

import (
	"context"
	"testing"

	"github.com/xan-com/xan-pythia/internal/model"
	"github.com/xan-com/xan-pythia/internal/repository"
)

func TestUserAssignmentCRUD(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	// Create prerequisite customer and user
	custRepo := repository.NewCustomerRepository(pool)
	cust, err := custRepo.Create(ctx, model.CreateCustomerInput{Name: "Test Customer"})
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}
	userRepo := repository.NewUserRepository(pool)
	user, err := userRepo.Create(ctx, model.CreateUserInput{
		Type: "customer_staff", FirstName: "Test", LastName: "User",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	repo := repository.NewUserAssignmentRepository(pool)

	// Create
	input := model.CreateUserAssignmentInput{
		UserID:     user.ID,
		CustomerID: cust.ID,
		Role:       "admin",
	}
	assignment, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if assignment.Role != "admin" {
		t.Errorf("Role = %q, want %q", assignment.Role, "admin")
	}

	// Get
	got, err := repo.GetByID(ctx, assignment.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Role != "admin" {
		t.Errorf("GetByID Role = %q, want %q", got.Role, "admin")
	}

	// List by customer
	assignments, err := repo.ListByCustomer(ctx, cust.ID, model.ListParams{Limit: 10})
	if err != nil {
		t.Fatalf("ListByCustomer: %v", err)
	}
	if len(assignments) == 0 {
		t.Error("ListByCustomer returned no assignments")
	}

	// Update
	newRole := "user"
	updated, err := repo.Update(ctx, assignment.ID, model.UpdateUserAssignmentInput{Role: &newRole})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Role != "user" {
		t.Errorf("Updated Role = %q, want %q", updated.Role, "user")
	}

	// Delete
	if err := repo.Delete(ctx, assignment.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Cleanup
	_ = userRepo.Delete(ctx, user.ID)
	_ = custRepo.Delete(ctx, cust.ID)
}
```

- [ ] **Step 2: Implement user assignment repository**

```go
// internal/repository/user_assignment.go
package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xan-com/xan-pythia/internal/model"
)

type UserAssignmentRepository struct {
	pool *pgxpool.Pool
}

func NewUserAssignmentRepository(pool *pgxpool.Pool) *UserAssignmentRepository {
	return &UserAssignmentRepository{pool: pool}
}

func (r *UserAssignmentRepository) Create(ctx context.Context, input model.CreateUserAssignmentInput) (model.UserAssignment, error) {
	var a model.UserAssignment
	err := r.pool.QueryRow(ctx,
		`INSERT INTO user_assignments (user_id, customer_id, role, email, phone, notes)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, user_id, customer_id, role, email, phone, notes, created_at, updated_at`,
		input.UserID, input.CustomerID, input.Role, input.Email, input.Phone, input.Notes,
	).Scan(&a.ID, &a.UserID, &a.CustomerID, &a.Role, &a.Email, &a.Phone, &a.Notes, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return a, fmt.Errorf("creating user assignment: %w", err)
	}
	return a, nil
}

func (r *UserAssignmentRepository) GetByID(ctx context.Context, id pgtype.UUID) (model.UserAssignment, error) {
	var a model.UserAssignment
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, customer_id, role, email, phone, notes, created_at, updated_at
		 FROM user_assignments WHERE id = $1`, id,
	).Scan(&a.ID, &a.UserID, &a.CustomerID, &a.Role, &a.Email, &a.Phone, &a.Notes, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return a, fmt.Errorf("getting user assignment: %w", err)
	}
	return a, nil
}

func (r *UserAssignmentRepository) ListByCustomer(ctx context.Context, customerID pgtype.UUID, params model.ListParams) ([]model.UserAssignment, error) {
	params.Normalize()
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, customer_id, role, email, phone, notes, created_at, updated_at
		 FROM user_assignments WHERE customer_id = $1 ORDER BY role LIMIT $2 OFFSET $3`,
		customerID, params.Limit, params.Offset,
	)
	if err != nil {
		return nil, fmt.Errorf("listing user assignments: %w", err)
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToStructByPos[model.UserAssignment])
}

func (r *UserAssignmentRepository) Update(ctx context.Context, id pgtype.UUID, input model.UpdateUserAssignmentInput) (model.UserAssignment, error) {
	var a model.UserAssignment
	err := r.pool.QueryRow(ctx,
		`UPDATE user_assignments SET
			role = COALESCE($2, role),
			email = COALESCE($3, email),
			phone = COALESCE($4, phone),
			notes = COALESCE($5, notes)
		 WHERE id = $1
		 RETURNING id, user_id, customer_id, role, email, phone, notes, created_at, updated_at`,
		id, input.Role, input.Email, input.Phone, input.Notes,
	).Scan(&a.ID, &a.UserID, &a.CustomerID, &a.Role, &a.Email, &a.Phone, &a.Notes, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return a, fmt.Errorf("updating user assignment: %w", err)
	}
	return a, nil
}

func (r *UserAssignmentRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM user_assignments WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting user assignment: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("user assignment not found")
	}
	return nil
}
```

- [ ] **Step 3: Implement user assignment handler**

Note: user assignments are nested under customers: `GET/POST /api/v1/customers/{id}/user-assignments`

```go
// internal/handler/user_assignment.go
package handler

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/xan-com/xan-pythia/internal/model"
	"github.com/xan-com/xan-pythia/internal/repository"
)

type UserAssignmentHandler struct {
	repo *repository.UserAssignmentRepository
}

func NewUserAssignmentHandler(repo *repository.UserAssignmentRepository) *UserAssignmentHandler {
	return &UserAssignmentHandler{repo: repo}
}

func (h *UserAssignmentHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/customers/{customerId}/user-assignments", h.list)
	mux.HandleFunc("POST /api/v1/customers/{customerId}/user-assignments", h.create)
	mux.HandleFunc("GET /api/v1/user-assignments/{id}", h.get)
	mux.HandleFunc("PUT /api/v1/user-assignments/{id}", h.update)
	mux.HandleFunc("DELETE /api/v1/user-assignments/{id}", h.delete)
}

func (h *UserAssignmentHandler) list(w http.ResponseWriter, r *http.Request) {
	customerID, err := parseUUID(r.PathValue("customerId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer ID")
		return
	}
	params := paginationParams(r)
	assignments, err := h.repo.ListByCustomer(r.Context(), customerID, params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list user assignments")
		return
	}
	writeJSON(w, http.StatusOK, assignments)
}

func (h *UserAssignmentHandler) create(w http.ResponseWriter, r *http.Request) {
	customerID, err := parseUUID(r.PathValue("customerId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer ID")
		return
	}
	var input model.CreateUserAssignmentInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input.CustomerID = customerID
	if !input.UserID.Valid {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}
	if input.Role == "" {
		writeError(w, http.StatusBadRequest, "role is required")
		return
	}
	assignment, err := h.repo.Create(r.Context(), input)
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "user is already assigned to this customer")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create user assignment")
		return
	}
	writeJSON(w, http.StatusCreated, assignment)
}

func (h *UserAssignmentHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user assignment ID")
		return
	}
	assignment, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "user assignment not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get user assignment")
		return
	}
	writeJSON(w, http.StatusOK, assignment)
}

func (h *UserAssignmentHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user assignment ID")
		return
	}
	var input model.UpdateUserAssignmentInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	assignment, err := h.repo.Update(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "user assignment not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update user assignment")
		return
	}
	writeJSON(w, http.StatusOK, assignment)
}

func (h *UserAssignmentHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user assignment ID")
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		if isFKViolation(err) {
			writeError(w, http.StatusConflict, "user assignment has dependent records")
			return
		}
		writeError(w, http.StatusNotFound, "user assignment not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 4: Run tests**

```bash
go vet ./...
go test ./internal/repository/... -v -run TestUserAssignmentCRUD
```

- [ ] **Step 5: Commit**

```bash
git add internal/repository/user_assignment.go internal/repository/user_assignment_test.go internal/handler/user_assignment.go
git commit -m "feat: add user assignment repository and handler"
```

---

### Task 13: Asset Repository & Handler

**Files:**
- Create: `internal/repository/asset.go`
- Create: `internal/repository/asset_test.go`
- Create: `internal/handler/asset.go`

- [ ] **Step 1: Write failing test**

```go
// internal/repository/asset_test.go
package repository_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/xan-com/xan-pythia/internal/model"
	"github.com/xan-com/xan-pythia/internal/repository"
)

func TestAssetCRUD(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	custRepo := repository.NewCustomerRepository(pool)
	cust, err := custRepo.Create(ctx, model.CreateCustomerInput{Name: "Asset Test Customer"})
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}

	repo := repository.NewAssetRepository(pool)

	// Create
	input := model.CreateAssetInput{
		CustomerID: cust.ID,
		Type:       "hardware",
		Name:       "Laptop Dell XPS 15",
		Metadata:   json.RawMessage(`{"serial":"ABC123"}`),
	}
	asset, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if asset.Name != "Laptop Dell XPS 15" {
		t.Errorf("Name = %q, want %q", asset.Name, "Laptop Dell XPS 15")
	}

	// Get
	got, err := repo.GetByID(ctx, asset.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Type != "hardware" {
		t.Errorf("Type = %q, want %q", got.Type, "hardware")
	}

	// List by customer
	assets, err := repo.ListByCustomer(ctx, cust.ID, model.ListParams{Limit: 10})
	if err != nil {
		t.Fatalf("ListByCustomer: %v", err)
	}
	if len(assets) == 0 {
		t.Error("ListByCustomer returned no assets")
	}

	// Update
	newName := "Laptop Dell XPS 17"
	updated, err := repo.Update(ctx, asset.ID, model.UpdateAssetInput{Name: &newName})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "Laptop Dell XPS 17" {
		t.Errorf("Updated Name = %q, want %q", updated.Name, "Laptop Dell XPS 17")
	}

	// Delete
	if err := repo.Delete(ctx, asset.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Cleanup
	_ = custRepo.Delete(ctx, cust.ID)
}
```

- [ ] **Step 2: Implement asset repository**

```go
// internal/repository/asset.go
package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xan-com/xan-pythia/internal/model"
)

type AssetRepository struct {
	pool *pgxpool.Pool
}

func NewAssetRepository(pool *pgxpool.Pool) *AssetRepository {
	return &AssetRepository{pool: pool}
}

func (r *AssetRepository) Create(ctx context.Context, input model.CreateAssetInput) (model.Asset, error) {
	var a model.Asset
	metadata := input.Metadata
	if metadata == nil {
		metadata = []byte("{}")
	}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO assets (customer_id, type, name, description, metadata)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, customer_id, type, name, description, metadata, created_at, updated_at`,
		input.CustomerID, input.Type, input.Name, input.Description, metadata,
	).Scan(&a.ID, &a.CustomerID, &a.Type, &a.Name, &a.Description, &a.Metadata, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return a, fmt.Errorf("creating asset: %w", err)
	}
	return a, nil
}

func (r *AssetRepository) GetByID(ctx context.Context, id pgtype.UUID) (model.Asset, error) {
	var a model.Asset
	err := r.pool.QueryRow(ctx,
		`SELECT id, customer_id, type, name, description, metadata, created_at, updated_at
		 FROM assets WHERE id = $1`, id,
	).Scan(&a.ID, &a.CustomerID, &a.Type, &a.Name, &a.Description, &a.Metadata, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return a, fmt.Errorf("getting asset: %w", err)
	}
	return a, nil
}

func (r *AssetRepository) ListByCustomer(ctx context.Context, customerID pgtype.UUID, params model.ListParams) ([]model.Asset, error) {
	params.Normalize()
	query := `SELECT id, customer_id, type, name, description, metadata, created_at, updated_at
		 FROM assets WHERE customer_id = $1`
	args := []any{customerID}
	argN := 2
	if params.Filter != "" {
		query += fmt.Sprintf(` AND type = $%d`, argN)
		args = append(args, params.Filter)
		argN++
	}
	if params.Search != "" {
		query += fmt.Sprintf(` AND name ILIKE $%d`, argN)
		args = append(args, "%"+params.Search+"%")
		argN++
	}
	query += fmt.Sprintf(` ORDER BY name LIMIT $%d OFFSET $%d`, argN, argN+1)
	args = append(args, params.Limit, params.Offset)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing assets: %w", err)
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToStructByPos[model.Asset])
}

func (r *AssetRepository) Update(ctx context.Context, id pgtype.UUID, input model.UpdateAssetInput) (model.Asset, error) {
	var a model.Asset
	err := r.pool.QueryRow(ctx,
		`UPDATE assets SET
			type = COALESCE($2, type),
			name = COALESCE($3, name),
			description = COALESCE($4, description),
			metadata = COALESCE($5, metadata)
		 WHERE id = $1
		 RETURNING id, customer_id, type, name, description, metadata, created_at, updated_at`,
		id, input.Type, input.Name, input.Description, input.Metadata,
	).Scan(&a.ID, &a.CustomerID, &a.Type, &a.Name, &a.Description, &a.Metadata, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return a, fmt.Errorf("updating asset: %w", err)
	}
	return a, nil
}

func (r *AssetRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM assets WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting asset: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("asset not found")
	}
	return nil
}
```

- [ ] **Step 3: Implement asset handler**

```go
// internal/handler/asset.go
package handler

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/xan-com/xan-pythia/internal/model"
	"github.com/xan-com/xan-pythia/internal/repository"
)

type AssetHandler struct {
	repo *repository.AssetRepository
}

func NewAssetHandler(repo *repository.AssetRepository) *AssetHandler {
	return &AssetHandler{repo: repo}
}

func (h *AssetHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/customers/{customerId}/assets", h.list)
	mux.HandleFunc("POST /api/v1/customers/{customerId}/assets", h.create)
	mux.HandleFunc("GET /api/v1/assets/{id}", h.get)
	mux.HandleFunc("PUT /api/v1/assets/{id}", h.update)
	mux.HandleFunc("DELETE /api/v1/assets/{id}", h.delete)
}

func (h *AssetHandler) list(w http.ResponseWriter, r *http.Request) {
	customerID, err := parseUUID(r.PathValue("customerId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer ID")
		return
	}
	params := paginationParams(r)
	assets, err := h.repo.ListByCustomer(r.Context(), customerID, params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list assets")
		return
	}
	writeJSON(w, http.StatusOK, assets)
}

func (h *AssetHandler) create(w http.ResponseWriter, r *http.Request) {
	customerID, err := parseUUID(r.PathValue("customerId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer ID")
		return
	}
	var input model.CreateAssetInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input.CustomerID = customerID
	if input.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if input.Type != "hardware" && input.Type != "software" {
		writeError(w, http.StatusBadRequest, "type must be 'hardware' or 'software'")
		return
	}
	asset, err := h.repo.Create(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create asset")
		return
	}
	writeJSON(w, http.StatusCreated, asset)
}

func (h *AssetHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset ID")
		return
	}
	asset, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "asset not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get asset")
		return
	}
	writeJSON(w, http.StatusOK, asset)
}

func (h *AssetHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset ID")
		return
	}
	var input model.UpdateAssetInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	asset, err := h.repo.Update(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "asset not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update asset")
		return
	}
	writeJSON(w, http.StatusOK, asset)
}

func (h *AssetHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid asset ID")
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		if isFKViolation(err) {
			writeError(w, http.StatusConflict, "asset has dependent records")
			return
		}
		writeError(w, http.StatusNotFound, "asset not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 4: Run tests**

```bash
go vet ./...
go test ./internal/repository/... -v -run TestAssetCRUD
```

- [ ] **Step 5: Commit**

```bash
git add internal/repository/asset.go internal/repository/asset_test.go internal/handler/asset.go
git commit -m "feat: add asset repository and handler"
```

---

### Task 14: License Repository & Handler

**Files:**
- Create: `internal/repository/license.go`
- Create: `internal/repository/license_test.go`
- Create: `internal/handler/license.go`

- [ ] **Step 1: Write failing test (includes license consistency trigger test)**

```go
// internal/repository/license_test.go
package repository_test

import (
	"context"
	"strings"
	"testing"

	"github.com/xan-com/xan-pythia/internal/model"
	"github.com/xan-com/xan-pythia/internal/repository"
)

func TestLicenseCRUD(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	custRepo := repository.NewCustomerRepository(pool)
	cust, err := custRepo.Create(ctx, model.CreateCustomerInput{Name: "License Test Customer"})
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}

	repo := repository.NewLicenseRepository(pool)

	// Create company-level license (no user_assignment_id)
	input := model.CreateLicenseInput{
		CustomerID:  cust.ID,
		ProductName: "Microsoft 365 E3",
		Quantity:    50,
	}
	license, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if license.ProductName != "Microsoft 365 E3" {
		t.Errorf("ProductName = %q, want %q", license.ProductName, "Microsoft 365 E3")
	}
	if license.Quantity != 50 {
		t.Errorf("Quantity = %d, want %d", license.Quantity, 50)
	}

	// Get
	got, err := repo.GetByID(ctx, license.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ProductName != "Microsoft 365 E3" {
		t.Errorf("GetByID ProductName = %q", got.ProductName)
	}

	// List by customer
	licenses, err := repo.ListByCustomer(ctx, cust.ID, model.ListParams{Limit: 10})
	if err != nil {
		t.Fatalf("ListByCustomer: %v", err)
	}
	if len(licenses) == 0 {
		t.Error("ListByCustomer returned no licenses")
	}

	// Update
	newProduct := "Microsoft 365 E5"
	updated, err := repo.Update(ctx, license.ID, model.UpdateLicenseInput{ProductName: &newProduct})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.ProductName != "Microsoft 365 E5" {
		t.Errorf("Updated ProductName = %q", updated.ProductName)
	}

	// Delete
	if err := repo.Delete(ctx, license.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Cleanup
	_ = custRepo.Delete(ctx, cust.ID)
}

func TestLicenseConsistencyTrigger(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	custRepo := repository.NewCustomerRepository(pool)
	userRepo := repository.NewUserRepository(pool)
	assignRepo := repository.NewUserAssignmentRepository(pool)
	licenseRepo := repository.NewLicenseRepository(pool)

	// Create two customers
	custA, _ := custRepo.Create(ctx, model.CreateCustomerInput{Name: "Customer A"})
	custB, _ := custRepo.Create(ctx, model.CreateCustomerInput{Name: "Customer B"})

	// Create a user and assign to Customer A
	user, _ := userRepo.Create(ctx, model.CreateUserInput{
		Type: "customer_staff", FirstName: "Test", LastName: "User",
	})
	assignment, _ := assignRepo.Create(ctx, model.CreateUserAssignmentInput{
		UserID: user.ID, CustomerID: custA.ID, Role: "user",
	})

	// Try to create a license for Customer B with assignment from Customer A
	_, err := licenseRepo.Create(ctx, model.CreateLicenseInput{
		CustomerID:       custB.ID,
		UserAssignmentID: &assignment.ID,
		ProductName:      "Bad License",
		Quantity:         1,
	})
	if err == nil {
		t.Fatal("expected error from consistency trigger, got nil")
	}
	if !strings.Contains(err.Error(), "does not match") {
		t.Errorf("expected 'does not match' error, got: %v", err)
	}

	// Cleanup
	_ = assignRepo.Delete(ctx, assignment.ID)
	_ = userRepo.Delete(ctx, user.ID)
	_ = custRepo.Delete(ctx, custA.ID)
	_ = custRepo.Delete(ctx, custB.ID)
}
```

- [ ] **Step 2: Implement license repository**

```go
// internal/repository/license.go
package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xan-com/xan-pythia/internal/model"
)

type LicenseRepository struct {
	pool *pgxpool.Pool
}

func NewLicenseRepository(pool *pgxpool.Pool) *LicenseRepository {
	return &LicenseRepository{pool: pool}
}

func (r *LicenseRepository) Create(ctx context.Context, input model.CreateLicenseInput) (model.License, error) {
	var l model.License
	var assignmentID *pgtype.UUID
	if input.UserAssignmentID != nil {
		assignmentID = input.UserAssignmentID
	}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO licenses (customer_id, user_assignment_id, product_name, license_key, quantity, valid_from, valid_until)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, customer_id, user_assignment_id, product_name, license_key, quantity, valid_from, valid_until, created_at, updated_at`,
		input.CustomerID, assignmentID, input.ProductName, input.LicenseKey, input.Quantity, input.ValidFrom, input.ValidUntil,
	).Scan(&l.ID, &l.CustomerID, &l.UserAssignmentID, &l.ProductName, &l.LicenseKey, &l.Quantity, &l.ValidFrom, &l.ValidUntil, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		return l, fmt.Errorf("creating license: %w", err)
	}
	return l, nil
}

func (r *LicenseRepository) GetByID(ctx context.Context, id pgtype.UUID) (model.License, error) {
	var l model.License
	err := r.pool.QueryRow(ctx,
		`SELECT id, customer_id, user_assignment_id, product_name, license_key, quantity, valid_from, valid_until, created_at, updated_at
		 FROM licenses WHERE id = $1`, id,
	).Scan(&l.ID, &l.CustomerID, &l.UserAssignmentID, &l.ProductName, &l.LicenseKey, &l.Quantity, &l.ValidFrom, &l.ValidUntil, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		return l, fmt.Errorf("getting license: %w", err)
	}
	return l, nil
}

func (r *LicenseRepository) ListByCustomer(ctx context.Context, customerID pgtype.UUID, params model.ListParams) ([]model.License, error) {
	params.Normalize()
	rows, err := r.pool.Query(ctx,
		`SELECT id, customer_id, user_assignment_id, product_name, license_key, quantity, valid_from, valid_until, created_at, updated_at
		 FROM licenses WHERE customer_id = $1 ORDER BY product_name LIMIT $2 OFFSET $3`,
		customerID, params.Limit, params.Offset,
	)
	if err != nil {
		return nil, fmt.Errorf("listing licenses: %w", err)
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToStructByPos[model.License])
}

func (r *LicenseRepository) Update(ctx context.Context, id pgtype.UUID, input model.UpdateLicenseInput) (model.License, error) {
	var l model.License
	err := r.pool.QueryRow(ctx,
		`UPDATE licenses SET
			user_assignment_id = COALESCE($2, user_assignment_id),
			product_name = COALESCE($3, product_name),
			license_key = COALESCE($4, license_key),
			quantity = COALESCE($5, quantity),
			valid_from = COALESCE($6, valid_from),
			valid_until = COALESCE($7, valid_until)
		 WHERE id = $1
		 RETURNING id, customer_id, user_assignment_id, product_name, license_key, quantity, valid_from, valid_until, created_at, updated_at`,
		id, input.UserAssignmentID, input.ProductName, input.LicenseKey, input.Quantity, input.ValidFrom, input.ValidUntil,
	).Scan(&l.ID, &l.CustomerID, &l.UserAssignmentID, &l.ProductName, &l.LicenseKey, &l.Quantity, &l.ValidFrom, &l.ValidUntil, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		return l, fmt.Errorf("updating license: %w", err)
	}
	return l, nil
}

func (r *LicenseRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM licenses WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting license: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("license not found")
	}
	return nil
}
```

- [ ] **Step 3: Implement license handler**

```go
// internal/handler/license.go
package handler

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/xan-com/xan-pythia/internal/model"
	"github.com/xan-com/xan-pythia/internal/repository"
)

type LicenseHandler struct {
	repo *repository.LicenseRepository
}

func NewLicenseHandler(repo *repository.LicenseRepository) *LicenseHandler {
	return &LicenseHandler{repo: repo}
}

func (h *LicenseHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/customers/{customerId}/licenses", h.list)
	mux.HandleFunc("POST /api/v1/customers/{customerId}/licenses", h.create)
	mux.HandleFunc("GET /api/v1/licenses/{id}", h.get)
	mux.HandleFunc("PUT /api/v1/licenses/{id}", h.update)
	mux.HandleFunc("DELETE /api/v1/licenses/{id}", h.delete)
}

func (h *LicenseHandler) list(w http.ResponseWriter, r *http.Request) {
	customerID, err := parseUUID(r.PathValue("customerId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer ID")
		return
	}
	params := paginationParams(r)
	licenses, err := h.repo.ListByCustomer(r.Context(), customerID, params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list licenses")
		return
	}
	writeJSON(w, http.StatusOK, licenses)
}

func (h *LicenseHandler) create(w http.ResponseWriter, r *http.Request) {
	customerID, err := parseUUID(r.PathValue("customerId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer ID")
		return
	}
	var input model.CreateLicenseInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input.CustomerID = customerID
	if input.ProductName == "" {
		writeError(w, http.StatusBadRequest, "product_name is required")
		return
	}
	if input.Quantity <= 0 {
		writeError(w, http.StatusBadRequest, "quantity must be positive")
		return
	}
	license, err := h.repo.Create(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to create license: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, license)
}

func (h *LicenseHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid license ID")
		return
	}
	license, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "license not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get license")
		return
	}
	writeJSON(w, http.StatusOK, license)
}

func (h *LicenseHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid license ID")
		return
	}
	var input model.UpdateLicenseInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	license, err := h.repo.Update(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "license not found")
			return
		}
		writeError(w, http.StatusBadRequest, "failed to update license: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, license)
}

func (h *LicenseHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid license ID")
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		if isFKViolation(err) {
			writeError(w, http.StatusConflict, "license has dependent records")
			return
		}
		writeError(w, http.StatusNotFound, "license not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 4: Run tests**

```bash
go vet ./...
go test ./internal/repository/... -v -run "TestLicense"
```

Expected: both TestLicenseCRUD and TestLicenseConsistencyTrigger pass.

- [ ] **Step 5: Commit**

```bash
git add internal/repository/license.go internal/repository/license_test.go internal/handler/license.go
git commit -m "feat: add license repository and handler with consistency trigger test"
```

---

### Task 15: Customer Service Repository & Handler

**Files:**
- Create: `internal/repository/customer_service.go`
- Create: `internal/repository/customer_service_test.go`
- Create: `internal/handler/customer_service.go`

- [ ] **Step 1: Write failing test**

```go
// internal/repository/customer_service_test.go
package repository_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/xan-com/xan-pythia/internal/model"
	"github.com/xan-com/xan-pythia/internal/repository"
)

func TestCustomerServiceCRUD(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	custRepo := repository.NewCustomerRepository(pool)
	svcRepo := repository.NewServiceRepository(pool)

	cust, _ := custRepo.Create(ctx, model.CreateCustomerInput{Name: "CS Test Customer"})
	svc, _ := svcRepo.Create(ctx, model.CreateServiceInput{Name: "Managed Firewall"})

	repo := repository.NewCustomerServiceRepository(pool)

	// Create
	input := model.CreateCustomerServiceInput{
		CustomerID:     cust.ID,
		ServiceID:      svc.ID,
		Customizations: json.RawMessage(`{"tier":"premium"}`),
	}
	cs, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !cs.ID.Valid {
		t.Error("ID should be valid")
	}

	// Get
	got, err := repo.GetByID(ctx, cs.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if !got.CustomerID.Valid {
		t.Error("CustomerID should be valid")
	}

	// List by customer
	list, err := repo.ListByCustomer(ctx, cust.ID, model.ListParams{Limit: 10})
	if err != nil {
		t.Fatalf("ListByCustomer: %v", err)
	}
	if len(list) == 0 {
		t.Error("ListByCustomer returned no results")
	}

	// Update
	newNotes := "Updated notes"
	updated, err := repo.Update(ctx, cs.ID, model.UpdateCustomerServiceInput{Notes: &newNotes})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Notes.String != "Updated notes" {
		t.Errorf("Notes = %q, want %q", updated.Notes.String, "Updated notes")
	}

	// Delete
	if err := repo.Delete(ctx, cs.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Cleanup
	_ = svcRepo.Delete(ctx, svc.ID)
	_ = custRepo.Delete(ctx, cust.ID)
}
```

- [ ] **Step 2: Implement customer_service repository**

```go
// internal/repository/customer_service.go
package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xan-com/xan-pythia/internal/model"
)

type CustomerServiceRepository struct {
	pool *pgxpool.Pool
}

func NewCustomerServiceRepository(pool *pgxpool.Pool) *CustomerServiceRepository {
	return &CustomerServiceRepository{pool: pool}
}

func (r *CustomerServiceRepository) Create(ctx context.Context, input model.CreateCustomerServiceInput) (model.CustomerService, error) {
	var cs model.CustomerService
	customizations := input.Customizations
	if customizations == nil {
		customizations = []byte("{}")
	}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO customer_services (customer_id, service_id, customizations, notes)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, customer_id, service_id, customizations, notes, created_at, updated_at`,
		input.CustomerID, input.ServiceID, customizations, input.Notes,
	).Scan(&cs.ID, &cs.CustomerID, &cs.ServiceID, &cs.Customizations, &cs.Notes, &cs.CreatedAt, &cs.UpdatedAt)
	if err != nil {
		return cs, fmt.Errorf("creating customer service: %w", err)
	}
	return cs, nil
}

func (r *CustomerServiceRepository) GetByID(ctx context.Context, id pgtype.UUID) (model.CustomerService, error) {
	var cs model.CustomerService
	err := r.pool.QueryRow(ctx,
		`SELECT id, customer_id, service_id, customizations, notes, created_at, updated_at
		 FROM customer_services WHERE id = $1`, id,
	).Scan(&cs.ID, &cs.CustomerID, &cs.ServiceID, &cs.Customizations, &cs.Notes, &cs.CreatedAt, &cs.UpdatedAt)
	if err != nil {
		return cs, fmt.Errorf("getting customer service: %w", err)
	}
	return cs, nil
}

func (r *CustomerServiceRepository) ListByCustomer(ctx context.Context, customerID pgtype.UUID, params model.ListParams) ([]model.CustomerService, error) {
	params.Normalize()
	rows, err := r.pool.Query(ctx,
		`SELECT id, customer_id, service_id, customizations, notes, created_at, updated_at
		 FROM customer_services WHERE customer_id = $1 ORDER BY created_at LIMIT $2 OFFSET $3`,
		customerID, params.Limit, params.Offset,
	)
	if err != nil {
		return nil, fmt.Errorf("listing customer services: %w", err)
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToStructByPos[model.CustomerService])
}

func (r *CustomerServiceRepository) Update(ctx context.Context, id pgtype.UUID, input model.UpdateCustomerServiceInput) (model.CustomerService, error) {
	var cs model.CustomerService
	err := r.pool.QueryRow(ctx,
		`UPDATE customer_services SET
			customizations = COALESCE($2, customizations),
			notes = COALESCE($3, notes)
		 WHERE id = $1
		 RETURNING id, customer_id, service_id, customizations, notes, created_at, updated_at`,
		id, input.Customizations, input.Notes,
	).Scan(&cs.ID, &cs.CustomerID, &cs.ServiceID, &cs.Customizations, &cs.Notes, &cs.CreatedAt, &cs.UpdatedAt)
	if err != nil {
		return cs, fmt.Errorf("updating customer service: %w", err)
	}
	return cs, nil
}

func (r *CustomerServiceRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM customer_services WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting customer service: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("customer service not found")
	}
	return nil
}
```

- [ ] **Step 3: Implement customer_service handler**

```go
// internal/handler/customer_service.go
package handler

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/xan-com/xan-pythia/internal/model"
	"github.com/xan-com/xan-pythia/internal/repository"
)

type CustomerServiceHandler struct {
	repo *repository.CustomerServiceRepository
}

func NewCustomerServiceHandler(repo *repository.CustomerServiceRepository) *CustomerServiceHandler {
	return &CustomerServiceHandler{repo: repo}
}

func (h *CustomerServiceHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/customers/{customerId}/services", h.list)
	mux.HandleFunc("POST /api/v1/customers/{customerId}/services", h.create)
	mux.HandleFunc("GET /api/v1/customer-services/{id}", h.get)
	mux.HandleFunc("PUT /api/v1/customer-services/{id}", h.update)
	mux.HandleFunc("DELETE /api/v1/customer-services/{id}", h.delete)
}

func (h *CustomerServiceHandler) list(w http.ResponseWriter, r *http.Request) {
	customerID, err := parseUUID(r.PathValue("customerId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer ID")
		return
	}
	params := paginationParams(r)
	services, err := h.repo.ListByCustomer(r.Context(), customerID, params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list customer services")
		return
	}
	writeJSON(w, http.StatusOK, services)
}

func (h *CustomerServiceHandler) create(w http.ResponseWriter, r *http.Request) {
	customerID, err := parseUUID(r.PathValue("customerId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer ID")
		return
	}
	var input model.CreateCustomerServiceInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input.CustomerID = customerID
	if !input.ServiceID.Valid {
		writeError(w, http.StatusBadRequest, "service_id is required")
		return
	}
	cs, err := h.repo.Create(r.Context(), input)
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "customer already subscribed to this service")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create customer service")
		return
	}
	writeJSON(w, http.StatusCreated, cs)
}

func (h *CustomerServiceHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer service ID")
		return
	}
	cs, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "customer service not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get customer service")
		return
	}
	writeJSON(w, http.StatusOK, cs)
}

func (h *CustomerServiceHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer service ID")
		return
	}
	var input model.UpdateCustomerServiceInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	cs, err := h.repo.Update(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "customer service not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update customer service")
		return
	}
	writeJSON(w, http.StatusOK, cs)
}

func (h *CustomerServiceHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid customer service ID")
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		if isFKViolation(err) {
			writeError(w, http.StatusConflict, "customer service has dependent records")
			return
		}
		writeError(w, http.StatusNotFound, "customer service not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 4: Run tests**

```bash
go vet ./...
go test ./internal/repository/... -v -run TestCustomerServiceCRUD
```

- [ ] **Step 5: Commit**

```bash
git add internal/repository/customer_service.go internal/repository/customer_service_test.go internal/handler/customer_service.go
git commit -m "feat: add customer service repository and handler"
```

---

## Phase 4: Wiring & Middleware

### Task 16: Logging Middleware

**Files:**
- Create: `internal/middleware/logging.go`

- [ ] **Step 1: Implement logging middleware**

```go
// internal/middleware/logging.go
package middleware

import (
	"log"
	"net/http"
	"time"
)

type wrappedWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *wrappedWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &wrappedWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, wrapped.statusCode, time.Since(start))
	})
}
```

- [ ] **Step 2: Verify compilation**

```bash
go vet ./internal/middleware/...
```

- [ ] **Step 3: Commit**

```bash
git add internal/middleware/logging.go
git commit -m "feat: add request logging middleware"
```

---

### Task 17: Wire Everything in main.go

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Update main.go with all handlers and middleware**

```go
// cmd/server/main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/xan-com/xan-pythia/internal/database"
	"github.com/xan-com/xan-pythia/internal/handler"
	"github.com/xan-com/xan-pythia/internal/middleware"
	"github.com/xan-com/xan-pythia/internal/repository"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	// Run migrations
	if err := database.RunMigrations(databaseURL, "db/migrations"); err != nil {
		log.Fatalf("running migrations: %v", err)
	}
	log.Println("Migrations completed")

	// Connect pool
	ctx := context.Background()
	pool, err := database.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer pool.Close()
	log.Println("Database connected")

	// Repositories
	customerRepo := repository.NewCustomerRepository(pool)
	userRepo := repository.NewUserRepository(pool)
	serviceRepo := repository.NewServiceRepository(pool)
	userAssignmentRepo := repository.NewUserAssignmentRepository(pool)
	assetRepo := repository.NewAssetRepository(pool)
	licenseRepo := repository.NewLicenseRepository(pool)
	customerServiceRepo := repository.NewCustomerServiceRepository(pool)

	// Router
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Register handlers
	handler.NewCustomerHandler(customerRepo).RegisterRoutes(mux)
	handler.NewUserHandler(userRepo).RegisterRoutes(mux)
	handler.NewServiceHandler(serviceRepo).RegisterRoutes(mux)
	handler.NewUserAssignmentHandler(userAssignmentRepo).RegisterRoutes(mux)
	handler.NewAssetHandler(assetRepo).RegisterRoutes(mux)
	handler.NewLicenseHandler(licenseRepo).RegisterRoutes(mux)
	handler.NewCustomerServiceHandler(customerServiceRepo).RegisterRoutes(mux)

	// Static files
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	// Middleware
	srv := middleware.Logging(mux)

	log.Printf("Starting server on :%s", port)
	if err := http.ListenAndServe(":"+port, srv); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./cmd/server/
```

- [ ] **Step 3: Full integration test with Docker**

```bash
docker compose up --build -d
# Wait for startup
sleep 5
# Test health
curl http://localhost:8080/health
# Test create customer
curl -X POST http://localhost:8080/api/v1/customers -H "Content-Type: application/json" -d '{"name":"Test Corp"}'
# Test list customers
curl http://localhost:8080/api/v1/customers
docker compose down
```

- [ ] **Step 4: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat: wire all handlers and middleware in main.go"
```

---

## Phase 5: Frontend (HTMX)

> **Note:** Frontend tasks are scoped as a follow-up plan. The REST API is fully functional after Phase 4. HTMX templates, layouts, and interactive views will be planned separately once the API is validated.

---

## Summary

| Phase | Tasks | What's built |
|-------|-------|-------------|
| 1: Foundation | 1-5 | Go project, Docker, DB connection, migrations (7 tables, triggers, indexes) |
| 2: Core Entities | 6-10 | Models + repo + handler for customers, users, services |
| 3: Dependent Entities | 11-15 | Models + repo + handler for user_assignments, assets, licenses, customer_services |
| 4: Wiring | 16-17 | Logging middleware, full main.go wiring |
| 5: Frontend | TBD | HTMX templates (separate plan) |

**Total: 17 tasks, ~85 steps**

Each task produces a working commit. Tests validate repository operations against a real PostgreSQL instance.
