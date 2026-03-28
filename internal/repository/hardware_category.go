package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xan-com/xan-argus/internal/model"
)

type HardwareCategoryRepository struct {
	pool *pgxpool.Pool
}

func NewHardwareCategoryRepository(pool *pgxpool.Pool) *HardwareCategoryRepository {
	return &HardwareCategoryRepository{pool: pool}
}

// --- Categories ---

func (r *HardwareCategoryRepository) Create(ctx context.Context, input model.CreateHardwareCategoryInput) (model.HardwareCategory, error) {
	var c model.HardwareCategory
	err := r.pool.QueryRow(ctx,
		`INSERT INTO hardware_categories (name, description)
		 VALUES ($1, $2)
		 RETURNING id, name, description, created_at, updated_at`,
		input.Name, input.Description,
	).Scan(&c.ID, &c.Name, &c.Description, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return c, fmt.Errorf("creating hardware category: %w", err)
	}
	return c, nil
}

func (r *HardwareCategoryRepository) GetByID(ctx context.Context, id pgtype.UUID) (model.HardwareCategory, error) {
	var c model.HardwareCategory
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, description, created_at, updated_at
		 FROM hardware_categories WHERE id = $1`, id,
	).Scan(&c.ID, &c.Name, &c.Description, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return c, fmt.Errorf("getting hardware category: %w", err)
	}

	fields, err := r.ListFields(ctx, id)
	if err != nil {
		return c, fmt.Errorf("listing fields for category: %w", err)
	}
	c.Fields = fields
	return c, nil
}

func (r *HardwareCategoryRepository) List(ctx context.Context) ([]model.HardwareCategory, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, description, created_at, updated_at
		 FROM hardware_categories ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("listing hardware categories: %w", err)
	}
	defer rows.Close()
	// Manual scan — HardwareCategory has a Fields slice that RowToStructByPos can't skip
	var categories []model.HardwareCategory
	for rows.Next() {
		var c model.HardwareCategory
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning hardware category: %w", err)
		}
		categories = append(categories, c)
	}
	return categories, rows.Err()
}

func (r *HardwareCategoryRepository) Update(ctx context.Context, id pgtype.UUID, input model.UpdateHardwareCategoryInput) (model.HardwareCategory, error) {
	var c model.HardwareCategory
	err := r.pool.QueryRow(ctx,
		`UPDATE hardware_categories SET
			name        = COALESCE($2, name),
			description = COALESCE($3, description)
		 WHERE id = $1
		 RETURNING id, name, description, created_at, updated_at`,
		id, input.Name, input.Description,
	).Scan(&c.ID, &c.Name, &c.Description, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return c, fmt.Errorf("updating hardware category: %w", err)
	}
	return c, nil
}

func (r *HardwareCategoryRepository) Delete(ctx context.Context, id pgtype.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM hardware_categories WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting hardware category: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// --- Field Definitions ---

func (r *HardwareCategoryRepository) ListFields(ctx context.Context, categoryID pgtype.UUID) ([]model.FieldDefinition, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, category_id, name, field_type, required, sort_order, created_at, updated_at
		 FROM category_field_definitions
		 WHERE category_id = $1
		 ORDER BY sort_order, name`, categoryID)
	if err != nil {
		return nil, fmt.Errorf("listing field definitions: %w", err)
	}
	defer rows.Close()
	return pgx.CollectRows(rows, pgx.RowToStructByPos[model.FieldDefinition])
}

func (r *HardwareCategoryRepository) CreateField(ctx context.Context, input model.CreateFieldDefinitionInput) (model.FieldDefinition, error) {
	var f model.FieldDefinition
	err := r.pool.QueryRow(ctx,
		`INSERT INTO category_field_definitions (category_id, name, field_type, sort_order)
		 VALUES ($1, $2, $3,
		   (SELECT COALESCE(MAX(sort_order), -1) + 1
		    FROM category_field_definitions WHERE category_id = $1))
		 RETURNING id, category_id, name, field_type, required, sort_order, created_at, updated_at`,
		input.CategoryID, input.Name, input.FieldType,
	).Scan(&f.ID, &f.CategoryID, &f.Name, &f.FieldType, &f.Required, &f.SortOrder, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return f, fmt.Errorf("creating field definition: %w", err)
	}
	return f, nil
}

func (r *HardwareCategoryRepository) UpdateField(ctx context.Context, fieldID pgtype.UUID, input model.UpdateFieldDefinitionInput) (model.FieldDefinition, error) {
	var f model.FieldDefinition
	err := r.pool.QueryRow(ctx,
		`UPDATE category_field_definitions SET
			name       = COALESCE($2, name),
			sort_order = COALESCE($3, sort_order)
		 WHERE id = $1
		 RETURNING id, category_id, name, field_type, required, sort_order, created_at, updated_at`,
		fieldID, input.Name, input.SortOrder,
	).Scan(&f.ID, &f.CategoryID, &f.Name, &f.FieldType, &f.Required, &f.SortOrder, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return f, fmt.Errorf("updating field definition: %w", err)
	}
	return f, nil
}

func (r *HardwareCategoryRepository) DeleteField(ctx context.Context, fieldID pgtype.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM category_field_definitions WHERE id = $1`, fieldID)
	if err != nil {
		return fmt.Errorf("deleting field definition: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
