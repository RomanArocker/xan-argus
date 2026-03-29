# CSV Import/Export Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add CSV import (upsert) and export for all entities, with a web UI and REST API.

**Architecture:** A new `internal/importer/` package provides a generic import engine driven by per-entity configs (columns, FK resolvers, match keys). A single `ImportHandler` exposes the REST API. The `PageHandler` gets a new import page. Export buttons are added to existing list pages.

**Tech Stack:** Go stdlib (`encoding/csv`, `bytes`, `io`), pgx transactions, HTMX for the UI.

**Spec:** `docs/superpowers/specs/2026-03-29-csv-import-export-design.md`

---

## File Map

### New files

| File | Responsibility |
|------|---------------|
| `db/migrations/009_import_indexes.sql` | Unique indexes for upsert match keys |
| `internal/importer/config.go` | `EntityConfig`, `ColumnConfig`, `FKConfig` types |
| `internal/importer/registry.go` | `Registry` — registers all entity configs, lookup by name |
| `internal/importer/resolver.go` | `Resolver` — FK name→UUID resolution with caching |
| `internal/importer/engine.go` | `Engine` — CSV parse, validate, match-key lookup, upsert execution |
| `internal/importer/exporter.go` | `Exporter` — CSV export + template generation |
| `internal/importer/customers.go` | Customer entity config |
| `internal/importer/hardware_categories.go` | Hardware categories entity config |
| `internal/importer/services.go` | Services entity config |
| `internal/importer/users.go` | Users entity config |
| `internal/importer/field_definitions.go` | Field definitions entity config |
| `internal/importer/user_assignments.go` | User assignments entity config |
| `internal/importer/customer_services.go` | Customer services entity config |
| `internal/importer/assets.go` | Assets entity config |
| `internal/importer/licenses.go` | Licenses entity config |
| `internal/handler/import.go` | `ImportHandler` — REST API for import/export/template |
| `web/templates/import/page.html` | Import page template |
| `internal/importer/engine_test.go` | Tests for engine (CSV parsing, validation) |
| `internal/importer/resolver_test.go` | Tests for FK resolver |
| `internal/importer/exporter_test.go` | Tests for CSV export |

### Modified files

| File | Change |
|------|--------|
| `cmd/server/main.go` | Wire `ImportHandler`, pass pool to importer |
| `internal/handler/page.go` | Add `importPage` route + handler |
| `internal/handler/template.go` | Register `import/page.html` |
| `web/templates/layout.html` | Add "Import" nav link |
| `web/templates/customers/list.html` | Add export CSV button |
| `web/templates/users/list.html` | Add export CSV button |
| `web/templates/services/list.html` | Add export CSV button |
| `web/templates/categories/list.html` | Add export CSV button |

---

## Task 1: Database migration — unique indexes for upsert match keys

**Files:**
- Create: `db/migrations/009_import_indexes.sql`

- [ ] **Step 1: Write the migration file**

```sql
-- +goose Up

-- Services: no unique index exists yet (customers and hardware_categories already have one)
CREATE UNIQUE INDEX idx_services_name_active ON services(name) WHERE deleted_at IS NULL;

-- Users: composite natural key for import matching
CREATE UNIQUE INDEX idx_users_name_type_active ON users(first_name, last_name, type) WHERE deleted_at IS NULL;

-- Assets: composite natural key for import matching
CREATE UNIQUE INDEX idx_assets_name_customer_active ON assets(name, customer_id) WHERE deleted_at IS NULL;

-- Licenses: composite natural key for import matching
CREATE UNIQUE INDEX idx_licenses_product_customer_key_active ON licenses(product_name, customer_id, license_key) WHERE deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_licenses_product_customer_key_active;
DROP INDEX IF EXISTS idx_assets_name_customer_active;
DROP INDEX IF EXISTS idx_users_name_type_active;
DROP INDEX IF EXISTS idx_services_name_active;
```

- [ ] **Step 2: Verify migration syntax**

Run: `go vet ./...`
Expected: passes (migrations are SQL, but this confirms no Go breakage)

- [ ] **Step 3: Commit**

```bash
git add db/migrations/009_import_indexes.sql
git commit -m "feat: add unique indexes for CSV import upsert match keys"
```

---

## Task 2: Core types — EntityConfig, ColumnConfig, FKConfig

**Files:**
- Create: `internal/importer/config.go`

- [ ] **Step 1: Create the config types**

```go
package importer

// EntityConfig describes how an entity is imported/exported via CSV.
type EntityConfig struct {
	Name      string         // URL-safe name: "customers", "user-assignments"
	Table     string         // DB table: "customers", "user_assignments"
	MatchKeys []string       // DB columns for upsert match: ["name"] or ["customer_id", "service_id"]
	Columns   []ColumnConfig // Ordered column definitions
}

// ColumnConfig describes a single CSV column and its mapping to a DB column.
type ColumnConfig struct {
	Header   string    // CSV header name: "name", "customer", "metadata"
	DBColumn string    // DB column name: "name", "customer_id", "metadata"
	Required bool      // Must be non-empty in CSV
	Type     string    // "text", "uuid", "int", "date", "bool", "json"
	FK       *FKConfig // Non-nil if this column is a foreign key reference
}

// FKConfig describes how a CSV value maps to a foreign key UUID.
type FKConfig struct {
	Table     string // Referenced table: "customers"
	LookupCol string // Column to match: "name" (for name strategy) or "id" (for uuid strategy)
	Strategy  string // "name" = lookup by name, "uuid" = parse as UUID and verify existence
}

// ImportResult is returned by the engine after an import attempt.
type ImportResult struct {
	Total   int           `json:"total"`
	Created int           `json:"created"`
	Updated int           `json:"updated"`
	Errors  []ImportError `json:"errors"`
}

// ImportError describes a validation or execution error for a specific CSV row.
type ImportError struct {
	Row     int    `json:"row"`
	Column  string `json:"column"`
	Message string `json:"message"`
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go vet ./...`
Expected: passes

- [ ] **Step 3: Commit**

