# CSV Import/Export Design Spec

**Date:** 2026-03-29
**Status:** Draft
**Scope:** Bulk data import/export via CSV for all XAN-Argus entities

## Overview

Add CSV import and export capabilities to XAN-Argus, enabling bulk data loading (initial migration, ongoing updates) and data extraction. A generic import engine with per-entity configuration handles parsing, validation, FK resolution, and upsert — all within a single database transaction (all-or-nothing).

## Requirements

- **Format:** CSV only (stdlib `encoding/csv`, no external dependencies)
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

### Example configs

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

**Assets** (with FKs):
```go
EntityConfig{
    Name: "assets", Table: "assets",
    MatchKeys: []string{"name", "customer_id"},
    Columns: []ColumnConfig{
        {Header: "name", DBColumn: "name", Required: true, Type: "text"},
        {Header: "customer", DBColumn: "customer_id", Required: true, Type: "uuid",
            FK: &FKConfig{Table: "customers", LookupCol: "name", Strategy: "name"}},
        {Header: "category", DBColumn: "category_id", Type: "uuid",
            FK: &FKConfig{Table: "hardware_categories", LookupCol: "name", Strategy: "name"}},
        {Header: "metadata", DBColumn: "metadata", Type: "json"},
        {Header: "field_values", DBColumn: "field_values", Type: "json"},
    },
}
```

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
- Does a record with this natural key exist?
  - `SELECT id FROM {table} WHERE {match_key_1} = $1 AND ... AND deleted_at IS NULL`
- If yes → mark as UPDATE (with existing ID)
- If no → mark as INSERT

### Phase 4: Execute (single transaction)
- `BEGIN`
- Execute all INSERTs and UPDATEs
- SQL generated dynamically from EntityConfig
- On error → `ROLLBACK`, return error list
- All OK → `COMMIT`

If Phase 2/3 produced errors, Phase 4 is never started — user gets all validation errors at once.

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
- FK columns reverse-resolved: UUID→Name where possible
- JSONB fields as JSON strings
- Export CSV can be re-used directly as import CSV (round-trip)

### HTTP headers
```
Content-Type: text/csv
Content-Disposition: attachment; filename="customers_2026-03-29.csv"
```

## API Design

| Method | Route | Description |
|--------|-------|-------------|
| `POST` | `/api/v1/import/{entity}` | CSV upload, multipart/form-data |
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
