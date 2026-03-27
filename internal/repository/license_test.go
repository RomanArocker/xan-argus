package repository_test

import (
	"context"
	"strings"
	"testing"

	"github.com/xan-com/xan-pythia/internal/model"
	"github.com/xan-com/xan-pythia/internal/repository"
)

func TestLicenseCRUD(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	// Prereq: customer
	customerRepo := repository.NewCustomerRepository(pool)
	customer, err := customerRepo.Create(ctx, model.CreateCustomerInput{Name: "License Test Customer"})
	if err != nil {
		t.Fatalf("Create customer: %v", err)
	}
	t.Cleanup(func() { customerRepo.Delete(ctx, customer.ID) }) //nolint:errcheck

	repo := repository.NewLicenseRepository(pool)

	// Create
	input := model.CreateLicenseInput{
		CustomerID:  customer.ID,
		ProductName: "Acrobat Pro",
		Quantity:    5,
	}
	license, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if license.ProductName != "Acrobat Pro" {
		t.Errorf("ProductName = %q, want %q", license.ProductName, "Acrobat Pro")
	}
	if license.Quantity != 5 {
		t.Errorf("Quantity = %d, want %d", license.Quantity, 5)
	}
	if !license.ID.Valid {
		t.Error("ID should be valid")
	}
	t.Cleanup(func() { repo.Delete(ctx, license.ID) }) //nolint:errcheck

	// GetByID
	got, err := repo.GetByID(ctx, license.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ProductName != "Acrobat Pro" {
		t.Errorf("GetByID ProductName = %q, want %q", got.ProductName, "Acrobat Pro")
	}

	// Update
	newName := "Acrobat Pro DC"
	newQty := 10
	updated, err := repo.Update(ctx, license.ID, model.UpdateLicenseInput{
		ProductName: &newName,
		Quantity:    &newQty,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.ProductName != "Acrobat Pro DC" {
		t.Errorf("Updated ProductName = %q, want %q", updated.ProductName, "Acrobat Pro DC")
	}
	if updated.Quantity != 10 {
		t.Errorf("Updated Quantity = %d, want %d", updated.Quantity, 10)
	}

	// ListByCustomer
	licenses, err := repo.ListByCustomer(ctx, customer.ID, model.ListParams{Limit: 10})
	if err != nil {
		t.Fatalf("ListByCustomer: %v", err)
	}
	if len(licenses) == 0 {
		t.Error("ListByCustomer returned no results")
	}

	// Delete
	if err := repo.Delete(ctx, license.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = repo.GetByID(ctx, license.ID)
	if err == nil {
		t.Error("GetByID after Delete should return error")
	}
}

func TestLicenseConsistencyTrigger(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	customerRepo := repository.NewCustomerRepository(pool)
	userRepo := repository.NewUserRepository(pool)
	uaRepo := repository.NewUserAssignmentRepository(pool)
	licenseRepo := repository.NewLicenseRepository(pool)

	// Create two customers
	customerA, err := customerRepo.Create(ctx, model.CreateCustomerInput{Name: "Trigger Test Customer A"})
	if err != nil {
		t.Fatalf("Create customerA: %v", err)
	}
	t.Cleanup(func() { customerRepo.Delete(ctx, customerA.ID) }) //nolint:errcheck

	customerB, err := customerRepo.Create(ctx, model.CreateCustomerInput{Name: "Trigger Test Customer B"})
	if err != nil {
		t.Fatalf("Create customerB: %v", err)
	}
	t.Cleanup(func() { customerRepo.Delete(ctx, customerB.ID) }) //nolint:errcheck

	// Create user
	user, err := userRepo.Create(ctx, model.CreateUserInput{Type: "employee", FirstName: "Trigger", LastName: "TestUser"})
	if err != nil {
		t.Fatalf("Create user: %v", err)
	}
	t.Cleanup(func() { userRepo.Delete(ctx, user.ID) }) //nolint:errcheck

	// Assign user to customerA
	assignment, err := uaRepo.Create(ctx, model.CreateUserAssignmentInput{
		UserID:     user.ID,
		CustomerID: customerA.ID,
		Role:       "user",
	})
	if err != nil {
		t.Fatalf("Create user assignment: %v", err)
	}
	t.Cleanup(func() { uaRepo.Delete(ctx, assignment.ID) }) //nolint:errcheck

	// Try to create license for customerB with assignment from customerA — expect trigger error
	_, err = licenseRepo.Create(ctx, model.CreateLicenseInput{
		CustomerID:       customerB.ID,
		UserAssignmentID: &assignment.ID,
		ProductName:      "Should Fail",
		Quantity:         1,
	})
	if err == nil {
		t.Fatal("expected consistency trigger error, got nil")
	}
	if !strings.Contains(err.Error(), "does not match") {
		t.Errorf("expected 'does not match' in error, got: %v", err)
	}
}