```bash
git add internal/importer/config.go
git commit -m "feat: add importer config types (EntityConfig, ColumnConfig, FKConfig)"
```

---

## Task 3: Registry — register and look up entity configs

**Files:**
- Create: `internal/importer/registry.go`

- [ ] **Step 1: Create the registry**

```go
package importer

import "fmt"

// Registry holds all entity configs and provides lookup by name.
type Registry struct {
	configs map[string]*EntityConfig
	order   []string // dependency order for UI display
}

// NewRegistry creates a registry with all entity configs pre-registered.
func NewRegistry() *Registry {
	r := &Registry{
		configs: make(map[string]*EntityConfig),
		order: []string{
			"customers",
			"hardware-categories",
			"services",
			"users",
			"field-definitions",
			"user-assignments",
			"customer-services",
			"assets",
			"licenses",
		},
	}
	r.register(customersConfig())
	r.register(hardwareCategoriesConfig())
	r.register(servicesConfig())
	r.register(usersConfig())
	r.register(fieldDefinitionsConfig())
	r.register(userAssignmentsConfig())
	r.register(customerServicesConfig())
	r.register(assetsConfig())
	r.register(licensesConfig())
	return r
}

func (r *Registry) register(cfg *EntityConfig) {
	r.configs[cfg.Name] = cfg
}

// Get returns the config for the given entity name, or an error if not found.
func (r *Registry) Get(name string) (*EntityConfig, error) {
	cfg, ok := r.configs[name]
	if !ok {
		return nil, fmt.Errorf("unknown entity: %s", name)
	}
	return cfg, nil
}

// OrderedNames returns entity names in dependency order (for UI display).
func (r *Registry) OrderedNames() []string {
	return r.order
}
```

Note: This will not compile yet because the config functions (`customersConfig()`, etc.) don't exist. That's OK — they come in Task 5. For now, comment out the register calls and `order` entries, or create stubs. Alternatively, implement Task 3 and Task 5 together.

- [ ] **Step 2: Commit**

```bash
git add internal/importer/registry.go
git commit -m "feat: add importer registry for entity config lookup"
```

---

## Task 4: FK Resolver — name-to-UUID resolution with caching

**Files:**
- Create: `internal/importer/resolver.go`

- [ ] **Step 1: Create the resolver**

```go
package importer

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Resolver handles FK resolution from CSV values to UUIDs.
// It caches lookups per import to avoid repeated DB queries.
type Resolver struct {
	pool  *pgxpool.Pool
	cache map[string]pgtype.UUID // "table:value" → UUID
}

// NewResolver creates a new resolver with an empty cache.
func NewResolver(pool *pgxpool.Pool) *Resolver {
	return &Resolver{
		pool:  pool,
		cache: make(map[string]pgtype.UUID),
	}
}

// Resolve looks up a FK value and returns the referenced UUID.
// For "name" strategy: SELECT id FROM table WHERE lookupCol = value AND deleted_at IS NULL
// For "uuid" strategy: parse value as UUID and verify existence.
func (r *Resolver) Resolve(ctx context.Context, fk *FKConfig, value string) (pgtype.UUID, error) {
	cacheKey := fk.Table + ":" + value

	if cached, ok := r.cache[cacheKey]; ok {
		return cached, nil
	}

	var id pgtype.UUID

	if fk.Strategy == "uuid" {
		if err := id.Scan(value); err != nil {
			return id, fmt.Errorf("invalid UUID: %s", value)
		}
		// Verify existence
		var exists bool
		err := r.pool.QueryRow(ctx,
			fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE id = $1 AND deleted_at IS NULL)", fk.Table),
			id,
		).Scan(&exists)
		if err != nil {
			return id, fmt.Errorf("checking %s existence: %w", fk.Table, err)
		}
		if !exists {
			return id, fmt.Errorf("%s with ID '%s' not found", fk.Table, value)
		}
	} else {
		// Name-based lookup
		err := r.pool.QueryRow(ctx,
			fmt.Sprintf("SELECT id FROM %s WHERE %s = $1 AND deleted_at IS NULL", fk.Table, fk.LookupCol),
			value,
		).Scan(&id)
		if err != nil {
			if err == pgx.ErrNoRows {
				return id, fmt.Errorf("%s '%s' not found", fk.Table, value)
			}
			return id, fmt.Errorf("resolving %s '%s': %w", fk.Table, value, err)
		}
	}

	r.cache[cacheKey] = id
	return id, nil
}
```

Note on SQL injection safety: `fk.Table` and `fk.LookupCol` come from hardcoded `EntityConfig` structs, never from user input. They are safe to interpolate. The `value` is always passed as a parameter `$1`.

Note on ownership: The `Resolver` is created internally by the `Engine` per import call (see `Engine.Import()` in Task 6). It is not wired in `main.go` — the Engine owns and manages it.

- [ ] **Step 2: Verify it compiles**

Run: `go vet ./...`
Expected: passes

- [ ] **Step 3: Commit**

```bash
git add internal/importer/resolver.go
git commit -m "feat: add FK resolver with caching for CSV import"
```

Note: `resolver_test.go` is listed in the file map but resolver tests require a live DB connection (FK lookups hit PostgreSQL). These are covered by the integration tests in Task 11. Unit tests for the resolver's caching logic can be added later if needed.

---

## Task 5: All entity configs

**Files:**
- Create: `internal/importer/customers.go`
- Create: `internal/importer/hardware_categories.go`
- Create: `internal/importer/services.go`
- Create: `internal/importer/users.go`
- Create: `internal/importer/field_definitions.go`
- Create: `internal/importer/user_assignments.go`
- Create: `internal/importer/customer_services.go`
- Create: `internal/importer/assets.go`
- Create: `internal/importer/licenses.go`

- [ ] **Step 1: Create customers.go**

```go
package importer

func customersConfig() *EntityConfig {
	return &EntityConfig{
		Name:      "customers",
		Table:     "customers",
		MatchKeys: []string{"name"},
		Columns: []ColumnConfig{
			{Header: "name", DBColumn: "name", Required: true, Type: "text"},
			{Header: "contact_email", DBColumn: "contact_email", Type: "text"},
			{Header: "notes", DBColumn: "notes", Type: "text"},
		},
	}
}
```

