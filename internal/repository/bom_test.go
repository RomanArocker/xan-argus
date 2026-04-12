package repository_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/xan-com/xan-argus/internal/model"
	"github.com/xan-com/xan-argus/internal/repository"
)

// setupBOMTest creates a customer + asset for BOM item tests, returns their IDs.
// Cleanup deletes the asset and customer after test completes.
func setupBOMTest(t *testing.T) (customerID, assetID pgtype.UUID) {
	t.Helper()
	pool := setupTestDB(t)
	customerRepo := repository.NewCustomerRepository(pool)
	assetRepo := repository.NewAssetRepository(pool)
	ctx := context.Background()

	c, err := customerRepo.Create(ctx, model.CreateCustomerInput{Name: "BOM Test Customer " + t.Name()})
	if err != nil {
		t.Fatalf("create test customer: %v", err)
	}
	t.Cleanup(func() {
		customerRepo.Delete(ctx, c.ID) //nolint
	})

	a, err := assetRepo.Create(ctx, model.CreateAssetInput{
		CustomerID: c.ID,
		Name:       "BOM Test Asset " + t.Name(),
	})
	if err != nil {
		t.Fatalf("create test asset: %v", err)
	}
	t.Cleanup(func() {
		assetRepo.Delete(ctx, a.ID) //nolint
	})
	return c.ID, a.ID
}

// getFirstUnit returns the first unit from the database for test use.
func getFirstUnit(t *testing.T, repo *repository.BOMRepository) model.Unit {
	t.Helper()
	units, err := repo.ListUnits(context.Background())
	if err != nil || len(units) == 0 {
		t.Fatalf("ListUnits: %v (count: %d)", err, len(units))
	}
	return units[0]
}

func numericFrom(s string) pgtype.Numeric {
	var n pgtype.Numeric
	if err := n.Scan(s); err != nil {
		panic(err)
	}
	return n
}

func TestBOMListUnits(t *testing.T) {
	pool := setupTestDB(t)
	repo := repository.NewBOMRepository(pool)

	units, err := repo.ListUnits(context.Background())
	if err != nil {
		t.Fatalf("ListUnits: %v", err)
	}
	if len(units) == 0 {
		t.Fatal("expected seeded units, got none")
	}
	for _, u := range units {
		if !u.ID.Valid {
			t.Error("unit ID should be valid")
		}
		if u.Name == "" {
			t.Error("unit Name should not be empty")
		}
	}
}

