package main

import (
	"errors"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("Warning: failed to load .env: %v", err)
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	m, err := migrate.New("file://migrations", databaseURL)
	if err != nil {
		log.Fatalf("failed to initialize migrator: %v", err)
	}
	defer func() {
		sourceErr, databaseErr := m.Close()
		if sourceErr != nil {
			log.Printf("Warning: failed to close migrate source: %v", sourceErr)
		}
		if databaseErr != nil {
			log.Printf("Warning: failed to close migrate database: %v", databaseErr)
		}
	}()

	version, dirty, err := m.Version()
	if err != nil {
		if !errors.Is(err, migrate.ErrNilVersion) {
			log.Fatalf("failed to read migration version: %v", err)
		}
	}
	if dirty {
		log.Fatalf("migration state is dirty at version %d", version)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("failed to apply migrations: %v", err)
	}
}
