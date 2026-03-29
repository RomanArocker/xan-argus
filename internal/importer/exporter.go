package importer

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// Exporter generates CSV exports and empty templates for any entity.
type Exporter struct {
	pool     *pgxpool.Pool
	registry *Registry
}

// NewExporter creates a new Exporter.
func NewExporter(pool *pgxpool.Pool, registry *Registry) *Exporter {
	return &Exporter{pool: pool, registry: registry}
}

// WriteTemplate writes a UTF-8 BOM + CSV header row only (no data rows).
func (e *Exporter) WriteTemplate(w io.Writer, cfg *EntityConfig) error {
	if _, err := w.Write(utf8BOM); err != nil {
		return fmt.Errorf("writing BOM: %w", err)
	}
	cw := csv.NewWriter(w)
	cw.UseCRLF = true
	headers := make([]string, len(cfg.Columns))
	for i, col := range cfg.Columns {
		headers[i] = col.Header
	}
	if err := cw.Write(headers); err != nil {
		return fmt.Errorf("writing headers: %w", err)
	}
	cw.Flush()
	return cw.Error()
}

// Export writes a full CSV export (BOM + header + data rows) for the named entity.
func (e *Exporter) Export(ctx context.Context, w io.Writer, entityName string) error {
	cfg, err := e.registry.Get(entityName)
	if err != nil {
		return err
	}

	// Build SELECT query from column config.
	cols := make([]string, len(cfg.Columns))
	for i, col := range cfg.Columns {
		cols[i] = col.DBColumn
	}
	query := fmt.Sprintf(
		"SELECT %s FROM %s WHERE deleted_at IS NULL ORDER BY created_at",
		strings.Join(cols, ", "),
		cfg.Table,
	)

	rows, err := e.pool.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("querying %s: %w", cfg.Table, err)
	}
	defer rows.Close()

	// Write BOM.
	if _, err := w.Write(utf8BOM); err != nil {
		return fmt.Errorf("writing BOM: %w", err)
	}

	cw := csv.NewWriter(w)
	cw.UseCRLF = true

	// Write header row.
	headers := make([]string, len(cfg.Columns))
	for i, col := range cfg.Columns {
		headers[i] = col.Header
	}
	if err := cw.Write(headers); err != nil {
		return fmt.Errorf("writing headers: %w", err)
	}

	// Reverse-resolve cache: table -> uuid_string -> name.
	cache := make(map[string]map[string]string)

	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return fmt.Errorf("scanning row: %w", err)
		}

		record := make([]string, len(cfg.Columns))
		for i, col := range cfg.Columns {
			s, err := formatValue(ctx, e.pool, vals[i], col, cache)
			if err != nil {
				return fmt.Errorf("formatting column %s: %w", col.Header, err)
			}
			record[i] = s
		}

		if err := cw.Write(record); err != nil {
			return fmt.Errorf("writing row: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating rows: %w", err)
	}

	cw.Flush()
	return cw.Error()
}

// formatValue converts a DB value to a CSV string.
func formatValue(ctx context.Context, pool *pgxpool.Pool, val interface{}, col ColumnConfig, cache map[string]map[string]string) (string, error) {
	if val == nil {
		return "", nil
	}

	// FK with "name" strategy: reverse-resolve UUID -> name.
	if col.FK != nil && col.FK.Strategy == "name" {
		switch v := val.(type) {
		case [16]byte:
			uuidStr := formatUUID(v)
			name, err := reverseResolve(ctx, pool, uuidStr, col.FK, cache)
			if err != nil {
				return uuidStr, nil // fall back to UUID string
			}
			return name, nil
		case string:
			name, err := reverseResolve(ctx, pool, v, col.FK, cache)
			if err != nil {
				return v, nil
			}
			return name, nil
		}
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
		return formatUUID(v), nil
	case map[string]interface{}:
		b, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("marshaling JSONB: %w", err)
		}
		return string(b), nil
	default:
		return fmt.Sprintf("%v", v), nil
	}
}

// reverseResolve looks up a name from a UUID in the given FK table.
func reverseResolve(ctx context.Context, pool *pgxpool.Pool, uuidStr string, fk *FKConfig, cache map[string]map[string]string) (string, error) {
	if cache[fk.Table] == nil {
		cache[fk.Table] = make(map[string]string)
	}
	if name, ok := cache[fk.Table][uuidStr]; ok {
		return name, nil
	}

	query := fmt.Sprintf(
		"SELECT %s FROM %s WHERE id = $1 AND deleted_at IS NULL",
		fk.LookupCol,
		fk.Table,
	)
	var name string
	err := pool.QueryRow(ctx, query, uuidStr).Scan(&name)
	if err != nil {
		return "", fmt.Errorf("reverse resolve %s in %s: %w", uuidStr, fk.Table, err)
	}

	cache[fk.Table][uuidStr] = name
	return name, nil
}

// formatUUID formats a [16]byte UUID as a standard UUID string.
func formatUUID(id [16]byte) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		id[0:4], id[4:6], id[6:8], id[8:10], id[10:16])
}
