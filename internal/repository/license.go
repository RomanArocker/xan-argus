package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xan-com/xan-argus/internal/model"
)

type LicenseRepository struct {
	pool *pgxpool.Pool
}

func NewLicenseRepository(pool *pgxpool.Pool) *LicenseRepository {
	return &LicenseRepository{pool: pool}
}

func (r *LicenseRepository) Create(ctx context.Context, input model.CreateLicenseInput) (model.License, error) {
	var l model.License
	err := r.pool.QueryRow(ctx,
		`INSERT INTO licenses (customer_id, user_assignment_id, product_name, license_key, quantity, valid_from, valid_until)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, customer_id, user_assignment_id, product_name, license_key, quantity, valid_from, valid_until, created_at, updated_at`,
		input.CustomerID, input.UserAssignmentID, input.ProductName, input.LicenseKey,
		input.Quantity, input.ValidFrom, input.ValidUntil,
	).Scan(&l.ID, &l.CustomerID, &l.UserAssignmentID, &l.ProductName, &l.LicenseKey,
		&l.Quantity, &l.ValidFrom, &l.ValidUntil, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		return l, fmt.Errorf("creating license: %w", err)
	}
	return l, nil
}

func (r *LicenseRepository) GetByID(ctx context.Context, id pgtype.UUID) (model.License, error) {
	var l model.License
	err := r.pool.QueryRow(ctx,
		`SELECT id, customer_id, user_assignment_id, product_name, license_key, quantity, valid_from, valid_until, created_at, updated_at
		 FROM licenses WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&l.ID, &l.CustomerID, &l.UserAssignmentID, &l.ProductName, &l.LicenseKey,
		&l.Quantity, &l.ValidFrom, &l.ValidUntil, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		return l, fmt.Errorf("getting license: %w", err)
	}
	return l, nil
}

func (r *LicenseRepository) ListByCustomer(ctx context.Context, customerID pgtype.UUID, params model.ListParams) ([]model.License, error) {
	params.Normalize()
	query := `SELECT id, customer_id, user_assignment_id, product_name, license_key, quantity, valid_from, valid_until, created_at, updated_at
	          FROM licenses WHERE customer_id = $1 AND deleted_at IS NULL`
	args := []any{customerID}
	argN := 2
	if params.Search != "" {
		query += fmt.Sprintf(` AND product_name ILIKE $%d`, argN)
		args = append(args, "%"+params.Search+"%")
		argN++
	}
	query += fmt.Sprintf(` ORDER BY product_name LIMIT $%d OFFSET $%d`, argN, argN+1)
	args = append(args, params.Limit, params.Offset)
	rows, err := r.pool.Query(ctx, query, args...)
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
			user_assignment_id = $2,
			product_name       = COALESCE($3, product_name),
			license_key        = $4,
			quantity           = COALESCE($5, quantity),
			valid_from         = $6,
			valid_until        = $7
		 WHERE id = $1 AND deleted_at IS NULL
		 RETURNING id, customer_id, user_assignment_id, product_name, license_key, quantity, valid_from, valid_until, created_at, updated_at`,
		id, input.UserAssignmentID, input.ProductName, input.LicenseKey,
		input.Quantity, input.ValidFrom, input.ValidUntil,
	).Scan(&l.ID, &l.CustomerID, &l.UserAssignmentID, &l.ProductName, &l.LicenseKey,
		&l.Quantity, &l.ValidFrom, &l.ValidUntil, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		return l, fmt.Errorf("updating license: %w", err)
	}
	return l, nil
}

func (r *LicenseRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	result, err := r.pool.Exec(ctx, `UPDATE licenses SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("deleting license: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("license not found")
	}
	return nil
}
