package importer

import (
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

// columnMapping maps a CSV column index to its EntityConfig column definition.
type columnMapping struct {
	csvIndex int
	config   ColumnConfig
}

// Engine orchestrates CSV import for any registered entity.
type Engine struct {
	pool     *pgxpool.Pool
	registry *Registry
}

// NewEngine creates a new Engine.
func NewEngine(pool *pgxpool.Pool, registry *Registry) *Engine {
	return &Engine{pool: pool, registry: registry}
}

// stripBOM removes the UTF-8 BOM (0xEF, 0xBB, 0xBF) from the start of data if present.
func stripBOM(data []byte) []byte {
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return data[3:]
	}
	return data
}

// parseHeaders reads the first CSV row and maps headers to ColumnConfig entries.
// Unknown columns are silently ignored. Returns an error if any required column is absent.
func parseHeaders(r io.Reader, cfg *EntityConfig) ([]columnMapping, error) {
	cr := csv.NewReader(r)
	headers, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf("reading CSV headers: %w", err)
	}

	// Build a lookup from header name → ColumnConfig.
	colByHeader := make(map[string]ColumnConfig, len(cfg.Columns))
	for _, col := range cfg.Columns {
		colByHeader[col.Header] = col
	}

	// Map each CSV column index to its config.
	mappings := make([]columnMapping, 0, len(headers))
	found := make(map[string]bool)
	for i, h := range headers {
		h = strings.TrimSpace(h)
		if col, ok := colByHeader[h]; ok {
			mappings = append(mappings, columnMapping{csvIndex: i, config: col})
			found[h] = true
		}
	}

	// Validate that all required columns are present.
	for _, col := range cfg.Columns {
		if col.Required && !found[col.Header] {
			return nil, fmt.Errorf("missing required column: %s", col.Header)
		}
	}

	return mappings, nil
}

// parseValue converts a raw CSV string to the appropriate Go type for the column.
func parseValue(ctx context.Context, val string, col ColumnConfig, resolver *Resolver) (interface{}, error) {
	// FK resolution takes priority (the FK config also sets Type to "uuid").
	if col.FK != nil {
		if val == "" {
			if col.Required {
				return nil, fmt.Errorf("required FK value is empty")
			}
			return nil, nil //nolint:nilnil
		}
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
			return nil, fmt.Errorf("invalid UUID %q", val)
		}
		return id, nil

	case "int":
		n, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid integer %q", val)
		}
		return n, nil

	case "bool":
		switch strings.ToLower(strings.TrimSpace(val)) {
		case "true", "1", "yes":
			return true, nil
		case "false", "0", "no":
			return false, nil
		default:
			return nil, fmt.Errorf("invalid boolean %q (use true/false/1/0/yes/no)", val)
		}

	case "date":
		if val == "" {
			var d pgtype.Date
			return d, nil // zero/null date
		}
		t, err := time.Parse("2006-01-02", val)
		if err != nil {
			return nil, fmt.Errorf("invalid date %q (expected YYYY-MM-DD)", val)
		}
		var d pgtype.Date
		if err := d.Scan(t); err != nil {
			return nil, fmt.Errorf("converting date %q: %w", val, err)
		}
		return d, nil

	case "json":
		if val == "" {
			return "{}", nil
		}
		if !json.Valid([]byte(val)) {
			return nil, fmt.Errorf("invalid JSON %q", val)
		}
		return val, nil

	default:
		return val, nil
	}
}

// parsedRow holds the DB column→value map for one validated CSV row.
type parsedRow map[string]interface{}

// lookupMatchKey queries the table for an existing record matching the match keys.
// Returns nil when no record is found.
func lookupMatchKey(ctx context.Context, pool *pgxpool.Pool, cfg *EntityConfig, row parsedRow) (*pgtype.UUID, error) {
	if len(cfg.MatchKeys) == 0 {
		return nil, nil //nolint:nilnil
	}

	where := make([]string, 0, len(cfg.MatchKeys))
	args := make([]interface{}, 0, len(cfg.MatchKeys)+1)
	for i, mk := range cfg.MatchKeys {
		where = append(where, fmt.Sprintf("%s = $%d", mk, i+1))
		args = append(args, row[mk])
	}
	args = append(args, nil) // placeholder — deleted_at IS NULL uses no param

	query := fmt.Sprintf(
		"SELECT id FROM %s WHERE %s AND deleted_at IS NULL",
		cfg.Table,
		strings.Join(where, " AND "),
	)

	var id pgtype.UUID
	err := pool.QueryRow(ctx, query, args[:len(cfg.MatchKeys)]...).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil //nolint:nilnil
		}
		return nil, fmt.Errorf("looking up match key in %s: %w", cfg.Table, err)
	}
	return &id, nil
}

