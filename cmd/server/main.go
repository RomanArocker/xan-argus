package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/xan-com/xan-pythia/internal/database"
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

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"ok"}`)
	})

	log.Printf("Starting server on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
