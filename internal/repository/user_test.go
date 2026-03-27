package repository_test

import (
	"context"
	"testing"

	"github.com/xan-com/xan-pythia/internal/model"
	"github.com/xan-com/xan-pythia/internal/repository"
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
