package repository_test

import (
	"context"
	"testing"

	"github.com/xan-com/xan-argus/internal/model"
	"github.com/xan-com/xan-argus/internal/repository"
)

func TestHardwareCategoryCRUD(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	repo := repository.NewHardwareCategoryRepository(pool)

	// Create
	input := model.CreateHardwareCategoryInput{Name: "Test Category"}
	cat, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if cat.Name != "Test Category" {
		t.Errorf("Name = %q, want %q", cat.Name, "Test Category")
	}
	if !cat.ID.Valid {
		t.Error("ID should be valid")
	}
	t.Cleanup(func() { repo.Delete(ctx, cat.ID) }) //nolint:errcheck

	// GetByID (no fields yet)
	got, err := repo.GetByID(ctx, cat.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "Test Category" {
		t.Errorf("GetByID Name = %q, want %q", got.Name, "Test Category")
	}
	if len(got.Fields) != 0 {
		t.Errorf("Fields count = %d, want 0", len(got.Fields))
	}

	// Update
	newName := "Updated Category"
	updated, err := repo.Update(ctx, cat.ID, model.UpdateHardwareCategoryInput{Name: &newName})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "Updated Category" {
		t.Errorf("Updated Name = %q, want %q", updated.Name, "Updated Category")
	}

	// List (should include seeded + test category)
	categories, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(categories) == 0 {
		t.Error("List returned no results")
	}

	// Delete
	if err := repo.Delete(ctx, cat.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = repo.GetByID(ctx, cat.ID)
	if err == nil {
		t.Error("GetByID after Delete should return error")
	}
}

func TestFieldDefinitionCRUD(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	repo := repository.NewHardwareCategoryRepository(pool)

	// Prereq: category
	cat, err := repo.Create(ctx, model.CreateHardwareCategoryInput{Name: "Field Test Category"})
	if err != nil {
		t.Fatalf("Create category: %v", err)
	}
	t.Cleanup(func() { repo.Delete(ctx, cat.ID) }) //nolint:errcheck

	// Create field
	fieldInput := model.CreateFieldDefinitionInput{
		CategoryID: cat.ID,
		Name:       "RAM",
		FieldType:  "number",
	}
	field, err := repo.CreateField(ctx, fieldInput)
	if err != nil {
		t.Fatalf("CreateField: %v", err)
	}
	if field.Name != "RAM" {
		t.Errorf("Name = %q, want %q", field.Name, "RAM")
	}
	if field.FieldType != "number" {
		t.Errorf("FieldType = %q, want %q", field.FieldType, "number")
	}
	if field.Required {
		t.Error("Required should be false by default")
	}

	// GetByID should include field
	got, err := repo.GetByID(ctx, cat.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if len(got.Fields) != 1 {
		t.Fatalf("Fields count = %d, want 1", len(got.Fields))
	}
	if got.Fields[0].Name != "RAM" {
		t.Errorf("Field Name = %q, want %q", got.Fields[0].Name, "RAM")
	}

	// Update field
	newFieldName := "Memory (GB)"
	updatedField, err := repo.UpdateField(ctx, field.ID, model.UpdateFieldDefinitionInput{Name: &newFieldName})
	if err != nil {
		t.Fatalf("UpdateField: %v", err)
	}
	if updatedField.Name != "Memory (GB)" {
		t.Errorf("Updated field Name = %q, want %q", updatedField.Name, "Memory (GB)")
	}

	// Delete field
	if err := repo.DeleteField(ctx, field.ID); err != nil {
		t.Fatalf("DeleteField: %v", err)
	}
	got2, err := repo.GetByID(ctx, cat.ID)
	if err != nil {
		t.Fatalf("GetByID after field delete: %v", err)
	}
	if len(got2.Fields) != 0 {
		t.Errorf("Fields count after delete = %d, want 0", len(got2.Fields))
	}
}
