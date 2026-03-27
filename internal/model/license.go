package model

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type License struct {
	ID               pgtype.UUID `json:"id"`
	CustomerID       pgtype.UUID `json:"customer_id"`
	UserAssignmentID pgtype.UUID `json:"user_assignment_id"`
	ProductName      string      `json:"product_name"`
	LicenseKey       pgtype.Text `json:"license_key"`
	Quantity         int         `json:"quantity"`
	ValidFrom        pgtype.Date `json:"valid_from"`
	ValidUntil       pgtype.Date `json:"valid_until"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
}

type CreateLicenseInput struct {
	CustomerID       pgtype.UUID  `json:"customer_id"`
	UserAssignmentID *pgtype.UUID `json:"user_assignment_id,omitempty"`
	ProductName      string       `json:"product_name"`
	LicenseKey       *string      `json:"license_key,omitempty"`
	Quantity         int          `json:"quantity"`
	ValidFrom        *pgtype.Date `json:"valid_from,omitempty"`
	ValidUntil       *pgtype.Date `json:"valid_until,omitempty"`
}

type UpdateLicenseInput struct {
	UserAssignmentID *pgtype.UUID `json:"user_assignment_id,omitempty"`
	ProductName      *string      `json:"product_name,omitempty"`
	LicenseKey       *string      `json:"license_key,omitempty"`
	Quantity         *int         `json:"quantity,omitempty"`
	ValidFrom        *pgtype.Date `json:"valid_from,omitempty"`
	ValidUntil       *pgtype.Date `json:"valid_until,omitempty"`
}