- [ ] **Step 2: Create hardware_categories.go**

```go
package importer

func hardwareCategoriesConfig() *EntityConfig {
	return &EntityConfig{
		Name:      "hardware-categories",
		Table:     "hardware_categories",
		MatchKeys: []string{"name"},
		Columns: []ColumnConfig{
			{Header: "name", DBColumn: "name", Required: true, Type: "text"},
			{Header: "description", DBColumn: "description", Type: "text"},
		},
	}
}
```

- [ ] **Step 3: Create services.go**

```go
package importer

func servicesConfig() *EntityConfig {
	return &EntityConfig{
		Name:      "services",
		Table:     "services",
		MatchKeys: []string{"name"},
		Columns: []ColumnConfig{
			{Header: "name", DBColumn: "name", Required: true, Type: "text"},
			{Header: "description", DBColumn: "description", Type: "text"},
		},
	}
}
```

- [ ] **Step 4: Create users.go**

```go
package importer

func usersConfig() *EntityConfig {
	return &EntityConfig{
		Name:      "users",
		Table:     "users",
		MatchKeys: []string{"first_name", "last_name", "type"},
		Columns: []ColumnConfig{
			{Header: "type", DBColumn: "type", Required: true, Type: "text"},
			{Header: "first_name", DBColumn: "first_name", Required: true, Type: "text"},
			{Header: "last_name", DBColumn: "last_name", Required: true, Type: "text"},
		},
	}
}
```

- [ ] **Step 5: Create field_definitions.go**

```go
package importer

func fieldDefinitionsConfig() *EntityConfig {
	return &EntityConfig{
		Name:      "field-definitions",
		Table:     "category_field_definitions",
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
}
```

- [ ] **Step 6: Create user_assignments.go**

```go
package importer

func userAssignmentsConfig() *EntityConfig {
	return &EntityConfig{
		Name:      "user-assignments",
		Table:     "user_assignments",
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
}
```

- [ ] **Step 7: Create customer_services.go**

```go
package importer

func customerServicesConfig() *EntityConfig {
	return &EntityConfig{
		Name:      "customer-services",
		Table:     "customer_services",
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
}
```

- [ ] **Step 8: Create assets.go**

```go
package importer

func assetsConfig() *EntityConfig {
	return &EntityConfig{
		Name:      "assets",
		Table:     "assets",
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
}
```

- [ ] **Step 9: Create licenses.go**

```go
package importer

func licensesConfig() *EntityConfig {
	return &EntityConfig{
		Name:      "licenses",
		Table:     "licenses",
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
}
```

- [ ] **Step 10: Verify all configs compile and registry works**

Run: `go vet ./...`
Expected: passes

- [ ] **Step 11: Commit**

```bash
git add internal/importer/
git commit -m "feat: add all entity configs and registry for CSV import"
```

---

## Task 6: Import Engine — CSV parsing, validation, and upsert execution

**Files:**
- Create: `internal/importer/engine.go`
- Create: `internal/importer/engine_test.go`

This is the core of the import system. The engine:
1. Parses CSV headers and maps them to `EntityConfig.Columns`
2. Validates each row (types, required fields)
3. Resolves FK references via the Resolver
4. Looks up match keys to determine INSERT vs UPDATE
5. Executes all changes in a single transaction

- [ ] **Step 1: Write a test for CSV header validation**

Create `internal/importer/engine_test.go`:

```go
package importer

import (
	"strings"
	"testing"
)

func TestParseHeaders_Valid(t *testing.T) {
	cfg := customersConfig()
	csv := "name,contact_email,notes\n"
	colMap, err := parseHeaders(strings.NewReader(csv), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(colMap) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(colMap))
	}
}

func TestParseHeaders_MissingRequired(t *testing.T) {
	cfg := customersConfig()
	csv := "contact_email,notes\n"
	_, err := parseHeaders(strings.NewReader(csv), cfg)
	if err == nil {
		t.Fatal("expected error for missing required column 'name'")
	}
}

func TestParseHeaders_UnknownColumnsIgnored(t *testing.T) {
	cfg := customersConfig()
	csv := "name,contact_email,notes,extra_col\n"
	colMap, err := parseHeaders(strings.NewReader(csv), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// extra_col should be ignored — only 3 mapped columns
	if len(colMap) != 3 {
		t.Fatalf("expected 3 columns, got %d", len(colMap))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/importer/ -run TestParseHeaders -v`
Expected: FAIL (parseHeaders not defined)

- [ ] **Step 3: Implement the engine**

Create `internal/importer/engine.go`. This is a large file — the key sections are:

