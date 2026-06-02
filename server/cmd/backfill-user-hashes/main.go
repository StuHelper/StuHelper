package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/user"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
)

const defaultTimeout = 30 * time.Second

func main() {
	apply, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "backfill-user-hashes: %v\n", err)
		os.Exit(2)
	}
	if err := run(apply); err != nil {
		fmt.Fprintf(os.Stderr, "backfill-user-hashes: %v\n", err)
		os.Exit(1)
	}
}

func parseArgs(args []string) (bool, error) {
	apply := false
	for _, arg := range args {
		switch arg {
		case "--apply":
			apply = true
		case "--dry-run":
			apply = false
		case "--help", "-h":
			printUsage()
			os.Exit(0)
		default:
			return false, fmt.Errorf("unknown argument %q", arg)
		}
	}
	return apply, nil
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: backfill-user-hashes [--dry-run|--apply]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Dry-run counts users where user_hash is missing. --apply backfills them using HMAC_SECRET.")
}

func run(apply bool) error {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	hmacSecret := os.Getenv("HMAC_SECRET")
	appEnv := envWithDefault("APP_ENV", config.EnvDevelopment)
	isProduction := config.IsProductionLikeEnv(appEnv)
	if err := crypto.InitHMACKey(hmacSecret, isProduction); err != nil {
		return fmt.Errorf("initialize HMAC key: %w", err)
	}

	dbCfg := config.DatabaseConfig{
		URL:             databaseURL,
		MaxConns:        envInt32("DB_MAX_CONNS", 4),
		MinConns:        envInt32("DB_MIN_CONNS", 0),
		MaxConnLifetime: envInt("DB_MAX_CONN_LIFETIME", 30),
		MaxConnIdleTime: envInt("DB_MAX_CONN_IDLE_TIME", 5),
		QueryTimeout:    envInt("DB_QUERY_TIMEOUT", 5),
		SSLMode:         envWithDefault("DB_SSL_MODE", "disable"),
		SSLRootCert:     os.Getenv("DB_SSL_ROOT_CERT"),
		SSLCert:         os.Getenv("DB_SSL_CERT"),
		SSLKey:          os.Getenv("DB_SSL_KEY"),
	}
	pool, err := db.NewPGPool(dbCfg)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	database := db.NewDB(pool, time.Duration(dbCfg.QueryTimeout)*time.Second)
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	repo := user.NewUserSyncRepository(database, crypto.GetHMACKey())
	before, err := repo.CountMissingUserHashes(ctx)
	if err != nil {
		return err
	}
	if !apply {
		fmt.Printf("apply=false pending_user_hashes=%d updated=0 remaining=%d\n", before, before)
		return nil
	}

	updated, err := repo.BackfillUserHashes(ctx)
	if err != nil {
		return err
	}
	remaining, err := repo.CountMissingUserHashes(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("apply=true pending_user_hashes=%d updated=%d remaining=%d\n", before, updated, remaining)
	return nil
}

func envWithDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envInt32(key string, fallback int32) int32 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return fallback
	}
	return int32(parsed)
}
