package repository_test

import (
	"context"
	"testing"

	"github.com/xan-com/xan-argus/internal/model"
	"github.com/xan-com/xan-argus/internal/repository"
)

func TestUserAssignmentCRUD(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	// Prereqs: customer + user
	customerRepo := repository.NewCustomerRepository(pool)
	customer, err := customerRepo.Create(ctx, model.CreateCustomerInput{Name: "UA Test Customer"})
	if err != nil {
		t.Fatalf("Create customer: %v", err)
	}
	t.Cleanup(func() { customerRepo.Delete(ctx, customer.ID) }) //nolint:errcheck

	userRepo := repository.NewUserRepository(pool)
	user, err := userRepo.Create(ctx, model.CreateUserInput{Type: "customer_staff", FirstName: "UA", LastName: "TestUser"})
	if err != nil {
		t.Fatalf("Create user: %v", err)
	}
	t.Cleanup(func() { userRepo.Delete(ctx, user.ID) }) //nolint:errcheck

	repo := repository.NewUserAssignmentRepository(pool)

	// Create
	input := model.CreateUserAssignmentInput{
		UserID:     user.ID,
		CustomerID: customer.ID,
		Role:       "admin",
	}
	assignment, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if assignment.Role != "admin" {
		t.Errorf("Role = %q, want %q", assignment.Role, "admin")
	}
	if !assignment.ID.Valid {
		t.Error("ID should be valid")
	}
	t.Cleanup(func() { repo.Delete(ctx, assignment.ID) }) //nolint:errcheck

	// GetByID
	got, err := repo.GetByID(ctx, assignment.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Role != "admin" {
		t.Errorf("GetByID Role = %q, want %q", got.Role, "admin")
	}

	// Update
	newRole := "viewer"
	updated, err := repo.Update(ctx, assignment.ID, model.UpdateUserAssignmentInput{Role: &newRole})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Role != "viewer" {
		t.Errorf("Updated Role = %q, want %q", updated.Role, "viewer")
	}

	// ListByCustomer
	assignments, err := repo.ListByCustomer(ctx, customer.ID, model.ListParams{Limit: 10})
	if err != nil {
		t.Fatalf("ListByCustomer: %v", err)
	}
	if len(assignments) == 0 {
		t.Error("ListByCustomer returned no results")
	}

	// Unique violation: create same user+customer again
	_, err = repo.Create(ctx, model.CreateUserAssignmentInput{
		UserID:     user.ID,
		CustomerID: customer.ID,
		Role:       "viewer",
	})
	if err == nil {
		t.Error("expected unique violation, got nil")
	}

	// Delete
	if err := repo.Delete(ctx, assignment.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = repo.GetByID(ctx, assignment.ID)
	if err == nil {
		t.Error("GetByID after Delete should return error")
	}
}

func TestGetUserType(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	userRepo := repository.NewUserRepository(pool)

	customerStaff, err := userRepo.Create(ctx, model.CreateUserInput{Type: "customer_staff", FirstName: "CS", LastName: "User"})
	if err != nil {
		t.Fatalf("Create customer_staff: %v", err)
	}
	t.Cleanup(func() { userRepo.Delete(ctx, customerStaff.ID) }) //nolint:errcheck

	internalStaff, err := userRepo.Create(ctx, model.CreateUserInput{Type: "internal_staff", FirstName: "IS", LastName: "User"})
	if err != nil {
		t.Fatalf("Create internal_staff: %v", err)
	}
	t.Cleanup(func() { userRepo.Delete(ctx, internalStaff.ID) }) //nolint:errcheck

	repo := repository.NewUserAssignmentRepository(pool)

	typ, err := repo.GetUserType(ctx, customerStaff.ID)
	if err != nil {
		t.Fatalf("GetUserType customer_staff: %v", err)
	}
	if typ != "customer_staff" {
		t.Errorf("got %q, want %q", typ, "customer_staff")
	}

	typ, err = repo.GetUserType(ctx, internalStaff.ID)
	if err != nil {
		t.Fatalf("GetUserType internal_staff: %v", err)
	}
	if typ != "internal_staff" {
		t.Errorf("got %q, want %q", typ, "internal_staff")
	}
}

func TestUserAssignmentRejectsInternalStaff(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	customerRepo := repository.NewCustomerRepository(pool)
	customer, err := customerRepo.Create(ctx, model.CreateCustomerInput{Name: "Type Check Customer"})
	if err != nil {
		t.Fatalf("Create customer: %v", err)
	}
	t.Cleanup(func() { customerRepo.Delete(ctx, customer.ID) }) //nolint:errcheck

	userRepo := repository.NewUserRepository(pool)
	internalUser, err := userRepo.Create(ctx, model.CreateUserInput{Type: "internal_staff", FirstName: "Internal", LastName: "User"})
	if err != nil {
		t.Fatalf("Create internal user: %v", err)
	}
	t.Cleanup(func() { userRepo.Delete(ctx, internalUser.ID) }) //nolint:errcheck

	repo := repository.NewUserAssignmentRepository(pool)
	_, err = repo.Create(ctx, model.CreateUserAssignmentInput{
		UserID:     internalUser.ID,
		CustomerID: customer.ID,
		Role:       "admin",
	})
	if err == nil {
		t.Fatal("expected error when assigning internal_staff to customer, got nil")
	}
}