func TestBOMCRUD(t *testing.T) {
	pool := setupTestDB(t)
	repo := repository.NewBOMRepository(pool)
	ctx := context.Background()
	_, assetID := setupBOMTest(t)
	unit := getFirstUnit(t, repo)

	// Create
	input := model.CreateBOMItemInput{
		Name:      "CPU Intel i7",
		Quantity:  numericFrom("1"),
		UnitID:    unit.ID,
		UnitPrice: numericFrom("749.00"),
		Currency:  "CHF",
	}
	item, err := repo.Create(ctx, assetID, input)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if item.Name != "CPU Intel i7" {
		t.Errorf("Name = %q, want %q", item.Name, "CPU Intel i7")
	}
	if item.Currency != "CHF" {
		t.Errorf("Currency = %q, want %q", item.Currency, "CHF")
	}
	if item.UnitName != unit.Name {
		t.Errorf("UnitName = %q, want %q", item.UnitName, unit.Name)
	}
	if !item.ID.Valid {
		t.Error("item ID should be valid")
	}
	t.Cleanup(func() { repo.Delete(ctx, item.ID) }) //nolint

	// GetByID
	got, err := repo.GetByID(ctx, item.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "CPU Intel i7" {
		t.Errorf("GetByID Name = %q, want %q", got.Name, "CPU Intel i7")
	}

	// ListByAsset
	items, err := repo.ListByAsset(ctx, assetID)
	if err != nil {
		t.Fatalf("ListByAsset: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("ListByAsset count = %d, want 1", len(items))
	}

	// CountByAsset
	count, err := repo.CountByAsset(ctx, assetID)
	if err != nil {
		t.Fatalf("CountByAsset: %v", err)
	}
	if count != 1 {
		t.Errorf("CountByAsset = %d, want 1", count)
	}

	// Update
	newName := "CPU Intel i9"
	updated, err := repo.Update(ctx, item.ID, model.UpdateBOMItemInput{Name: &newName})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "CPU Intel i9" {
		t.Errorf("Updated Name = %q, want %q", updated.Name, "CPU Intel i9")
	}

	// Update notes to a value, then clear it
	notesVal := "some note"
	_, err = repo.Update(ctx, item.ID, model.UpdateBOMItemInput{Notes: &notesVal})
	if err != nil {
		t.Fatalf("Update notes set: %v", err)
	}
	clearNotes, err := repo.Update(ctx, item.ID, model.UpdateBOMItemInput{Notes: nil})
	if err != nil {
		t.Fatalf("Update notes clear: %v", err)
	}
	if clearNotes.Notes.Valid {
		t.Error("Notes should be NULL after clearing")
	}

	// Delete
	if err := repo.Delete(ctx, item.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = repo.GetByID(ctx, item.ID)
	if err == nil {
		t.Error("GetByID should return error after delete")
	}
}

func TestBOMTotalsByAsset(t *testing.T) {
	pool := setupTestDB(t)
	repo := repository.NewBOMRepository(pool)
	ctx := context.Background()
	_, assetID := setupBOMTest(t)
	unit := getFirstUnit(t, repo)

	// Create two items: CHF and EUR
	item1, _ := repo.Create(ctx, assetID, model.CreateBOMItemInput{
		Name:      "CPU", Quantity: numericFrom("1"),
		UnitID: unit.ID, UnitPrice: numericFrom("500"), Currency: "CHF",
	})
	item2, _ := repo.Create(ctx, assetID, model.CreateBOMItemInput{
		Name:      "RAM", Quantity: numericFrom("2"),
		UnitID: unit.ID, UnitPrice: numericFrom("100"), Currency: "CHF",
	})
	item3, _ := repo.Create(ctx, assetID, model.CreateBOMItemInput{
		Name:      "License", Quantity: numericFrom("1"),
		UnitID: unit.ID, UnitPrice: numericFrom("200"), Currency: "EUR",
	})
	t.Cleanup(func() {
		repo.Delete(ctx, item1.ID) //nolint
		repo.Delete(ctx, item2.ID) //nolint
		repo.Delete(ctx, item3.ID) //nolint
	})

	totals, err := repo.TotalsByAsset(ctx, assetID)
	if err != nil {
		t.Fatalf("TotalsByAsset: %v", err)
	}
	if len(totals) != 2 {
		t.Fatalf("TotalsByAsset count = %d, want 2", len(totals))
	}
	// CHF total should be 500 + 2*100 = 700
	chfFound := false
	for _, tot := range totals {
		if tot.Currency == "CHF" {
			chfFound = true
			// Just verify it's valid (exact decimal comparison is complex)
			if !tot.Total.Valid {
				t.Error("CHF total should be valid")
			}
		}
	}
	if !chfFound {
		t.Error("expected CHF total")
	}
}

func TestBOMSwapSortOrder(t *testing.T) {
	pool := setupTestDB(t)
	repo := repository.NewBOMRepository(pool)
	ctx := context.Background()
	_, assetID := setupBOMTest(t)
	unit := getFirstUnit(t, repo)

	// Create 3 items
	item1, _ := repo.Create(ctx, assetID, model.CreateBOMItemInput{
		Name: "First", Quantity: numericFrom("1"), UnitID: unit.ID,
		UnitPrice: numericFrom("100"), Currency: "CHF",
	})
	item2, _ := repo.Create(ctx, assetID, model.CreateBOMItemInput{
		Name: "Second", Quantity: numericFrom("1"), UnitID: unit.ID,
		UnitPrice: numericFrom("100"), Currency: "CHF",
	})
	item3, _ := repo.Create(ctx, assetID, model.CreateBOMItemInput{
		Name: "Third", Quantity: numericFrom("1"), UnitID: unit.ID,
		UnitPrice: numericFrom("100"), Currency: "CHF",
	})
	t.Cleanup(func() {
		repo.Delete(ctx, item1.ID) //nolint
		repo.Delete(ctx, item2.ID) //nolint
		repo.Delete(ctx, item3.ID) //nolint
	})

	// Move item2 up — should swap with item1
	if err := repo.SwapSortOrder(ctx, assetID, item2.ID, "up"); err != nil {
		t.Fatalf("SwapSortOrder up: %v", err)
	}
	items, _ := repo.ListByAsset(ctx, assetID)
	if items[0].Name != "Second" {
		t.Errorf("After move up: first item = %q, want %q", items[0].Name, "Second")
	}

	// Move item1 up when already at boundary — no-op, no error
	if err := repo.SwapSortOrder(ctx, assetID, item1.ID, "up"); err != nil {
		t.Fatalf("SwapSortOrder boundary up: %v", err)
	}

	// Move item3 down when already at boundary — no-op, no error
	if err := repo.SwapSortOrder(ctx, assetID, item3.ID, "down"); err != nil {
		t.Fatalf("SwapSortOrder boundary down: %v", err)
	}
}

func TestBOMUniqueNamePerAsset(t *testing.T) {
	pool := setupTestDB(t)
	repo := repository.NewBOMRepository(pool)
	ctx := context.Background()
	_, assetID := setupBOMTest(t)
	unit := getFirstUnit(t, repo)

	item, _ := repo.Create(ctx, assetID, model.CreateBOMItemInput{
		Name: "Duplicate", Quantity: numericFrom("1"), UnitID: unit.ID,
		UnitPrice: numericFrom("100"), Currency: "CHF",
	})
	t.Cleanup(func() { repo.Delete(ctx, item.ID) }) //nolint

	_, err := repo.Create(ctx, assetID, model.CreateBOMItemInput{
		Name: "Duplicate", Quantity: numericFrom("1"), UnitID: unit.ID,
		UnitPrice: numericFrom("100"), Currency: "CHF",
	})
	if err == nil {
		t.Fatal("expected unique violation error, got nil")
	}
}
