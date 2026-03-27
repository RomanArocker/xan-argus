package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xan-com/xan-argus/internal/model"
)

type CustomerRepository struct {
	pool *pgxpool.Pool
}

func NewCustomerRepository(pool *pgxpool.Pool) *CustomerRepository {
	return &CustomerRepository{pool: pool}
}

func (r *CustomerRepository) Create(ctx context.Context, input model.CreateCustomerInput) (model.Customer, error) {
	var c model.Customer
	err := r.pool.QueryRow(ctx,
		`INSERT INTO customers (name, contact_email, notes)
		 VALUES ($1, $2, $3)
		 RETURNING id, name, contact_email, notes, created_at, updated_at`,
		input.Name, input.ContactEmail, input.Notes,
	).Scan(&c.ID, &c.Name, &c.ContactEmail, &c.Notes, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return c, fmt.Errorf("creating customer: %w", err)
	}
	return c, nil
}

func (r *CustomerRepository) GetByID(ctx context.Context, id pgtype.UUID) (model.Customer, error) {
	var c model.Customer
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, contact_email, notes, created_at, updated_at
		 FROM customers WHERE id = $1`, id,
	).Scan(&c.ID, &c.Name, &c.ContactEmail, &c.Notes, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return c, fmt.Errorf("getting customer: %w", err)
	}
	return c, nil
}

func (r *CustomerRepository) List(ctx context.Context, params model.ListParams) ([]model.Customer, error) {
	params.Normalize()
	query := `SELECT id, name, contact_email, notes, created_at, updated_at FROM customers`
	args := []any{}
	if params.Search != "" {
		query += ` WHERE name ILIKE $1`
		args = append(args, "%"+params.Search+"%")
	}
	query += fmt.Sprintf(` ORDER BY name LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2)
	args = append(args, params.Limit, params.Offset)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing customers: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowToStructByPos[model.Customer])
}

func (r *CustomerRepository) Update(ctx context.Context, id pgtype.UUID, input model.UpdateCustomerInput) (model.Customer, error) {
	var c model.Customer
	err := r.pool.QueryRow(ctx,
		`UPDATE customers SET
			name = COALESCE($2, name),
			contact_email = COALESCE($3, contact_email),
			notes = COALESCE($4, notes)
		 WHERE id = $1
		 RETURNING id, name, contact_email, notes, created_at, updated_at`,
		id, input.Name, input.ContactEmail, input.Notes,
	).Scan(&c.ID, &c.Name, &c.ContactEmail, &c.Notes, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return c, fmt.Errorf("updating customer: %w", err)
	}
	return c, nil
}

func (r *CustomerRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM customers WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting customer: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("customer not found")
	}
	return nil
}
