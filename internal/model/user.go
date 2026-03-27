package model

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type User struct {
	ID        pgtype.UUID `json:"id"`
	Type      string      `json:"type"`
	FirstName string      `json:"first_name"`
	LastName  string      `json:"last_name"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

type CreateUserInput struct {
	Type      string `json:"type"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type UpdateUserInput struct {
	Type      *string `json:"type,omitempty"`
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
}
