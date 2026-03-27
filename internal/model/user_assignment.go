package model

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type UserAssignment struct {
	ID         pgtype.UUID `json:"id"`
	UserID     pgtype.UUID `json:"user_id"`
	CustomerID pgtype.UUID `json:"customer_id"`
	Role       string      `json:"role"`
	Email      pgtype.Text `json:"email"`
	Phone      pgtype.Text `json:"phone"`
	Notes      pgtype.Text `json:"notes"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
}

type CreateUserAssignmentInput struct {
	UserID     pgtype.UUID `json:"user_id"`
	CustomerID pgtype.UUID `json:"customer_id"`
	Role       string      `json:"role"`
	Email      *string     `json:"email,omitempty"`
	Phone      *string     `json:"phone,omitempty"`
	Notes      *string     `json:"notes,omitempty"`
}

type UpdateUserAssignmentInput struct {
	Role  *string `json:"role,omitempty"`
	Email *string `json:"email,omitempty"`
	Phone *string `json:"phone,omitempty"`
	Notes *string `json:"notes,omitempty"`
}
