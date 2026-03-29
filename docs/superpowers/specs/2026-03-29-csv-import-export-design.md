# CSV Import/Export Design Spec

**Date:** 2026-03-29
**Status:** Draft
**Scope:** Bulk data import/export via CSV for all XAN-Argus entities

## Overview

Add CSV import and export capabilities to XAN-Argus, enabling bulk data loading (initial migration, ongoing updates) and data extraction. A generic import engine with per-entity configuration handles parsing, validation, FK resolution, and upsert — all within a single database transaction (all-or-nothing).

## Requirements

- **Format:** CSV only (stdlib `encoding/csv`, no external dependencies). Comma-delimited, UTF-8. BOM stripped on import, BOM prepended on export (for Excel compatibility).
- **Entities:** All — customers, users, assets, licenses, services, customer_services, user_assignments, hardware_categories, category_field_definitions
- **Mode:** Upsert — creates new records or updates existing ones, matched by natural key
- **Error handling:** All-or-nothing — full validation before any DB writes; on errors, nothing is committed
- **FK references:** Name-based lookup where the referenced entity has a unique name constraint; UUID-based where not
- **JSONB fields:** Represented as JSON strings in a single CSV column
- **Export:** Full data export + empty template download per entity
- **Interface:** Web UI (import page + export buttons on list pages) + REST API

## Architecture

### New Package: `internal/importer/`

This is the one place where we deviate from the "no service layer" principle — import/export is genuinely cross-entity logic that doesn't fit in a single handler or repository.

```
internal/importer/
├── engine.go          # Generic CSV parse + validation + upsert orchestration
├── config.go          # EntityConfig type (columns, FK resolvers, match keys)
├── registry.go        # Registers all entity configs
├── resolver.go        # FK resolution (Name→UUID, with caching)
├── exporter.go        # CSV export + template generation
├── customers.go       # Config for customers
├── users.go           # Config for users
├── assets.go          # Config for assets
├── licenses.go        # Config for licenses
├── services.go        # Config for services
├── customer_services.go
├── user_assignments.go
├── hardware_categories.go
└── field_definitions.go
```

### Handler additions

```
internal/handler/
├── import.go          # API: POST /api/v1/import/{entity}, GET template, GET export
└── page.go            # New page routes for import UI
```

## Data Model

### EntityConfig

Each entity is described by a config struct that tells the engine how import/export works:

```go
type EntityConfig struct {
    Name      string         // "customers", "assets"
    Table     string         // DB table name
    MatchKeys []string       // Natural keys for upsert: ["name"]
    Columns   []ColumnConfig // Column definitions
}

type ColumnConfig struct {
    Header   string    // CSV header: "name", "customer", "metadata"
    DBColumn string    // DB column: "name", "customer_id", "metadata"
    Required bool      // Mandatory field?
    Type     string    // "text", "uuid", "int", "date", "bool", "json"
    FK       *FKConfig // If FK reference
}

type FKConfig struct {
    Table     string // Target table: "customers"
    LookupCol string // Lookup column: "name" (or "id" for UUID)
    Strategy  string // "name" or "uuid"
}
```

### All entity configs

**Match-key rules:** Match keys reference DB columns (post-FK-resolution). For entities whose natural key includes FK columns, the engine first resolves FKs to UUIDs in Phase 2, then uses the resolved UUIDs for match-key lookup in Phase 3.

**Customers** (simple, no FKs):
```go
EntityConfig{
    Name: "customers", Table: "customers",
    MatchKeys: []string{"name"},
    Columns: []ColumnConfig{
        {Header: "name", DBColumn: "name", Required: true, Type: "text"},
        {Header: "contact_email", DBColumn: "contact_email", Type: "text"},
        {Header: "notes", DBColumn: "notes", Type: "text"},
    },
}
```

**Hardware Categories** (simple, no FKs):
```go
EntityConfig{
    Name: "hardware-categories", Table: "hardware_categories",
    MatchKeys: []string{"name"},
    Columns: []ColumnConfig{
        {Header: "name", DBColumn: "name", Required: true, Type: "text"},
        {Header: "description", DBColumn: "description", Type: "text"},
    },
}
```

**Services** (simple, no FKs):
```go
EntityConfig{
    Name: "services", Table: "services",
    MatchKeys: []string{"name"},
    Columns: []ColumnConfig{
        {Header: "name", DBColumn: "name", Required: true, Type: "text"},
        {Header: "description", DBColumn: "description", Type: "text"},
    },
}
```