```go
package importer

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Engine orchestrates CSV import for any entity.
type Engine struct {
	pool     *pgxpool.Pool
	registry *Registry
}

// NewEngine creates a new import engine.
func NewEngine(pool *pgxpool.Pool, registry *Registry) *Engine {
	return &Engine{pool: pool, registry: registry}
}

// columnMapping maps a CSV column index to its ColumnConfig.
type columnMapping struct {
	csvIndex int
	config   ColumnConfig
}

// importRow holds the resolved values for a single CSV row.
type importRow struct {
	csvRow   int                    // 1-based row number (excluding header)
	values   map[string]interface{} // dbColumn → resolved value
	existsID *pgtype.UUID           // non-nil if UPDATE, nil if INSERT
}

// stripBOM removes a UTF-8 BOM from the beginning of the reader if present.
func stripBOM(data []byte) []byte {
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return data[3:]
	}
	return data
}

// parseHeaders reads the first CSV row and maps headers to column configs.
// Returns a list of column mappings (CSV index → ColumnConfig).
func parseHeaders(r io.Reader, cfg *EntityConfig) ([]columnMapping, error) {
	reader := csv.NewReader(r)
	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("reading CSV headers: %w", err)
	}

	// Build lookup: header name → ColumnConfig
	configByHeader := make(map[string]ColumnConfig)
	for _, col := range cfg.Columns {
		configByHeader[col.Header] = col
	}

	// Map CSV columns to configs
	var mappings []columnMapping
	for i, h := range headers {
		h = strings.TrimSpace(h)
		if col, ok := configByHeader[h]; ok {
			mappings = append(mappings, columnMapping{csvIndex: i, config: col})
		}
		// Unknown columns are silently ignored
	}

	// Check all required columns are present
	mapped := make(map[string]bool)
	for _, m := range mappings {
		mapped[m.config.Header] = true
	}
	for _, col := range cfg.Columns {
		if col.Required && !mapped[col.Header] {
			return nil, fmt.Errorf("missing required column: %s", col.Header)
		}
	}

	return mappings, nil
}

// Import processes a CSV file and upserts records for the given entity.
func (e *Engine) Import(ctx context.Context, entityName string, csvData []byte) (*ImportResult, error) {
	cfg, err := e.registry.Get(entityName)
	if err != nil {
		return nil, err
	}

	// Strip BOM
	csvData = stripBOM(csvData)

	// Phase 1: Parse headers
	mappings, err := parseHeaders(bytes.NewReader(csvData), cfg)
	if err != nil {
		return &ImportResult{Errors: []ImportError{{Row: 0, Message: err.Error()}}}, nil
	}

	// Phase 2: Parse and validate all rows
	reader := csv.NewReader(bytes.NewReader(csvData))
	_, _ = reader.Read() // skip header (already parsed)

	resolver := NewResolver(e.pool)
	var rows []importRow
	var errors []ImportError
	rowNum := 1

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			errors = append(errors, ImportError{Row: rowNum, Message: fmt.Sprintf("CSV parse error: %v", err)})
			rowNum++
			continue
		}

		row := importRow{csvRow: rowNum, values: make(map[string]interface{})}

		for _, m := range mappings {
			val := ""
			if m.csvIndex < len(record) {
				val = strings.TrimSpace(record[m.csvIndex])
			}

			// Required check
			if m.config.Required && val == "" {
				errors = append(errors, ImportError{Row: rowNum, Column: m.config.Header, Message: fmt.Sprintf("%s is required", m.config.Header)})
				continue
			}

			// Empty optional → nil
			if val == "" {
				row.values[m.config.DBColumn] = nil
				continue
			}

			// Type validation + FK resolution
			parsed, err := parseValue(ctx, val, m.config, resolver)
			if err != nil {
				errors = append(errors, ImportError{Row: rowNum, Column: m.config.Header, Message: err.Error()})
				continue
			}
			row.values[m.config.DBColumn] = parsed
		}

		rows = append(rows, row)
		rowNum++
	}

	if len(errors) > 0 {
		return &ImportResult{Total: rowNum - 1, Errors: errors}, nil
	}

	// Phase 3: Match-key lookup
	for i := range rows {
		id, err := lookupMatchKey(ctx, e.pool, cfg, &rows[i])
		if err != nil {
			errors = append(errors, ImportError{Row: rows[i].csvRow, Message: fmt.Sprintf("match key lookup: %v", err)})
			continue
		}
		rows[i].existsID = id
	}

	if len(errors) > 0 {
		return &ImportResult{Total: len(rows), Errors: errors}, nil
	}

	// Phase 4: Execute in transaction
	result := &ImportResult{Total: len(rows)}
	tx, err := e.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, row := range rows {
		if row.existsID != nil {
			if err := executeUpdate(ctx, tx, cfg, &row); err != nil {
				return nil, fmt.Errorf("row %d: %w", row.csvRow, err)
			}
			result.Updated++
		} else {
			if err := executeInsert(ctx, tx, cfg, &row); err != nil {
				return nil, fmt.Errorf("row %d: %w", row.csvRow, err)
			}
			result.Created++
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing transaction: %w", err)
	}

	return result, nil
}

// parseValue converts a CSV string to the appropriate Go type.
func parseValue(ctx context.Context, val string, col ColumnConfig, resolver *Resolver) (interface{}, error) {
	// FK resolution first (if applicable)
	if col.FK != nil {
		id, err := resolver.Resolve(ctx, col.FK, val)
		if err != nil {
			return nil, err
		}
		return id, nil
	}

	switch col.Type {
	case "text":
		return val, nil
	case "uuid":
		var id pgtype.UUID
		if err := id.Scan(val); err != nil {
			return nil, fmt.Errorf("invalid UUID: %s", val)
		}
		return id, nil
	case "int":
		n, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid integer: %s", val)
		}
		return n, nil
	case "bool":
		switch strings.ToLower(val) {
		case "true", "1", "yes":
			return true, nil
		case "false", "0", "no":
			return false, nil
		default:
			return nil, fmt.Errorf("invalid boolean: %s", val)
		}
	case "date":
		// Accept YYYY-MM-DD
		t, err := time.Parse("2006-01-02", val)
		if err != nil {
			return nil, fmt.Errorf("invalid date (expected YYYY-MM-DD): %s", val)
		}
		return pgtype.Date{Time: t, Valid: true}, nil
	case "json":
		if !json.Valid([]byte(val)) {
			return nil, fmt.Errorf("invalid JSON: %s", val)
		}
		return val, nil
	default:
		return val, nil
	}
}

// lookupMatchKey checks if a record with the given match keys exists.
// Returns the existing record's ID if found, nil if not.
func lookupMatchKey(ctx context.Context, pool *pgxpool.Pool, cfg *EntityConfig, row *importRow) (*pgtype.UUID, error) {
	if len(cfg.MatchKeys) == 0 {
		return nil, nil
	}

	var conditions []string
	var args []interface{}
	for i, key := range cfg.MatchKeys {
		val, ok := row.values[key]
		if !ok || val == nil {
			return nil, nil // can't match if a key is missing
		}
		conditions = append(conditions, fmt.Sprintf("%s = $%d", key, i+1))
		args = append(args, val)
	}

	query := fmt.Sprintf(
		"SELECT id FROM %s WHERE %s AND deleted_at IS NULL",
		cfg.Table,
		strings.Join(conditions, " AND "),
	)

	var id pgtype.UUID
	err := pool.QueryRow(ctx, query, args...).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &id, nil
}

// executeInsert inserts a new record.
func executeInsert(ctx context.Context, tx pgx.Tx, cfg *EntityConfig, row *importRow) error {
	var cols []string
	var placeholders []string
	var args []interface{}
	i := 1

	for _, col := range cfg.Columns {
		val, ok := row.values[col.DBColumn]
		if !ok {
			continue
		}
		cols = append(cols, col.DBColumn)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		args = append(args, val)
		i++
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		cfg.Table,
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
	)

	_, err := tx.Exec(ctx, query, args...)
	return err
}

// executeUpdate updates an existing record by ID.
func executeUpdate(ctx context.Context, tx pgx.Tx, cfg *EntityConfig, row *importRow) error {
	var sets []string
	var args []interface{}
	i := 1

	for _, col := range cfg.Columns {
		val, ok := row.values[col.DBColumn]
		if !ok {
			continue
		}
		sets = append(sets, fmt.Sprintf("%s = $%d", col.DBColumn, i))
		args = append(args, val)
		i++
	}

	args = append(args, *row.existsID)
	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE id = $%d AND deleted_at IS NULL",
		cfg.Table,
		strings.Join(sets, ", "),
		i,
	)

	_, err := tx.Exec(ctx, query, args...)
	return err
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/importer/ -v`
Expected: all TestParseHeaders tests pass

