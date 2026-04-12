package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xan-com/xan-argus/internal/model"
)

type BOMRepository struct {
	pool *pgxpool.Pool
}

func NewBOMRepository(pool *pgxpool.Pool) *BOMRepository {
	return &BOMRepository{pool: pool}
}

const bomSelectCols = `
    abi.id, abi.asset_id, abi.name, abi.quantity, abi.unit_id, u.name,
    abi.unit_price, abi.currency, abi.notes, abi.sort_order, abi.created_at, abi.updated_at`

func (r *BOMRepository) ListByAsset(ctx context.Context, assetID pgtype.UUID) ([]model.BOMItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT`+bomSelectCols+`
		FROM asset_bom_items abi
		JOIN units u ON u.id = abi.unit_id
		WHERE abi.asset_id = $1 AND abi.deleted_at IS NULL
		ORDER BY abi.sort_order ASC, abi.created_at ASC`, assetID)
	if err != nil {
		return nil, fmt.Errorf("listing bom items: %w", err)
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToStructByPos[model.BOMItem])
}

func (r *BOMRepository) CountByAsset(ctx context.Context, assetID pgtype.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM asset_bom_items WHERE asset_id = $1 AND deleted_at IS NULL`,
		assetID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting bom items: %w", err)
	}
	return count, nil
}

func (r *BOMRepository) TotalsByAsset(ctx context.Context, assetID pgtype.UUID) ([]model.BOMTotals, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT currency, SUM(quantity * unit_price)
		FROM asset_bom_items
		WHERE asset_id = $1 AND deleted_at IS NULL
		GROUP BY currency
		ORDER BY currency`, assetID)
	if err != nil {
		return nil, fmt.Errorf("getting bom totals: %w", err)
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToStructByPos[model.BOMTotals])
}

func (r *BOMRepository) Create(ctx context.Context, assetID pgtype.UUID, input model.CreateBOMItemInput) (model.BOMItem, error) {
	var item model.BOMItem
	err := r.pool.QueryRow(ctx, `
		WITH inserted AS (
			INSERT INTO asset_bom_items (asset_id, name, quantity, unit_id, unit_price, currency, notes, sort_order)
			VALUES ($1, $2, $3, $4, $5, $6, $7,
				COALESCE($8, (SELECT COALESCE(MAX(sort_order), -1) + 1
				              FROM asset_bom_items WHERE asset_id = $1 AND deleted_at IS NULL)))
			RETURNING *
		)
		SELECT`+bomSelectCols+`
		FROM inserted abi
		JOIN units u ON u.id = abi.unit_id`,
		assetID, input.Name, input.Quantity, input.UnitID, input.UnitPrice,
		input.Currency, input.Notes, input.SortOrder,
	).Scan(
		&item.ID, &item.AssetID, &item.Name, &item.Quantity, &item.UnitID, &item.UnitName,
		&item.UnitPrice, &item.Currency, &item.Notes, &item.SortOrder,
		&item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return item, fmt.Errorf("creating bom item: %w", err)
	}
	return item, nil
}

func (r *BOMRepository) GetByID(ctx context.Context, id pgtype.UUID) (model.BOMItem, error) {
	var item model.BOMItem
	err := r.pool.QueryRow(ctx, `
		SELECT`+bomSelectCols+`
		FROM asset_bom_items abi
		JOIN units u ON u.id = abi.unit_id
		WHERE abi.id = $1 AND abi.deleted_at IS NULL`, id,
	).Scan(
		&item.ID, &item.AssetID, &item.Name, &item.Quantity, &item.UnitID, &item.UnitName,
		&item.UnitPrice, &item.Currency, &item.Notes, &item.SortOrder,
		&item.CreatedAt, &item.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return item, pgx.ErrNoRows
	}
	if err != nil {
		return item, fmt.Errorf("getting bom item: %w", err)
	}
	return item, nil
}

func (r *BOMRepository) Update(ctx context.Context, id pgtype.UUID, input model.UpdateBOMItemInput) (model.BOMItem, error) {
	var item model.BOMItem
	err := r.pool.QueryRow(ctx, `
		WITH updated AS (
			UPDATE asset_bom_items SET
				name       = COALESCE($2, name),
				quantity   = COALESCE($3, quantity),
				unit_id    = COALESCE($4, unit_id),
				unit_price = COALESCE($5, unit_price),
				currency   = COALESCE($6, currency),
				notes      = $7,
				sort_order = COALESCE($8, sort_order)
			WHERE id = $1 AND deleted_at IS NULL
			RETURNING *
		)
		SELECT`+bomSelectCols+`
		FROM updated abi
		JOIN units u ON u.id = abi.unit_id`,
		id, input.Name, input.Quantity, input.UnitID, input.UnitPrice,
		input.Currency, input.Notes, input.SortOrder,
	).Scan(
		&item.ID, &item.AssetID, &item.Name, &item.Quantity, &item.UnitID, &item.UnitName,
		&item.UnitPrice, &item.Currency, &item.Notes, &item.SortOrder,
		&item.CreatedAt, &item.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return item, pgx.ErrNoRows
	}
	if err != nil {
		return item, fmt.Errorf("updating bom item: %w", err)
	}
	return item, nil
}

func (r *BOMRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	result, err := r.pool.Exec(ctx,
		`UPDATE asset_bom_items SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("deleting bom item: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("bom item not found")
	}
	return nil
}

// SwapSortOrder swaps the sort_order of id with its neighbor in the given direction
// ("up" = lower sort_order, "down" = higher sort_order). No-op at boundary.
func (r *BOMRepository) SwapSortOrder(ctx context.Context, assetID, id pgtype.UUID, direction string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint

	var currentSort int
	err = tx.QueryRow(ctx,
		`SELECT sort_order FROM asset_bom_items WHERE id = $1 AND asset_id = $2 AND deleted_at IS NULL`,
		id, assetID,
	).Scan(&currentSort)
	if err == pgx.ErrNoRows {
		return fmt.Errorf("bom item not found")
	}
	if err != nil {
		return fmt.Errorf("getting bom item sort order: %w", err)
	}

	var neighborQ string
	if direction == "up" {
		neighborQ = `SELECT id, sort_order FROM asset_bom_items
			WHERE asset_id = $1 AND deleted_at IS NULL AND sort_order < $2
			ORDER BY sort_order DESC LIMIT 1`
	} else {
		neighborQ = `SELECT id, sort_order FROM asset_bom_items
			WHERE asset_id = $1 AND deleted_at IS NULL AND sort_order > $2
			ORDER BY sort_order ASC LIMIT 1`
	}

	var neighborID pgtype.UUID
	var neighborSort int
	err = tx.QueryRow(ctx, neighborQ, assetID, currentSort).Scan(&neighborID, &neighborSort)
	if err == pgx.ErrNoRows {
		return nil // Already at boundary — no-op
	}
	if err != nil {
		return fmt.Errorf("finding neighbor: %w", err)
	}

	if _, err = tx.Exec(ctx,
		`UPDATE asset_bom_items SET sort_order = $1 WHERE id = $2`, neighborSort, id,
	); err != nil {
		return fmt.Errorf("updating sort order: %w", err)
	}
	if _, err = tx.Exec(ctx,
		`UPDATE asset_bom_items SET sort_order = $1 WHERE id = $2`, currentSort, neighborID,
	); err != nil {
		return fmt.Errorf("updating neighbor sort order: %w", err)
	}
	return tx.Commit(ctx)
}

func (r *BOMRepository) ListUnits(ctx context.Context) ([]model.Unit, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name FROM units ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("listing units: %w", err)
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToStructByPos[model.Unit])
}