**Users** (no FKs, no unique name — match by first_name + last_name + type):
```go
EntityConfig{
    Name: "users", Table: "users",
    MatchKeys: []string{"first_name", "last_name", "type"},
    Columns: []ColumnConfig{
        {Header: "type", DBColumn: "type", Required: true, Type: "text"},
        {Header: "first_name", DBColumn: "first_name", Required: true, Type: "text"},
        {Header: "last_name", DBColumn: "last_name", Required: true, Type: "text"},
    },
}
```
Note: Users have no single unique name column. The composite match key (first_name + last_name + type) is a best-effort match. If duplicates exist, the import will fail with an ambiguity error. A migration adds a partial unique index on `(first_name, last_name, type) WHERE deleted_at IS NULL` to enforce this.

**Field Definitions** (FK to hardware_categories):
```go
EntityConfig{
    Name: "field-definitions", Table: "category_field_definitions",
    MatchKeys: []string{"category_id", "name"},
    Columns: []ColumnConfig{
        {Header: "category", DBColumn: "category_id", Required: true, Type: "uuid",
            FK: &FKConfig{Table: "hardware_categories", LookupCol: "name", Strategy: "name"}},
        {Header: "name", DBColumn: "name", Required: true, Type: "text"},
        {Header: "field_type", DBColumn: "field_type", Required: true, Type: "text"},
        {Header: "required", DBColumn: "required", Type: "bool"},
        {Header: "sort_order", DBColumn: "sort_order", Type: "int"},
    },
}
```

**User Assignments** (composite FK match key — user + customer):
```go
EntityConfig{
    Name: "user-assignments", Table: "user_assignments",
    MatchKeys: []string{"user_id", "customer_id"},
    Columns: []ColumnConfig{
        {Header: "user_id", DBColumn: "user_id", Required: true, Type: "uuid",
            FK: &FKConfig{Table: "users", LookupCol: "id", Strategy: "uuid"}},
        {Header: "customer", DBColumn: "customer_id", Required: true, Type: "uuid",
            FK: &FKConfig{Table: "customers", LookupCol: "name", Strategy: "name"}},
        {Header: "role", DBColumn: "role", Type: "text"},
        {Header: "email", DBColumn: "email", Type: "text"},
        {Header: "phone", DBColumn: "phone", Type: "text"},
        {Header: "notes", DBColumn: "notes", Type: "text"},
    },
}
```
Note: `user_id` requires UUID because users have no single unique name. When exporting, user_id is exported as UUID. The export includes a comment row or separate user-export reference so the user can cross-reference.

**Customer Services** (composite FK match key):
```go
EntityConfig{
    Name: "customer-services", Table: "customer_services",
    MatchKeys: []string{"customer_id", "service_id"},
    Columns: []ColumnConfig{
        {Header: "customer", DBColumn: "customer_id", Required: true, Type: "uuid",
            FK: &FKConfig{Table: "customers", LookupCol: "name", Strategy: "name"}},
        {Header: "service", DBColumn: "service_id", Required: true, Type: "uuid",
            FK: &FKConfig{Table: "services", LookupCol: "name", Strategy: "name"}},
        {Header: "customizations", DBColumn: "customizations", Type: "json"},
        {Header: "notes", DBColumn: "notes", Type: "text"},
    },
}
```

**Assets** (multiple FKs, optional user_assignment):
```go
EntityConfig{
    Name: "assets", Table: "assets",
    MatchKeys: []string{"name", "customer_id"},
    Columns: []ColumnConfig{
        {Header: "name", DBColumn: "name", Required: true, Type: "text"},
        {Header: "description", DBColumn: "description", Type: "text"},
        {Header: "customer", DBColumn: "customer_id", Required: true, Type: "uuid",
            FK: &FKConfig{Table: "customers", LookupCol: "name", Strategy: "name"}},
        {Header: "category", DBColumn: "category_id", Type: "uuid",
            FK: &FKConfig{Table: "hardware_categories", LookupCol: "name", Strategy: "name"}},
        {Header: "user_assignment_id", DBColumn: "user_assignment_id", Type: "uuid",
            FK: &FKConfig{Table: "user_assignments", LookupCol: "id", Strategy: "uuid"}},
        {Header: "metadata", DBColumn: "metadata", Type: "json"},
        {Header: "field_values", DBColumn: "field_values", Type: "json"},
    },
}
```