- [ ] **Step 5: Add more unit tests for parseValue**

Add to `engine_test.go`:

```go
func TestParseValue_Text(t *testing.T) {
	col := ColumnConfig{Type: "text"}
	val, err := parseValue(context.Background(), "hello", col, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "hello" {
		t.Fatalf("expected 'hello', got %v", val)
	}
}

func TestParseValue_Int(t *testing.T) {
	col := ColumnConfig{Type: "int"}
	val, err := parseValue(context.Background(), "42", col, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 42 {
		t.Fatalf("expected 42, got %v", val)
	}
}

func TestParseValue_IntInvalid(t *testing.T) {
	col := ColumnConfig{Type: "int"}
	_, err := parseValue(context.Background(), "abc", col, nil)
	if err == nil {
		t.Fatal("expected error for invalid int")
	}
}

func TestParseValue_Bool(t *testing.T) {
	col := ColumnConfig{Type: "bool"}
	for _, tc := range []struct {
		input string
		want  bool
	}{
		{"true", true}, {"1", true}, {"yes", true},
		{"false", false}, {"0", false}, {"no", false},
	} {
		val, err := parseValue(context.Background(), tc.input, col, nil)
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", tc.input, err)
		}
		if val != tc.want {
			t.Fatalf("for %q: expected %v, got %v", tc.input, tc.want, val)
		}
	}
}

func TestParseValue_Date(t *testing.T) {
	col := ColumnConfig{Type: "date"}
	val, err := parseValue(context.Background(), "2026-03-29", col, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	d, ok := val.(pgtype.Date)
	if !ok || !d.Valid {
		t.Fatalf("expected valid pgtype.Date, got %v", val)
	}
}

func TestParseValue_DateInvalid(t *testing.T) {
	col := ColumnConfig{Type: "date"}
	_, err := parseValue(context.Background(), "29.03.2026", col, nil)
	if err == nil {
		t.Fatal("expected error for invalid date format")
	}
}

func TestParseValue_JSON(t *testing.T) {
	col := ColumnConfig{Type: "json"}
	val, err := parseValue(context.Background(), `{"key":"value"}`, col, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != `{"key":"value"}` {
		t.Fatalf("expected JSON string, got %v", val)
	}
}

func TestParseValue_JSONInvalid(t *testing.T) {
	col := ColumnConfig{Type: "json"}
	_, err := parseValue(context.Background(), `{broken`, col, nil)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
```

- [ ] **Step 6: Run all tests**

Run: `go test ./internal/importer/ -v`
Expected: all tests pass

- [ ] **Step 7: Commit**

```bash
git add internal/importer/engine.go internal/importer/engine_test.go
git commit -m "feat: implement import engine with CSV parsing, validation, and upsert"
```

---

## Task 7: Exporter — CSV export and template generation

**Files:**
- Create: `internal/importer/exporter.go`
- Create: `internal/importer/exporter_test.go`

- [ ] **Step 1: Write a test for template generation**

Create `internal/importer/exporter_test.go`:

```go
package importer

import (
	"bytes"
	"testing"
)

func TestGenerateTemplate(t *testing.T) {
	cfg := customersConfig()
	exp := &Exporter{registry: NewRegistry()}
	var buf bytes.Buffer
	err := exp.WriteTemplate(&buf, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should contain BOM + headers
	output := buf.Bytes()
	// Check BOM
	if len(output) < 3 || output[0] != 0xEF || output[1] != 0xBB || output[2] != 0xBF {
		t.Fatal("expected UTF-8 BOM at start of template")
	}
	// Check headers
	content := string(output[3:])
	if content != "name,contact_email,notes\r\n" {
		t.Fatalf("unexpected template content: %q", content)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/importer/ -run TestGenerateTemplate -v`
Expected: FAIL (Exporter not defined)

- [ ] **Step 3: Implement the exporter**

Create `internal/importer/exporter.go`:

```go
package importer

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BOM is the UTF-8 Byte Order Mark, prepended to exports for Excel compatibility.
var bom = []byte{0xEF, 0xBB, 0xBF}

// Exporter generates CSV exports and templates for entities.
type Exporter struct {
	pool     *pgxpool.Pool
	registry *Registry
}

// NewExporter creates a new exporter.
func NewExporter(pool *pgxpool.Pool, registry *Registry) *Exporter {
	return &Exporter{pool: pool, registry: registry}
}

// WriteTemplate writes an empty CSV with just the header row (BOM + headers).
func (e *Exporter) WriteTemplate(w io.Writer, cfg *EntityConfig) error {
	if _, err := w.Write(bom); err != nil {
		return err
	}
	writer := csv.NewWriter(w)
	var headers []string
	for _, col := range cfg.Columns {
		headers = append(headers, col.Header)
	}
	if err := writer.Write(headers); err != nil {
		return err
	}
	writer.Flush()
	return writer.Error()
}

// Export writes all active records as CSV.
func (e *Exporter) Export(ctx context.Context, w io.Writer, entityName string) error {
	cfg, err := e.registry.Get(entityName)
	if err != nil {
		return err
	}

	// Build SELECT columns
	var selectCols []string
	for _, col := range cfg.Columns {
		selectCols = append(selectCols, col.DBColumn)
	}

	query := fmt.Sprintf(
		"SELECT %s FROM %s WHERE deleted_at IS NULL ORDER BY created_at",
		strings.Join(selectCols, ", "),
		cfg.Table,
	)

	rows, err := e.pool.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("querying %s: %w", cfg.Table, err)
	}
	defer rows.Close()

	// Build reverse resolver for FK columns (UUID → name)
	reverseCache := make(map[string]map[string]string) // "table" → "uuid" → "name"

	// Write BOM + headers
	if _, err := w.Write(bom); err != nil {
		return err
	}
	writer := csv.NewWriter(w)
	var headers []string
	for _, col := range cfg.Columns {
		headers = append(headers, col.Header)
	}
	if err := writer.Write(headers); err != nil {
		return err
	}

	for rows.Next() {
		// Scan all values as interface{}
		values, err := rows.Values()
		if err != nil {
			return fmt.Errorf("scanning row: %w", err)
		}

		record := make([]string, len(cfg.Columns))
		for i, col := range cfg.Columns {
			if i >= len(values) || values[i] == nil {
				record[i] = ""
				continue
			}
			record[i], err = formatValue(ctx, e.pool, values[i], col, reverseCache)
			if err != nil {
				return fmt.Errorf("formatting %s: %w", col.Header, err)
			}
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	writer.Flush()
	return writer.Error()
}

// formatValue converts a DB value back to a CSV-friendly string.
// For name-strategy FKs, it reverse-resolves UUID → name.
func formatValue(ctx context.Context, pool *pgxpool.Pool, val interface{}, col ColumnConfig, cache map[string]map[string]string) (string, error) {
	// FK reverse resolution
	if col.FK != nil && col.FK.Strategy == "name" {
		return reverseResolve(ctx, pool, val, col.FK, cache)
	}

	switch v := val.(type) {
	case string:
		return v, nil
	case int32:
		return fmt.Sprintf("%d", v), nil
	case int64:
		return fmt.Sprintf("%d", v), nil
	case bool:
		if v {
			return "true", nil
		}
		return "false", nil
	case time.Time:
		return v.Format("2006-01-02"), nil
	case [16]byte:
		// UUID
		id := pgtype.UUID{Bytes: v, Valid: true}
		return fmt.Sprintf("%x-%x-%x-%x-%x", id.Bytes[0:4], id.Bytes[4:6], id.Bytes[6:8], id.Bytes[8:10], id.Bytes[10:16]), nil
	case map[string]interface{}:
		// JSONB — pgx returns as map
		b, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		return string(b), nil
	default:
		return fmt.Sprintf("%v", v), nil
	}
}

// reverseResolve converts a UUID back to a human-readable name.
func reverseResolve(ctx context.Context, pool *pgxpool.Pool, val interface{}, fk *FKConfig, cache map[string]map[string]string) (string, error) {
	// Get UUID as string
	var uuidStr string
	switch v := val.(type) {
	case [16]byte:
		id := pgtype.UUID{Bytes: v, Valid: true}
		uuidStr = fmt.Sprintf("%x-%x-%x-%x-%x", id.Bytes[0:4], id.Bytes[4:6], id.Bytes[6:8], id.Bytes[8:10], id.Bytes[10:16])
	default:
		return fmt.Sprintf("%v", val), nil
	}

	// Check cache
	if tableCache, ok := cache[fk.Table]; ok {
		if name, ok := tableCache[uuidStr]; ok {
			return name, nil
		}
	}

	// Lookup
	var name string
	err := pool.QueryRow(ctx,
		fmt.Sprintf("SELECT %s FROM %s WHERE id = $1 AND deleted_at IS NULL", fk.LookupCol, fk.Table),
		val,
	).Scan(&name)
	if err != nil {
		return uuidStr, nil // fall back to UUID if lookup fails
	}

	// Cache
	if _, ok := cache[fk.Table]; !ok {
		cache[fk.Table] = make(map[string]string)
	}
	cache[fk.Table][uuidStr] = name
	return name, nil
}
```

Note: The `json` import is needed — add `"encoding/json"` to the import block.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/importer/ -v`
Expected: all tests pass

- [ ] **Step 5: Commit**

```bash
git add internal/importer/exporter.go internal/importer/exporter_test.go
git commit -m "feat: implement CSV exporter with reverse FK resolution and templates"
```

---

## Task 8: Import API handler

**Files:**
- Create: `internal/handler/import.go`
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Create the import handler**

Create `internal/handler/import.go`:

```go
package handler

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/xan-com/xan-argus/internal/importer"
)

