package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/piimigration"
)

const piiClosureVersion uint = 18

func sanitizeVersionForLog(version uint) string {
	return fmt.Sprintf("%d", version)
}

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
		version = 0
	}
	if dirty {
		log.Fatalf("migration state is dirty at version %s", sanitizeVersionForLog(version))
	}

	if version < piiClosureVersion {
		if err := migrateTo(m, piiClosureVersion-1); err != nil {
			log.Fatalf("failed to migrate to version %d: %v", piiClosureVersion-1, err)
		}
		if err := runPIIClosure(databaseURL); err != nil {
			log.Fatalf("failed to close PII migration gap: %v", err)
		}
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("failed to apply migrations: %v", err)
	}
}

func migrateTo(m *migrate.Migrate, version uint) error {
	if err := m.Migrate(version); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

func runPIIClosure(databaseURL string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("connect db: %w", err)
	}
	defer pool.Close()

	pending, err := piimigration.CollectPendingSfzjh(ctx, pool)
	if err != nil {
		return err
	}
	if len(pending) == 0 {
		return nil
	}

	hmacKey, err := piimigration.LoadHMACKeyFromEnv()
	if err != nil {
		return err
	}
	cipher, err := piimigration.BuildCipherFromEnv()
	if err != nil {
		return err
	}
	if err := piimigration.ApplyPendingSfzjh(ctx, pool, pending, cipher, hmacKey, false); err != nil {
		return err
	}
	return nil
}
