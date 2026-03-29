package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/xan-com/xan-argus/internal/database"
	"github.com/xan-com/xan-argus/internal/handler"
	"github.com/xan-com/xan-argus/internal/importer"
	"github.com/xan-com/xan-argus/internal/middleware"
	"github.com/xan-com/xan-argus/internal/repository"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	// Run migrations
	if err := database.RunMigrations(databaseURL, "db/migrations"); err != nil {
		log.Fatalf("running migrations: %v", err)
	}
	log.Println("Migrations completed")

	// Connect pool
	ctx := context.Background()
	pool, err := database.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer pool.Close()
	log.Println("Database connected")

	// Repositories
	customerRepo := repository.NewCustomerRepository(pool)
	userRepo := repository.NewUserRepository(pool)
	serviceRepo := repository.NewServiceRepository(pool)
	userAssignmentRepo := repository.NewUserAssignmentRepository(pool)
	assetRepo := repository.NewAssetRepository(pool)
	licenseRepo := repository.NewLicenseRepository(pool)
	customerServiceRepo := repository.NewCustomerServiceRepository(pool)
	hardwareCategoryRepo := repository.NewHardwareCategoryRepository(pool)

	// Import/Export
	importRegistry := importer.NewRegistry()
	importEngine := importer.NewEngine(pool, importRegistry)
	importExporter := importer.NewExporter(pool, importRegistry)

	// Router
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Template engine + page handler
	tmpl, err := handler.NewTemplateEngine("web/templates")
	if err != nil {
		log.Fatalf("loading templates: %v", err)
	}
	handler.NewPageHandler(tmpl, customerRepo, userRepo, serviceRepo, userAssignmentRepo, assetRepo, licenseRepo, customerServiceRepo, hardwareCategoryRepo, importRegistry).RegisterRoutes(mux)

	// Register handlers
	handler.NewCustomerHandler(customerRepo).RegisterRoutes(mux)
	handler.NewUserHandler(userRepo).RegisterRoutes(mux)
	handler.NewServiceHandler(serviceRepo).RegisterRoutes(mux)
	handler.NewUserAssignmentHandler(userAssignmentRepo).RegisterRoutes(mux)
	handler.NewAssetHandler(assetRepo, hardwareCategoryRepo).RegisterRoutes(mux)
	handler.NewLicenseHandler(licenseRepo).RegisterRoutes(mux)
	handler.NewCustomerServiceHandler(customerServiceRepo).RegisterRoutes(mux)
	handler.NewHardwareCategoryHandler(hardwareCategoryRepo).RegisterRoutes(mux)
	handler.NewImportHandler(importEngine, importExporter, importRegistry).RegisterRoutes(mux)

	// Static files
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	// Middleware
	srv := middleware.Logging(mux)

	log.Printf("Starting server on :%s", port)
	if err := http.ListenAndServe(":"+port, srv); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
