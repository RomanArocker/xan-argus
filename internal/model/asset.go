package model

import (
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type Asset struct {
	ID          pgtype.UUID     `json:"id"`
	CustomerID  pgtype.UUID     `json:"customer_id"`
	CategoryID  pgtype.UUID     `json:"category_id"`
	Name        string          `json:"name"`
	Description pgtype.Text     `json:"description"`
	Metadata    json.RawMessage `json:"metadata"`
	FieldValues json.RawMessage `json:"field_values"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// AssetResponse wraps Asset with an inline category for API responses.
// Kept separate from Asset so RowToStructByPos still works for DB scanning.
type AssetResponse struct {
	Asset
	Category *HardwareCategory `json:"category,omitempty"`
}

type CreateAssetInput struct {
	CustomerID  pgtype.UUID     `json:"customer_id"`
	CategoryID  pgtype.UUID     `json:"category_id,omitempty"`
	Name        string          `json:"name"`
	Description *string         `json:"description,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	FieldValues json.RawMessage `json:"field_values,omitempty"`
}

type UpdateAssetInput struct {
	CategoryID  pgtype.UUID     `json:"category_id,omitempty"`
	Name        *string         `json:"name,omitempty"`
	Description *string         `json:"description,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	FieldValues json.RawMessage `json:"field_values,omitempty"`
}
