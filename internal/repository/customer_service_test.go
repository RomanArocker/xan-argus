package repository_test

import (
	"context"
	"testing"

	"github.com/xan-com/xan-pythia/internal/model"
	"github.com/xan-com/xan-pythia/internal/repository"
)

func TestCustomerServiceCRUD(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	// Prereqs: customer + service
	customerRepo := repository.NewCustomerRepository(pool)
	customer, err := customerRepo.Create(ctx, model.CreateCustomerInput{Name: "CS Test Customer"})
	if err != nil {
		t.Fatalf("Create customer: %v", err)
	}
	t.Cleanup(func() { customerRepo.Delete(ctx, customer.ID) }) //nolint:errcheck

	serviceRepo := repository.NewServiceRepository(pool)
	service, err := serviceRepo.Create(ctx, model.CreateServiceInput{Name: "CS Test Service"})
	if err != nil {
		t.Fatalf("Create service: %v", err)
	}
	t.Cleanup(func() { serviceRepo.Delete(ctx, service.ID) }) //nolint:errcheck

	repo := repository.NewCustomerServiceRepository(pool)

	// Create
	input := model.CreateCustomerServiceInput{
		CustomerID: customer.ID,
		ServiceID:  service.ID,
	}
	cs, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !cs.ID.Valid {
		t.Error("ID should be valid")
	}
	// Customizations defaults to {}
	if string(cs.Customizations) == "" {
		t.Error("Customizations should not be empty")
	}
	t.Cleanup(func() { repo.Delete(ctx, cs.ID) }) //nolint:errcheck

	// GetByID
	got, err := repo.GetByID(ctx, cs.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ServiceID != service.ID {
		t.Errorf("GetByID ServiceID mismatch")
	}

	// Update
	notes := "updated notes"
	updated, err := repo.Update(ctx, cs.ID, model.UpdateCustomerServiceInput{Notes: &notes})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !updated.Notes.Valid || updated.Notes.String != "updated notes" {
		t.Errorf("Updated Notes = %v, want 'updated notes'", updated.Notes)
	}

	// ListByCustomer
	services, err := repo.ListByCustomer(ctx, customer.ID, model.ListParams{Limit: 10})
	if err != nil {
		t.Fatalf("ListByCustomer: %v", err)
	}
	if len(services) == 0 {
		t.Error("ListByCustomer returned no results")
	}

	// Unique violation: subscribe same service again
	_, err = repo.Create(ctx, model.CreateCustomerServiceInput{
		CustomerID: customer.ID,
		ServiceID:  service.ID,
	})
	if err == nil {
		t.Error("expected unique violation, got nil")
	}

	// Delete
	if err := repo.Delete(ctx, cs.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = repo.GetByID(ctx, cs.ID)
	if err == nil {
		t.Error("GetByID after Delete should return error")
	}
}