**Licenses** (match by product_name + customer + license_key):
```go
EntityConfig{
    Name: "licenses", Table: "licenses",
    MatchKeys: []string{"product_name", "customer_id", "license_key"},
    Columns: []ColumnConfig{
        {Header: "product_name", DBColumn: "product_name", Required: true, Type: "text"},
        {Header: "license_key", DBColumn: "license_key", Required: true, Type: "text"},
        {Header: "customer", DBColumn: "customer_id", Required: true, Type: "uuid",
            FK: &FKConfig{Table: "customers", LookupCol: "name", Strategy: "name"}},
        {Header: "user_assignment_id", DBColumn: "user_assignment_id", Type: "uuid",
            FK: &FKConfig{Table: "user_assignments", LookupCol: "id", Strategy: "uuid"}},
        {Header: "quantity", DBColumn: "quantity", Type: "int"},
        {Header: "valid_from", DBColumn: "valid_from", Type: "date"},
        {Header: "valid_until", DBColumn: "valid_until", Type: "date"},
    },
}
```
Note: Licenses have no existing unique constraint for this combination. A migration adds a partial unique index on `(product_name, customer_id, license_key) WHERE deleted_at IS NULL`.

### Import result types

```go
type ImportResult struct {
    Total   int           `json:"total"`
    Created int           `json:"created"`
    Updated int           `json:"updated"`
    Errors  []ImportError `json:"errors"`
}

type ImportError struct {
    Row     int    `json:"row"`
    Column  string `json:"column"`
    Message string `json:"message"`
}
```

## Import Engine Flow

### Phase 1: Parse & Validate Headers
- Read CSV, match header row against `EntityConfig.Columns`
- Missing required columns → immediate abort with clear message
- Unknown columns → warning, ignored

### Phase 2: Row Processing (no DB writes yet)
For each row:
- Type validation (int, date, bool, JSON parsing)
- Required field check (empty required fields)
- FK resolution: Name→UUID via DB lookup (cached per import so e.g. "Firma ABC" is looked up only once)
- Result: list of `ImportRow` with resolved values, or list of errors

### Phase 3: Match-Key Lookup
For each row:
- Does a record with this natural key exist? (uses post-FK-resolution values)
  - `SELECT id FROM {table} WHERE {match_key_1} = $1 AND {match_key_2} = $2 AND ... AND deleted_at IS NULL`
- If yes → mark as UPDATE (with existing ID)
- If no → mark as INSERT
- If multiple matches → error (should not happen with proper unique indexes, but guard against it)

### Phase 4: Execute (single transaction)

SQL strategy: **separate INSERT and UPDATE statements** (not `ON CONFLICT`). Phase 3 already determined which rows are creates vs updates, so Phase 4 executes them directly.

- `BEGIN`
- For each INSERT row:
  ```sql
  INSERT INTO {table} ({col1}, {col2}, ...) VALUES ($1, $2, ...)
  ```
- For each UPDATE row:
  ```sql
  UPDATE {table} SET {col1} = $1, {col2} = $2, ... WHERE id = $N AND deleted_at IS NULL
  ```
- On error → `ROLLBACK`, return error list
- All OK → `COMMIT`

This approach is chosen over `INSERT ... ON CONFLICT` because:
- Match keys may not all have unique indexes yet (migration adds the missing ones)
- Separate statements give clearer error messages per row
- Phase 3 already does the lookup, so `ON CONFLICT` would duplicate work

If Phase 2/3 produced errors, Phase 4 is never started — user gets all validation errors at once.

### Required migrations

The following partial unique indexes must be added to support upsert match keys:

```sql
-- Services: no unique index exists yet (unlike customers/hardware_categories)
CREATE UNIQUE INDEX idx_services_name_active
    ON services(name) WHERE deleted_at IS NULL;

-- Users: composite natural key for import matching
CREATE UNIQUE INDEX idx_users_name_type_active
    ON users(first_name, last_name, type) WHERE deleted_at IS NULL;

-- Assets: composite natural key for import matching
CREATE UNIQUE INDEX idx_assets_name_customer_active
    ON assets(name, customer_id) WHERE deleted_at IS NULL;

-- Licenses: composite natural key for import matching
CREATE UNIQUE INDEX idx_licenses_product_customer_key_active
    ON licenses(product_name, customer_id, license_key) WHERE deleted_at IS NULL;
```

