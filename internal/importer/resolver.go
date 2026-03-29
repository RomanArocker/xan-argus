package importer

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Resolver struct {
	pool  *pgxpool.Pool
	cache map[string]pgtype.UUID
}

func NewResolver(pool *pgxpool.Pool) *Resolver {
	return &Resolver{
		pool:  pool,
		cache: make(map[string]pgtype.UUID),
	}
}

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
