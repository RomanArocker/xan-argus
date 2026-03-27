package repository_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/xan-com/xan-pythia/internal/model"
	"github.com/xan-com/xan-pythia/internal/repository"
)

func TestAssetCRUD(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	// Prereqs: customer and category
	customerRepo := repository.NewCustomerRepository(pool)
	customer, err := customerRepo.Create(ctx, model.CreateCustomerInput{Name: "Asset Test Customer"})
	if err != nil {
		t.Fatalf("Create customer: %v", err)
	}
	t.Cleanup(func() { customerRepo.Delete(ctx, customer.ID) }) //nolint:errcheck

	catRepo := repository.NewHardwareCategoryRepository(pool)
	cat, err := catRepo.Create(ctx, model.CreateHardwareCategoryInput{Name: "Test Asset Category"})
	if err != nil {
		t.Fatalf("Create category: %v", err)
	}
	t.Cleanup(func() { catRepo.Delete(ctx, cat.ID) }) //nolint:errcheck

	repo := repository.NewAssetRepository(pool)

	// Create with category
	input := model.CreateAssetInput{
		CustomerID:  customer.ID,
		CategoryID:  cat.ID,
		Name:        "Test Laptop",
		FieldValues: json.RawMessage(`{"some-field": "some-value"}`),
	}
	asset, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if asset.Name != "Test Laptop" {
		t.Errorf("Name = %q, want %q", asset.Name, "Test Laptop")
	}
	if !asset.CategoryID.Valid {
		t.Error("CategoryID should be valid")
	}
	if !asset.ID.Valid {
		t.Error("ID should be valid")
	}
	if string(asset.Metadata) == "" {
		t.Error("Metadata should not be empty")
	}
	if string(asset.FieldValues) == "" {
		t.Error("FieldValues should not be empty")
	}
	t.Cleanup(func() { repo.Delete(ctx, asset.ID) }) //nolint:errcheck

	// Create without category (nullable)
	inputNoCat := model.CreateAssetInput{
		CustomerID: customer.ID,
		Name:       "Uncategorized Device",
	}
	assetNoCat, err := repo.Create(ctx, inputNoCat)
	if err != nil {
		t.Fatalf("Create without category: %v", err)
	}
	if assetNoCat.CategoryID.Valid {
		t.Error("CategoryID should be null for uncategorized asset")
	}
	t.Cleanup(func() { repo.Delete(ctx, assetNoCat.ID) }) //nolint:errcheck

	// GetByID
	got, err := repo.GetByID(ctx, asset.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "Test Laptop" {
		t.Errorf("GetByID Name = %q, want %q", got.Name, "Test Laptop")
	}

	// Update
	newName := "Updated Laptop"
	updated, err := repo.Update(ctx, asset.ID, model.UpdateAssetInput{Name: &newName})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "Updated Laptop" {
		t.Errorf("Updated Name = %q, want %q", updated.Name, "Updated Laptop")
	}

	// ListByCustomer
	assets, err := repo.ListByCustomer(ctx, customer.ID, model.ListParams{Limit: 10})
	if err != nil {
		t.Fatalf("ListByCustomer: %v", err)
	}
	if len(assets) < 2 {
		t.Errorf("ListByCustomer returned %d results, want at least 2", len(assets))
	}

	// ListByCustomer with search
	searched, err := repo.ListByCustomer(ctx, customer.ID, model.ListParams{Limit: 10, Search: "laptop"})
	if err != nil {
		t.Fatalf("ListByCustomer with search: %v", err)
	}
	if len(searched) == 0 {
		t.Error("ListByCustomer with search returned no results")
	}

	// Delete
	if err := repo.Delete(ctx, asset.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = repo.GetByID(ctx, asset.ID)
	if err == nil {
		t.Error("GetByID after Delete should return error")
	}
}
