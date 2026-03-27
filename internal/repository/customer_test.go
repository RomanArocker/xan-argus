package repository_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xan-com/xan-argus/internal/model"
	"github.com/xan-com/xan-argus/internal/repository"
)

func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("connecting to test db: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	return pool
}

func TestCustomerCRUD(t *testing.T) {
	pool := setupTestDB(t)
	repo := repository.NewCustomerRepository(pool)
	ctx := context.Background()

	// Create
	input := model.CreateCustomerInput{Name: "Acme Corp"}
	customer, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if customer.Name != "Acme Corp" {
		t.Errorf("Name = %q, want %q", customer.Name, "Acme Corp")
	}
	if !customer.ID.Valid {
		t.Error("ID should be valid")
	}

	// Get
	got, err := repo.GetByID(ctx, customer.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "Acme Corp" {
		t.Errorf("GetByID Name = %q, want %q", got.Name, "Acme Corp")
	}

	// Update
	newName := "Acme Inc"
	updated, err := repo.Update(ctx, customer.ID, model.UpdateCustomerInput{Name: &newName})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "Acme Inc" {
		t.Errorf("Updated Name = %q, want %q", updated.Name, "Acme Inc")
	}

	// List
	customers, err := repo.List(ctx, model.ListParams{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(customers) == 0 {
		t.Error("List returned no customers")
	}

	// Delete
	if err := repo.Delete(ctx, customer.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = repo.GetByID(ctx, customer.ID)
	if err == nil {
		t.Error("GetByID after Delete should return error")
	}
}
