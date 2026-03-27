package model

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type HardwareCategory struct {
	ID          pgtype.UUID       `json:"id"`
	Name        string            `json:"name"`
	Description pgtype.Text       `json:"description"`
	Fields      []FieldDefinition `json:"fields,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type FieldDefinition struct {
	ID         pgtype.UUID `json:"id"`
	CategoryID pgtype.UUID `json:"category_id"`
	Name       string      `json:"name"`
	FieldType  string      `json:"field_type"`
	Required   bool        `json:"required"`
	SortOrder  int         `json:"sort_order"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
}

type CreateHardwareCategoryInput struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

type UpdateHardwareCategoryInput struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

type CreateFieldDefinitionInput struct {
	CategoryID pgtype.UUID `json:"-"`
	Name       string      `json:"name"`
	FieldType  string      `json:"field_type"`
	SortOrder  *int        `json:"sort_order,omitempty"`
}

type UpdateFieldDefinitionInput struct {
	Name      *string `json:"name,omitempty"`
	SortOrder *int    `json:"sort_order,omitempty"`
}
