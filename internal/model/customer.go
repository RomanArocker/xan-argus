package model

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type Customer struct {
	ID           pgtype.UUID `json:"id"`
	Name         string      `json:"name"`
	ContactEmail pgtype.Text `json:"contact_email"`
	Notes        pgtype.Text `json:"notes"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

type CreateCustomerInput struct {
	Name         string  `json:"name"`
	ContactEmail *string `json:"contact_email,omitempty"`
	Notes        *string `json:"notes,omitempty"`
}

type UpdateCustomerInput struct {
	Name         *string `json:"name,omitempty"`
	ContactEmail *string `json:"contact_email,omitempty"`
	Notes        *string `json:"notes,omitempty"`
}
