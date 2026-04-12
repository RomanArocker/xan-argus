package repository_test

import (
	"context"
	"testing"

	"github.com/xan-com/xan-argus/internal/model"
	"github.com/xan-com/xan-argus/internal/repository"
)

func TestUserCRUD(t *testing.T) {
	pool := setupTestDB(t)
	repo := repository.NewUserRepository(pool)
	ctx := context.Background()

	input := model.CreateUserInput{
		Type: "internal_staff", FirstName: "Jane", LastName: "Doe",
	}
	user, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if user.FirstName != "Jane" {
		t.Errorf("FirstName = %q, want %q", user.FirstName, "Jane")
	}

	got, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.LastName != "Doe" {
		t.Errorf("LastName = %q, want %q", got.LastName, "Doe")
	}

	newFirst := "Janet"
	updated, err := repo.Update(ctx, user.ID, model.UpdateUserInput{FirstName: &newFirst})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.FirstName != "Janet" {
		t.Errorf("Updated FirstName = %q", updated.FirstName)
	}

	users, err := repo.List(ctx, model.ListParams{Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(users) == 0 {
		t.Error("List returned no users")
	}

	if err := repo.Delete(ctx, user.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestListAvailableForCustomer(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()

	customerRepo := repository.NewCustomerRepository(pool)
	customer, err := customerRepo.Create(ctx, model.CreateCustomerInput{Name: "Avail Filter Customer"})
	if err != nil {
		t.Fatalf("Create customer: %v", err)
	}
	t.Cleanup(func() { customerRepo.Delete(ctx, customer.ID) }) //nolint:errcheck

	userRepo := repository.NewUserRepository(pool)

	// customer_staff: assigned
	assigned, err := userRepo.Create(ctx, model.CreateUserInput{Type: "customer_staff", FirstName: "Assigned", LastName: "User"})
	if err != nil {
		t.Fatalf("Create assigned user: %v", err)
	}
	t.Cleanup(func() { userRepo.Delete(ctx, assigned.ID) }) //nolint:errcheck

	// customer_staff: not yet assigned
	available, err := userRepo.Create(ctx, model.CreateUserInput{Type: "customer_staff", FirstName: "Available", LastName: "User"})
	if err != nil {
		t.Fatalf("Create available user: %v", err)
	}
	t.Cleanup(func() { userRepo.Delete(ctx, available.ID) }) //nolint:errcheck

	// internal_staff: should never appear in results
	internal, err := userRepo.Create(ctx, model.CreateUserInput{Type: "internal_staff", FirstName: "Internal", LastName: "User"})
	if err != nil {
		t.Fatalf("Create internal user: %v", err)
	}
	t.Cleanup(func() { userRepo.Delete(ctx, internal.ID) }) //nolint:errcheck

	// Assign the "assigned" user to the customer
	assignmentRepo := repository.NewUserAssignmentRepository(pool)
	assignment, err := assignmentRepo.Create(ctx, model.CreateUserAssignmentInput{
		UserID:     assigned.ID,
		CustomerID: customer.ID,
		Role:       "viewer",
	})
	if err != nil {
		t.Fatalf("Create assignment: %v", err)
	}
	t.Cleanup(func() { assignmentRepo.Delete(ctx, assignment.ID) }) //nolint:errcheck

	results, err := userRepo.ListAvailableForCustomer(ctx, customer.ID)
	if err != nil {
		t.Fatalf("ListAvailableForCustomer: %v", err)
	}

	ids := make(map[string]bool, len(results))
	for _, u := range results {
		ids[u.FirstName] = true
	}

	if ids["Assigned"] {
		t.Error("already-assigned user should not appear in results")
	}
	if !ids["Available"] {
		t.Error("unassigned customer_staff should appear in results")
	}
	if ids["Internal"] {
		t.Error("internal_staff should not appear in results")
	}
}