// executeInsert inserts a new row using the parsed column values.
func executeInsert(ctx context.Context, tx pgx.Tx, cfg *EntityConfig, row parsedRow) error {
	cols := make([]string, 0, len(row))
	params := make([]string, 0, len(row))
	args := make([]interface{}, 0, len(row))
	i := 1
	for _, col := range cfg.Columns {
		val, ok := row[col.DBColumn]
		if !ok {
			continue
		}
		cols = append(cols, col.DBColumn)
		params = append(params, fmt.Sprintf("$%d", i))
		args = append(args, val)
		i++
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		cfg.Table,
		strings.Join(cols, ", "),
		strings.Join(params, ", "),
	)
	_, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("inserting into %s: %w", cfg.Table, err)
	}
	return nil
}

// executeUpdate updates an existing row identified by id.
func executeUpdate(ctx context.Context, tx pgx.Tx, cfg *EntityConfig, id pgtype.UUID, row parsedRow) error {
	sets := make([]string, 0, len(row))
	args := make([]interface{}, 0, len(row)+1)
	i := 1
	for _, col := range cfg.Columns {
		val, ok := row[col.DBColumn]
		if !ok {
			continue
		}
		sets = append(sets, fmt.Sprintf("%s = $%d", col.DBColumn, i))
		args = append(args, val)
		i++
	}
	args = append(args, id)

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE id = $%d AND deleted_at IS NULL",
		cfg.Table,
		strings.Join(sets, ", "),
		i,
	)
	result, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("updating %s: %w", cfg.Table, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("record not found or already deleted in %s", cfg.Table)
	}
	return nil
}

// Import parses csvData, validates, resolves FKs, and upserts all rows in one transaction.
func (e *Engine) Import(ctx context.Context, entityName string, csvData []byte) (*ImportResult, error) {
	cfg, err := e.registry.Get(entityName)
	if err != nil {
		return nil, err
	}

	csvData = stripBOM(csvData)
	resolver := NewResolver(e.pool)

	// Phase 1: parse headers.
	mappings, err := parseHeaders(strings.NewReader(string(csvData)), cfg)
	if err != nil {
		return nil, err
	}

	// Re-parse full CSV (headers already consumed above — re-read from string).
	cr := csv.NewReader(strings.NewReader(string(csvData)))
	// Skip header row.
	if _, err := cr.Read(); err != nil {
		return nil, fmt.Errorf("re-reading CSV header: %w", err)
	}

	type rowOp struct {
		row      parsedRow
		existsID *pgtype.UUID // nil → INSERT, non-nil → UPDATE
	}

	result := &ImportResult{Errors: []ImportError{}}
	var ops []rowOp
	rowNum := 0

	// Phase 2: validate, parse, resolve FKs.
	for {
		record, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading CSV row: %w", err)
		}
		rowNum++
		result.Total++

		row := make(parsedRow)
		rowErrors := false

		for _, m := range mappings {
			if m.csvIndex >= len(record) {
				if m.config.Required {
					result.Errors = append(result.Errors, ImportError{
						Row:     rowNum,
						Column:  m.config.Header,
						Message: "required field missing",
					})
					rowErrors = true
				}
				continue
			}
			val := strings.TrimSpace(record[m.csvIndex])

			if val == "" && m.config.Required && m.config.FK == nil {
				result.Errors = append(result.Errors, ImportError{
					Row:     rowNum,
					Column:  m.config.Header,
					Message: "required field is empty",
				})
				rowErrors = true
				continue
			}

			parsed, err := parseValue(ctx, val, m.config, resolver)
			if err != nil {
				result.Errors = append(result.Errors, ImportError{
					Row:     rowNum,
					Column:  m.config.Header,
					Message: err.Error(),
				})
				rowErrors = true
				continue
			}
			if parsed != nil {
				row[m.config.DBColumn] = parsed
			}
		}

		if rowErrors {
			ops = append(ops, rowOp{}) // placeholder to keep row count aligned
			continue
		}

		// Phase 3: lookup match key.
		existsID, err := lookupMatchKey(ctx, e.pool, cfg, row)
		if err != nil {
			result.Errors = append(result.Errors, ImportError{
				Row:     rowNum,
				Column:  "",
				Message: err.Error(),
			})
			ops = append(ops, rowOp{})
			continue
		}

		ops = append(ops, rowOp{row: row, existsID: existsID})
	}

	// If any validation errors, stop here.
	if len(result.Errors) > 0 {
		return result, nil
	}

	// Phase 4: execute all in one transaction.
	tx, err := e.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	for _, op := range ops {
		if op.row == nil {
			continue
		}
		if op.existsID == nil {
			if err := executeInsert(ctx, tx, cfg, op.row); err != nil {
				return nil, err
			}
			result.Created++
		} else {
			if err := executeUpdate(ctx, tx, cfg, *op.existsID, op.row); err != nil {
				return nil, err
			}
			result.Updated++
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing transaction: %w", err)
	}

	return result, nil
}
