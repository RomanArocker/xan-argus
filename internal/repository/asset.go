package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xan-com/xan-argus/internal/model"
)

type AssetRepository struct {
	pool *pgxpool.Pool
}

func NewAssetRepository(pool *pgxpool.Pool) *AssetRepository {
	return &AssetRepository{pool: pool}
}

func (r *AssetRepository) Create(ctx context.Context, input model.CreateAssetInput) (model.Asset, error) {
	if input.Metadata == nil {
		input.Metadata = json.RawMessage("{}")
	}
	if input.FieldValues == nil {
		input.FieldValues = json.RawMessage("{}")
	}
	var a model.Asset
	err := r.pool.QueryRow(ctx,
		`INSERT INTO assets (customer_id, category_id, name, description, metadata, field_values)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, customer_id, category_id, name, description, metadata, field_values, created_at, updated_at`,
		input.CustomerID, input.CategoryID, input.Name, input.Description, input.Metadata, input.FieldValues,
	).Scan(&a.ID, &a.CustomerID, &a.CategoryID, &a.Name, &a.Description, &a.Metadata, &a.FieldValues, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return a, fmt.Errorf("creating asset: %w", err)
	}
	return a, nil
}

func (r *AssetRepository) GetByID(ctx context.Context, id pgtype.UUID) (model.Asset, error) {
	var a model.Asset
	err := r.pool.QueryRow(ctx,
		`SELECT id, customer_id, category_id, name, description, metadata, field_values, created_at, updated_at
		 FROM assets WHERE id = $1`, id,
	).Scan(&a.ID, &a.CustomerID, &a.CategoryID, &a.Name, &a.Description, &a.Metadata, &a.FieldValues, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return a, fmt.Errorf("getting asset: %w", err)
	}
	return a, nil
}

func (r *AssetRepository) ListByCustomer(ctx context.Context, customerID pgtype.UUID, params model.ListParams) ([]model.Asset, error) {
	params.Normalize()
	query := `SELECT id, customer_id, category_id, name, description, metadata, field_values, created_at, updated_at
	          FROM assets WHERE customer_id = $1`
	args := []any{customerID}
	argN := 2
	if params.Filter != "" {
		query += fmt.Sprintf(` AND category_id = $%d`, argN)
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
	if input.CategoryID.Valid && input.FieldValues == nil {
		input.FieldValues = json.RawMessage("{}")
	}
	var a model.Asset
	err := r.pool.QueryRow(ctx,
		`UPDATE assets SET
			category_id = COALESCE($2, category_id),
			name        = COALESCE($3, name),
			description = COALESCE($4, description),
			metadata    = COALESCE($5, metadata),
			field_values = COALESCE($6, field_values)
		 WHERE id = $1
		 RETURNING id, customer_id, category_id, name, description, metadata, field_values, created_at, updated_at`,
		id, input.CategoryID, input.Name, input.Description, input.Metadata, input.FieldValues,
	).Scan(&a.ID, &a.CustomerID, &a.CategoryID, &a.Name, &a.Description, &a.Metadata, &a.FieldValues, &a.CreatedAt, &a.UpdatedAt)
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
		return pgx.ErrNoRows
	}
	return nil
}
