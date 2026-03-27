package repository_test

import (
	"context"
	"testing"

	"github.com/xan-com/xan-argus/internal/model"
	"github.com/xan-com/xan-argus/internal/repository"
)

func TestServiceCRUD(t *testing.T) {
	pool := setupTestDB(t)
	repo := repository.NewServiceRepository(pool)
	ctx := context.Background()

	desc := "Managed email"
	input := model.CreateServiceInput{Name: "Email Hosting", Description: &desc}
	svc, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if svc.Name != "Email Hosting" {
		t.Errorf("Name = %q", svc.Name)
	}

	got, err := repo.GetByID(ctx, svc.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "Email Hosting" {
		t.Errorf("Name = %q", got.Name)
	}

	newName := "Email Premium"
	updated, err := repo.Update(ctx, svc.ID, model.UpdateServiceInput{Name: &newName})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "Email Premium" {
		t.Errorf("Name = %q", updated.Name)
	}

	services, err := repo.List(ctx, model.ListParams{Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(services) == 0 {
		t.Error("List empty")
	}

	if err := repo.Delete(ctx, svc.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}