// ImportHandler handles CSV import, export, and template endpoints.
type ImportHandler struct {
	engine   *importer.Engine
	exporter *importer.Exporter
	registry *importer.Registry
}

// NewImportHandler creates a new import handler.
func NewImportHandler(engine *importer.Engine, exporter *importer.Exporter, registry *importer.Registry) *ImportHandler {
	return &ImportHandler{engine: engine, exporter: exporter, registry: registry}
}

// RegisterRoutes registers all import/export API routes.
func (h *ImportHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/import/{entity}", h.importCSV)
	mux.HandleFunc("GET /api/v1/import/{entity}/template", h.template)
	mux.HandleFunc("GET /api/v1/export/{entity}", h.export)
}

func (h *ImportHandler) importCSV(w http.ResponseWriter, r *http.Request) {
	entityName := r.PathValue("entity")

	// Validate entity exists
	if _, err := h.registry.Get(entityName); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("unknown entity: %s", entityName))
		return
	}

	// Parse multipart form (max 10MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing 'file' field")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read file")
		return
	}

	result, err := h.engine.Import(r.Context(), entityName, data)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("import failed: %v", err))
		return
	}

	if len(result.Errors) > 0 {
		writeJSON(w, http.StatusBadRequest, result)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *ImportHandler) template(w http.ResponseWriter, r *http.Request) {
	entityName := r.PathValue("entity")

	cfg, err := h.registry.Get(entityName)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("unknown entity: %s", entityName))
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s_template.csv"`, entityName))

	if err := h.exporter.WriteTemplate(w, cfg); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate template")
		return
	}
}

