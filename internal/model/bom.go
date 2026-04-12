package model

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// BOMItem represents one position in an asset's Bill of Materials.
type BOMItem struct {
	ID        pgtype.UUID    `json:"id"`
	AssetID   pgtype.UUID    `json:"asset_id"`
	Name      string         `json:"name"`
	Quantity  pgtype.Numeric `json:"quantity"`
	UnitID    pgtype.UUID    `json:"unit_id"`
	UnitName  string         `json:"unit_name"` // joined from units
	UnitPrice pgtype.Numeric `json:"unit_price"`
	Currency  string         `json:"currency"`
	Notes     pgtype.Text    `json:"notes"`
	SortOrder int            `json:"sort_order"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type CreateBOMItemInput struct {
	Name      string         `json:"name"`
	Quantity  pgtype.Numeric `json:"quantity"`
	UnitID    pgtype.UUID    `json:"unit_id"`
	UnitPrice pgtype.Numeric `json:"unit_price"`
	Currency  string         `json:"currency"`
	Notes     *string        `json:"notes,omitempty"`
	SortOrder *int           `json:"sort_order,omitempty"`
}

// UpdateBOMItemInput — nil pointer = keep existing. Notes uses direct assignment (nil clears it).
type UpdateBOMItemInput struct {
	Name      *string         `json:"name,omitempty"`
	Quantity  *pgtype.Numeric `json:"quantity,omitempty"`
	UnitID    *pgtype.UUID    `json:"unit_id,omitempty"`
	UnitPrice *pgtype.Numeric `json:"unit_price,omitempty"`
	Currency  *string         `json:"currency,omitempty"`
	Notes     *string         `json:"notes,omitempty"`
	SortOrder *int            `json:"sort_order,omitempty"`
}

type Unit struct {
	ID   pgtype.UUID `json:"id"`
	Name string      `json:"name"`
}

// BOMTotals holds the summed line totals (quantity * unit_price) for one currency.
type BOMTotals struct {
	Currency string         `json:"currency"`
	Total    pgtype.Numeric `json:"total"`
}
