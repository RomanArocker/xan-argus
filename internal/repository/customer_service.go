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

type CustomerServiceRepository struct {
	pool *pgxpool.Pool
}

func NewCustomerServiceRepository(pool *pgxpool.Pool) *CustomerServiceRepository {
	return &CustomerServiceRepository{pool: pool}
}

func (r *CustomerServiceRepository) Create(ctx context.Context, input model.CreateCustomerServiceInput) (model.CustomerService, error) {
	if input.Customizations == nil {
		input.Customizations = json.RawMessage("{}")
	}
	var cs model.CustomerService
	err := r.pool.QueryRow(ctx,
		`INSERT INTO customer_services (customer_id, service_id, customizations, notes)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, customer_id, service_id, customizations, notes, created_at, updated_at`,
		input.CustomerID, input.ServiceID, input.Customizations, input.Notes,
	).Scan(&cs.ID, &cs.CustomerID, &cs.ServiceID, &cs.Customizations, &cs.Notes, &cs.CreatedAt, &cs.UpdatedAt)
	if err != nil {
		return cs, fmt.Errorf("creating customer service: %w", err)
	}
	return cs, nil
}

func (r *CustomerServiceRepository) GetByID(ctx context.Context, id pgtype.UUID) (model.CustomerService, error) {
	var cs model.CustomerService
	err := r.pool.QueryRow(ctx,
		`SELECT id, customer_id, service_id, customizations, notes, created_at, updated_at
		 FROM customer_services WHERE id = $1`, id,
	).Scan(&cs.ID, &cs.CustomerID, &cs.ServiceID, &cs.Customizations, &cs.Notes, &cs.CreatedAt, &cs.UpdatedAt)
	if err != nil {
		return cs, fmt.Errorf("getting customer service: %w", err)
	}
	return cs, nil
}

func (r *CustomerServiceRepository) ListByCustomer(ctx context.Context, customerID pgtype.UUID, params model.ListParams) ([]model.CustomerService, error) {
	params.Normalize()
	query := `SELECT id, customer_id, service_id, customizations, notes, created_at, updated_at
	          FROM customer_services WHERE customer_id = $1`
	args := []any{customerID}
	argN := 2
	if params.Search != "" {
		query += fmt.Sprintf(` AND notes ILIKE $%d`, argN)
		args = append(args, "%"+params.Search+"%")
		argN++
	}
	query += fmt.Sprintf(` ORDER BY created_at LIMIT $%d OFFSET $%d`, argN, argN+1)
	args = append(args, params.Limit, params.Offset)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing customer services: %w", err)
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToStructByPos[model.CustomerService])
}

func (r *CustomerServiceRepository) Update(ctx context.Context, id pgtype.UUID, input model.UpdateCustomerServiceInput) (model.CustomerService, error) {
	var cs model.CustomerService
	err := r.pool.QueryRow(ctx,
		`UPDATE customer_services SET
			customizations = COALESCE($2, customizations),
			notes          = COALESCE($3, notes)
		 WHERE id = $1
		 RETURNING id, customer_id, service_id, customizations, notes, created_at, updated_at`,
		id, input.Customizations, input.Notes,
	).Scan(&cs.ID, &cs.CustomerID, &cs.ServiceID, &cs.Customizations, &cs.Notes, &cs.CreatedAt, &cs.UpdatedAt)
	if err != nil {
		return cs, fmt.Errorf("updating customer service: %w", err)
	}
	return cs, nil
}

func (r *CustomerServiceRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM customer_services WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting customer service: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("customer service not found")
	}
	return nil
}
