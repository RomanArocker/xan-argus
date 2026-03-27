package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xan-com/xan-pythia/internal/model"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, input model.CreateUserInput) (model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (type, first_name, last_name)
		 VALUES ($1, $2, $3)
		 RETURNING id, type, first_name, last_name, created_at, updated_at`,
		input.Type, input.FirstName, input.LastName,
	).Scan(&u.ID, &u.Type, &u.FirstName, &u.LastName, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return u, fmt.Errorf("creating user: %w", err)
	}
	return u, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id pgtype.UUID) (model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx,
		`SELECT id, type, first_name, last_name, created_at, updated_at
		 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Type, &u.FirstName, &u.LastName, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return u, fmt.Errorf("getting user: %w", err)
	}
	return u, nil
}

func (r *UserRepository) List(ctx context.Context, params model.ListParams) ([]model.User, error) {
	params.Normalize()
	query := `SELECT id, type, first_name, last_name, created_at, updated_at FROM users`
	args := []any{}
	argN := 1
	clauses := []string{}
	if params.Filter != "" {
		clauses = append(clauses, fmt.Sprintf(`type = $%d`, argN))
		args = append(args, params.Filter)
		argN++
	}
	if params.Search != "" {
		clauses = append(clauses, fmt.Sprintf(`(first_name ILIKE $%d OR last_name ILIKE $%d)`, argN, argN))
		args = append(args, "%"+params.Search+"%")
		argN++
	}
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += fmt.Sprintf(` ORDER BY last_name, first_name LIMIT $%d OFFSET $%d`, argN, argN+1)
	args = append(args, params.Limit, params.Offset)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, pgx.RowToStructByPos[model.User])
}

func (r *UserRepository) Update(ctx context.Context, id pgtype.UUID, input model.UpdateUserInput) (model.User, error) {
	var u model.User
	err := r.pool.QueryRow(ctx,
		`UPDATE users SET
			type = COALESCE($2, type),
			first_name = COALESCE($3, first_name),
			last_name = COALESCE($4, last_name)
		 WHERE id = $1
		 RETURNING id, type, first_name, last_name, created_at, updated_at`,
		id, input.Type, input.FirstName, input.LastName,
	).Scan(&u.ID, &u.Type, &u.FirstName, &u.LastName, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return u, fmt.Errorf("updating user: %w", err)
	}
	return u, nil
}

func (r *UserRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}
