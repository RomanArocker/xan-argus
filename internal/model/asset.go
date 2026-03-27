package model

import (
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type Asset struct {
	ID          pgtype.UUID     `json:"id"`
	CustomerID  pgtype.UUID     `json:"customer_id"`
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Description pgtype.Text     `json:"description"`
	Metadata    json.RawMessage `json:"metadata"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type CreateAssetInput struct {
	CustomerID  pgtype.UUID     `json:"customer_id"`
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Description *string         `json:"description,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
}

type UpdateAssetInput struct {
	Type        *string         `json:"type,omitempty"`
	Name        *string         `json:"name,omitempty"`
	Description *string         `json:"description,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
}
