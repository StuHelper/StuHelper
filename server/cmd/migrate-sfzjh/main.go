package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/piimigration"
)

func sanitizeCountForLog(count int) string {
	return fmt.Sprintf("%d", count)
}

func main() {
	dryRun := flag.Bool("dry-run", false, "read-only mode: log what would be changed without writing")
	flag.Parse()

	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("Warning: failed to load .env: %v", err)
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	pending, err := piimigration.CollectPendingSfzjh(ctx, pool)
	if err != nil {
		log.Fatalf("Failed to collect pending rows: %v", err)
	}
	if len(pending) == 0 {
		log.Println("No rows need migration. All sfzjh_enc values are already encrypted or empty.")
		return
	}

	hmacKey, err := piimigration.LoadHMACKeyFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	cipher, err := piimigration.BuildCipherFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	if err := piimigration.ApplyPendingSfzjh(ctx, pool, pending, cipher, hmacKey, *dryRun); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
	log.Printf("Migration complete: %s rows processed", sanitizeCountForLog(len(pending)))
}
