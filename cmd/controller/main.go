package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"nas-controller/internal/api"
	"nas-controller/internal/database"
	"nas-controller/internal/docker"
	"nas-controller/internal/services"
)

func main() {
	port := flag.String("port", "13000", "Port to run the controller on")
	dataDir := flag.String("data", "/data", "Data directory for repos, db, logs")
	flag.Parse()

	// Ensure data directory exists
	if err := os.MkdirAll(*dataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Ensure subdirectories exist
	for _, subdir := range []string{"repos", "logs", "icons"} {
		if err := os.MkdirAll(filepath.Join(*dataDir, subdir), 0755); err != nil {
			log.Fatalf("Failed to create %s directory: %v", subdir, err)
		}
	}

	// Initialize database
	db, err := database.New(filepath.Join(*dataDir, "controller.db"))
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize Docker client
	dockerClient, err := docker.NewClient()
	if err != nil {
		log.Fatalf("Failed to connect to Docker: %v", err)
	}
	defer dockerClient.Close()

	// Initialize services
	authService := services.NewAuthService(*dataDir)
	portAllocator := services.NewPortAllocator(db, dockerClient)
	gitService := services.NewGitService(*dataDir)
	buildService := services.NewBuildService(dockerClient, *dataDir)
	appManager := services.NewAppManager(db, dockerClient, gitService, buildService, portAllocator, *dataDir)

	// Check/generate password on first run
	password, isNew, err := authService.EnsurePassword()
	if err != nil {
		log.Fatalf("Failed to initialize authentication: %v", err)
	}
	if isNew {
		log.Printf("========================================")
		log.Printf("FIRST RUN - Generated password: %s", password)
		log.Printf("Save this password! It's also stored in %s/password.txt", *dataDir)
		log.Printf("========================================")
	}

	// Reconcile app states with Docker on startup
	if err := appManager.ReconcileStates(); err != nil {
		log.Printf("Warning: Failed to reconcile app states: %v", err)
	}

	// Start API server
	router := api.NewRouter(db, dockerClient, authService, appManager, buildService, portAllocator, *dataDir)

	log.Printf("NAS Controller starting on port %s", *port)
	log.Printf("Data directory: %s", *dataDir)

	if err := router.Run(":" + *port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
