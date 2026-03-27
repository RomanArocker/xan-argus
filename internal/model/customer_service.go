package model

import (
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type CustomerService struct {
	ID             pgtype.UUID     `json:"id"`
	CustomerID     pgtype.UUID     `json:"customer_id"`
	ServiceID      pgtype.UUID     `json:"service_id"`
	Customizations json.RawMessage `json:"customizations"`
	Notes          pgtype.Text     `json:"notes"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type CreateCustomerServiceInput struct {
	CustomerID     pgtype.UUID     `json:"customer_id"`
	ServiceID      pgtype.UUID     `json:"service_id"`
	Customizations json.RawMessage `json:"customizations,omitempty"`
	Notes          *string         `json:"notes,omitempty"`
}

type UpdateCustomerServiceInput struct {
	Customizations json.RawMessage `json:"customizations,omitempty"`
	Notes          *string         `json:"notes,omitempty"`
}
