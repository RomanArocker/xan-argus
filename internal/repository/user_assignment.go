package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xan-com/xan-pythia/internal/model"
)

type UserAssignmentRepository struct {
	pool *pgxpool.Pool
}

func NewUserAssignmentRepository(pool *pgxpool.Pool) *UserAssignmentRepository {
	return &UserAssignmentRepository{pool: pool}
}

func (r *UserAssignmentRepository) Create(ctx context.Context, input model.CreateUserAssignmentInput) (model.UserAssignment, error) {
	var a model.UserAssignment
	err := r.pool.QueryRow(ctx,
		`INSERT INTO user_assignments (user_id, customer_id, role, email, phone, notes)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, user_id, customer_id, role, email, phone, notes, created_at, updated_at`,
		input.UserID, input.CustomerID, input.Role, input.Email, input.Phone, input.Notes,
	).Scan(&a.ID, &a.UserID, &a.CustomerID, &a.Role, &a.Email, &a.Phone, &a.Notes, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return a, fmt.Errorf("creating user assignment: %w", err)
	}
	return a, nil
}

func (r *UserAssignmentRepository) GetByID(ctx context.Context, id pgtype.UUID) (model.UserAssignment, error) {
	var a model.UserAssignment
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, customer_id, role, email, phone, notes, created_at, updated_at
		 FROM user_assignments WHERE id = $1`, id,
	).Scan(&a.ID, &a.UserID, &a.CustomerID, &a.Role, &a.Email, &a.Phone, &a.Notes, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return a, fmt.Errorf("getting user assignment: %w", err)
	}
	return a, nil
}

func (r *UserAssignmentRepository) ListByCustomer(ctx context.Context, customerID pgtype.UUID, params model.ListParams) ([]model.UserAssignment, error) {
	params.Normalize()
	query := `SELECT id, user_id, customer_id, role, email, phone, notes, created_at, updated_at
	          FROM user_assignments WHERE customer_id = $1`
	args := []any{customerID}
	argN := 2
	if params.Search != "" {
		query += fmt.Sprintf(` AND role ILIKE $%d`, argN)
		args = append(args, "%"+params.Search+"%")
		argN++
	}
	query += fmt.Sprintf(` ORDER BY role LIMIT $%d OFFSET $%d`, argN, argN+1)
	args = append(args, params.Limit, params.Offset)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing user assignments: %w", err)
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToStructByPos[model.UserAssignment])
}

func (r *UserAssignmentRepository) Update(ctx context.Context, id pgtype.UUID, input model.UpdateUserAssignmentInput) (model.UserAssignment, error) {
	var a model.UserAssignment
	err := r.pool.QueryRow(ctx,
		`UPDATE user_assignments SET
			role  = COALESCE($2, role),
			email = COALESCE($3, email),
			phone = COALESCE($4, phone),
			notes = COALESCE($5, notes)
		 WHERE id = $1
		 RETURNING id, user_id, customer_id, role, email, phone, notes, created_at, updated_at`,
		id, input.Role, input.Email, input.Phone, input.Notes,
	).Scan(&a.ID, &a.UserID, &a.CustomerID, &a.Role, &a.Email, &a.Phone, &a.Notes, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return a, fmt.Errorf("updating user assignment: %w", err)
	}
	return a, nil
}

func (r *UserAssignmentRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM user_assignments WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting user assignment: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("user assignment not found")
	}
	return nil
}

func (r *UserAssignmentRepository) GetUserType(ctx context.Context, userID pgtype.UUID) (string, error) {
	var userType string
	err := r.pool.QueryRow(ctx,
		`SELECT type FROM users WHERE id = $1`, userID,
	).Scan(&userType)
	if err != nil {
		return "", fmt.Errorf("getting user type: %w", err)
	}
	return userType, nil
}