func (h *ImportHandler) export(w http.ResponseWriter, r *http.Request) {
	entityName := r.PathValue("entity")

	if _, err := h.registry.Get(entityName); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("unknown entity: %s", entityName))
		return
	}

	filename := fmt.Sprintf("%s_%s.csv", entityName, time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	if err := h.exporter.Export(r.Context(), w, entityName); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("export failed: %v", err))
		return
	}
}
```

- [ ] **Step 2: Wire up in main.go**

In `cmd/server/main.go`, after the existing repository declarations, add:

```go
	// Import/Export
	importRegistry := importer.NewRegistry()
	importEngine := importer.NewEngine(pool, importRegistry)
	importExporter := importer.NewExporter(pool, importRegistry)
```

Add the import to the imports block:
```go
	"github.com/xan-com/xan-argus/internal/importer"
```

After the existing handler registrations, add:
```go
	handler.NewImportHandler(importEngine, importExporter, importRegistry).RegisterRoutes(mux)
```

- [ ] **Step 3: Verify it compiles**

Run: `go vet ./...`
Expected: passes

- [ ] **Step 4: Commit**

```bash
git add internal/handler/import.go cmd/server/main.go
git commit -m "feat: add import/export API handler and wire up in main.go"
```

---

## Task 9: Import page template + page handler route

**Files:**
- Create: `web/templates/import/page.html`
- Modify: `internal/handler/page.go` — add `importPage` method and route
- Modify: `internal/handler/template.go` — register `import/page.html`
- Modify: `web/templates/layout.html` — add "Import" nav link

- [ ] **Step 1: Create the import page template**

Create `web/templates/import/page.html`:

```html
{{define "content"}}
<header class="page-header">
    <h1>Data Import</h1>
</header>

<div class="card">
    <section class="section">
        <h2>Import CSV</h2>

        <div class="form-group">
            <label for="entity-select">Entity</label>
            <select id="entity-select">
                {{range .Entities}}
                <option value="{{.}}">{{.}}</option>
                {{end}}
            </select>
        </div>

        <div class="flex gap-1 mb-1">
            <a id="template-link" href="/api/v1/import/customers/template" class="btn btn-sm">Download Template</a>
        </div>

        <form id="import-form" enctype="multipart/form-data">
            <div class="form-group">
                <label for="file">CSV File</label>
                <input type="file" id="file" name="file" accept=".csv" required>
            </div>

            <div id="import-message" aria-live="polite"></div>

            <button type="submit" class="btn btn-primary">Import</button>
        </form>
    </section>

    <section class="section">
        <h2>Export Data</h2>
        <p class="text-muted">Download all active records as CSV:</p>
        <div class="flex gap-1" style="flex-wrap:wrap">
            {{range .Entities}}
            <a href="/api/v1/export/{{.}}" class="btn btn-sm">{{.}}</a>
            {{end}}
        </div>
    </section>

    <section class="section">
        <h2>Recommended Import Order</h2>
        <p class="text-muted">Import entities in this order to satisfy dependencies:</p>
        <ol>
            {{range .Entities}}
            <li>{{.}}</li>
            {{end}}
        </ol>
    </section>
</div>

<script>
    // Update template download link when entity changes
    document.getElementById('entity-select').addEventListener('change', function() {
        document.getElementById('template-link').href = '/api/v1/import/' + this.value + '/template';
    });

    // Handle form submit — upload CSV via fetch
    document.getElementById('import-form').addEventListener('submit', function(evt) {
        evt.preventDefault();
        var entity = document.getElementById('entity-select').value;
        var fileInput = document.getElementById('file');
        if (!fileInput.files.length) return;

        var formData = new FormData();
        formData.append('file', fileInput.files[0]);

        var msgEl = document.getElementById('import-message');
        msgEl.innerHTML = '<div class="alert">Importing...</div>';

        fetch('/api/v1/import/' + entity, {
            method: 'POST',
            body: formData
        })
        .then(function(resp) { return resp.json().then(function(data) { return {ok: resp.ok, data: data}; }); })
        .then(function(result) {
            if (result.ok) {
                msgEl.innerHTML = '<div class="alert alert-success" role="alert">Import successful: ' +
                    result.data.created + ' created, ' + result.data.updated + ' updated (' + result.data.total + ' total)</div>';
            } else if (result.data.errors && result.data.errors.length > 0) {
                var html = '<div class="alert alert-error" role="alert">Import failed with ' + result.data.errors.length + ' error(s):</div>';
                html += '<table><thead><tr><th>Row</th><th>Column</th><th>Error</th></tr></thead><tbody>';
                result.data.errors.forEach(function(e) {
                    html += '<tr><td>' + e.row + '</td><td>' + (e.column || '') + '</td><td>' + e.message + '</td></tr>';
                });
                html += '</tbody></table>';
                msgEl.innerHTML = html;
            } else {
                msgEl.innerHTML = '<div class="alert alert-error" role="alert">' + (result.data.error || 'Import failed') + '</div>';
            }
        })
        .catch(function(err) {
            msgEl.innerHTML = '<div class="alert alert-error" role="alert">Network error: ' + err.message + '</div>';
        });
    });
</script>
{{end}}
```

- [ ] **Step 2: Add import page route to page.go**

In `internal/handler/page.go`, add to `RegisterRoutes`:

```go
	mux.HandleFunc("GET /import", h.importPage)
```

The `PageHandler` struct needs a new `importRegistry` field. Add it to the struct definition in `page.go`:

```go
	importRegistry *importer.Registry