Existing unique indexes already cover: customers (name), hardware_categories (name), category_field_definitions (category_id, name), user_assignments (user_id, customer_id), customer_services (customer_id, service_id).

## FK Resolver with Caching

```go
type Resolver struct {
    pool  *pgxpool.Pool
    cache map[string]map[string]pgtype.UUID  // "table:lookupValue" → UUID
}
```

**Flow:**
1. First row references customer "Firma ABC" → DB query: `SELECT id FROM customers WHERE name = $1 AND deleted_at IS NULL`
2. Result cached as `customers:Firma ABC → uuid`
3. All subsequent rows with "Firma ABC" → cache hit, no DB query

**Error case:** Name not found → `ImportError{Row: N, Column: "customer", Message: "customer 'Firma XYZ' not found"}`

**UUID strategy (for user references etc.):** When `Strategy: "uuid"`, the value is parsed as UUID and only checked for existence: `SELECT id FROM users WHERE id = $1 AND deleted_at IS NULL`

## Export & Templates

### Template download
- `GET /api/v1/import/{entity}/template` → empty CSV with correct headers
- Headers from `EntityConfig.Columns[].Header`
- Example for customers: `name,contact_email,notes`

### Data export
- `GET /api/v1/export/{entity}` → CSV with all active records
- Same column order as template
- FK columns reverse-resolved where the FK config uses `Strategy: "name"`:
  - `customer_id` → customer name
  - `category_id` → category name
  - `service_id` → service name
  - `category_id` (in field_definitions) → category name
- FK columns exported as UUID where `Strategy: "uuid"`:
  - `user_id` → UUID (users have no single unique name)
  - `user_assignment_id` → UUID
- JSONB fields as JSON strings
- Export CSV can be re-used directly as import CSV (round-trip — the format matches what import expects)

### HTTP headers
```
Content-Type: text/csv
Content-Disposition: attachment; filename="customers_2026-03-29.csv"
```

## API Design

| Method | Route | Description |
|--------|-------|-------------|
| `POST` | `/api/v1/import/{entity}` | CSV upload, multipart/form-data, field name: `file` |
| `GET` | `/api/v1/import/{entity}/template` | Empty CSV template |
| `GET` | `/api/v1/export/{entity}` | Data export as CSV |

`{entity}` values: `customers`, `users`, `assets`, `licenses`, `services`, `customer-services`, `user-assignments`, `hardware-categories`, `field-definitions`

### Import response

```json
// Success (200):
{"total": 15, "created": 10, "updated": 5, "errors": []}

// Validation errors (400):
{"total": 15, "created": 0, "updated": 0, "errors": [
  {"row": 3, "column": "name", "message": "name is required"},
  {"row": 7, "column": "customer", "message": "customer 'Unknown GmbH' not found"}
]}
```

## Web UI

### Import page (`/import`)
- Entity dropdown selector
- Template download link (updates with dropdown selection)
- File upload field for CSV
- Import button
- Result area: success (created/updated counts) or error table (row, column, message)
- Dependency order hint showing recommended import sequence

### Export integration
- Export CSV button on each existing list page
- Links directly to `/api/v1/export/{entity}`

### Navigation
- New "Import" entry in navbar

## Import Dependency Order

Entities must be imported in this order to satisfy FK dependencies:

```
1. customers              (no FK dependencies)
2. hardware_categories    (no FK dependencies)
3. services               (no FK dependencies)
4. users                  (no FK dependencies)
5. field_definitions      (→ hardware_categories)
6. user_assignments       (→ users, customers)
7. customer_services      (→ customers, services)
8. assets                 (→ customers, hardware_categories, user_assignments)
9. licenses               (→ customers, user_assignments)
```

The import page displays this order as guidance so users know which base entities must be imported first.

## Non-Goals

- Multi-entity batch import (single file with multiple entities) — user imports one entity at a time
- Excel (.xlsx) support — users export to CSV from Excel
- Async/background processing — imports are synchronous (sufficient for expected data volumes)
- Import history/audit — the existing audit_log table captures all INSERTs/UPDATEs from imports automatically
- Authentication/authorization — deferred per project scope
