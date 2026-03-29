package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xan-com/xan-argus/internal/model"
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
		`INSERT INTO services (name, description)
		 VALUES ($1, $2)
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
		`SELECT id, name, description, created_at, updated_at
		 FROM services WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&s.ID, &s.Name, &s.Description, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return s, fmt.Errorf("getting service: %w", err)
	}
	return s, nil
}

func (r *ServiceRepository) List(ctx context.Context, params model.ListParams) ([]model.Service, error) {
	params.Normalize()
	query := `SELECT id, name, description, created_at, updated_at FROM services WHERE deleted_at IS NULL`
	args := []any{}
	if params.Search != "" {
		query += ` AND name ILIKE $1`
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
		`UPDATE services SET
			name = COALESCE($2, name),
			description = COALESCE($3, description)
		 WHERE id = $1 AND deleted_at IS NULL
		 RETURNING id, name, description, created_at, updated_at`,
		id, input.Name, input.Description,
	).Scan(&s.ID, &s.Name, &s.Description, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return s, fmt.Errorf("updating service: %w", err)
	}
	return s, nil
}

func (r *ServiceRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	result, err := r.pool.Exec(ctx, `UPDATE services SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("deleting service: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("service not found")
	}
	return nil
}