```

Update `NewPageHandler` to accept the registry as its last parameter and assign it. The existing constructor takes `(tmpl, customerRepo, userRepo, serviceRepo, userAssignmentRepo, assetRepo, licenseRepo, customerServiceRepo, hardwareCategoryRepo)` — add `importRegistry *importer.Registry` after `hardwareCategoryRepo` and assign `importRegistry: importRegistry` in the struct literal. Also add the import `"github.com/xan-com/xan-argus/internal/importer"` to the file's import block.

Add the handler method:

```go
func (h *PageHandler) importPage(w http.ResponseWriter, r *http.Request) {
	h.tmpl.RenderPage(w, "import/page", map[string]any{
		"Title":    "Data Import",
		"Entities": h.importRegistry.OrderedNames(),
	})
}
```

- [ ] **Step 3: Register template in template.go**

In `NewTemplateEngine`, add to `pageFiles`:

```go
	filepath.Join(templateDir, "import", "page.html"),
```

- [ ] **Step 4: Add nav link in layout.html**

In `web/templates/layout.html`, add after the "Categories" link:

```html
            <a href="/import">Import</a>
```

- [ ] **Step 5: Update main.go**

Pass `importRegistry` to `NewPageHandler`. Update the `NewPageHandler` call:

```go
	pageHandler := handler.NewPageHandler(tmpl, customerRepo, userRepo, serviceRepo, userAssignmentRepo, assetRepo, licenseRepo, customerServiceRepo, hardwareCategoryRepo, importRegistry)
```

- [ ] **Step 6: Verify it compiles**

Run: `go vet ./...`
Expected: passes

- [ ] **Step 7: Commit**

```bash
git add web/templates/import/page.html internal/handler/page.go internal/handler/template.go web/templates/layout.html cmd/server/main.go
git commit -m "feat: add import page with entity selection, template download, and CSV upload"
```

---

## Task 10: Export buttons on list pages

Note: Only customers, users, services, and categories have dedicated list pages. Assets, licenses, user-assignments, customer-services, and field-definitions do not have standalone list pages — their export is available via the "Export Data" section on the import page (added in Task 9) and via the API directly (`GET /api/v1/export/{entity}`).

**Files:**
- Modify: `web/templates/customers/list.html`
- Modify: `web/templates/users/list.html`
- Modify: `web/templates/services/list.html`
- Modify: `web/templates/categories/list.html`

- [ ] **Step 1: Add export button to each list page**

On each list page, add an export button next to the existing "New" button in the `page-header`. The pattern is the same for all:

**customers/list.html** — in the `<header class="page-header">`, add:
```html
    <div class="flex gap-1">
        <a href="/api/v1/export/customers" class="btn btn-sm">Export CSV</a>
        <a href="/customers/new" class="btn btn-primary">+ New Customer</a>
    </div>
```
(Replace the standalone `<a href="/customers/new" ...>` with this wrapped version.)

**users/list.html:**
```html
    <div class="flex gap-1">
        <a href="/api/v1/export/users" class="btn btn-sm">Export CSV</a>
        <a href="/users/new" class="btn btn-primary">+ New User</a>
    </div>
```

**services/list.html:**
```html
    <div class="flex gap-1">
        <a href="/api/v1/export/services" class="btn btn-sm">Export CSV</a>
        <a href="/services/new" class="btn btn-primary">+ New Service</a>
    </div>
```

**categories/list.html:**
```html
    <div class="flex gap-1">
        <a href="/api/v1/export/hardware-categories" class="btn btn-sm">Export CSV</a>
        <a href="/categories/new" class="btn btn-primary">+ New Category</a>
    </div>
```

- [ ] **Step 2: Verify templates render correctly**

Run: `go vet ./...`
Expected: passes (template errors are runtime, but Go code should compile)

- [ ] **Step 3: Commit**

```bash
git add web/templates/customers/list.html web/templates/users/list.html web/templates/services/list.html web/templates/categories/list.html
git commit -m "feat: add export CSV buttons to all list pages"
```

---

## Task 11: Integration test — full docker compose verification

**Files:** None (manual verification)

- [ ] **Step 1: Build and run**

Run: `docker compose up --build`
Expected: application starts, migrations run (including 009_import_indexes.sql)

- [ ] **Step 2: Verify import page**

Open `http://localhost:8080/import`
Expected: Import page with entity dropdown, template download link, file upload, dependency order list

- [ ] **Step 3: Test template download**

Click "Download Template" with "customers" selected.
Expected: Downloads `customers_template.csv` with header row: `name,contact_email,notes`

- [ ] **Step 4: Test import**

Create a test CSV `test_customers.csv`:
```csv
name,contact_email,notes
Test Corp,test@example.com,Test notes
Another Inc,another@example.com,
```

Upload via the import page with "customers" selected.
Expected: "Import successful: 2 created, 0 updated (2 total)"

- [ ] **Step 5: Test upsert (re-import with changes)**

Modify `test_customers.csv`:
```csv
name,contact_email,notes
Test Corp,updated@example.com,Updated notes
Another Inc,another@example.com,Now with notes
Third Ltd,third@example.com,Brand new
```

Upload again.
Expected: "Import successful: 1 created, 2 updated (3 total)"

- [ ] **Step 6: Test export**

Go to `/customers` and click "Export CSV".
Expected: Downloads CSV with the 3 customers (plus any seed data), matching the import format.

- [ ] **Step 7: Test error handling**

Create a CSV with errors:
```csv
name,contact_email
,test@example.com
```

Upload.
Expected: Error table showing "Row 1, Column: name, name is required"

- [ ] **Step 8: Test FK resolution**

Create an assets CSV referencing a customer by name:
```csv
name,customer,description
Laptop 1,Test Corp,A test laptop
Laptop 2,Nonexistent Corp,Another laptop
```

Upload with "assets" selected.
Expected: Error: "customer 'Nonexistent Corp' not found" — nothing imported (all-or-nothing).

- [ ] **Step 9: Commit any fixes needed**

If any issues were found during testing, fix and commit.

- [ ] **Step 10: Final commit**

```bash
git add -A
git commit -m "feat: CSV import/export — complete implementation"
```
