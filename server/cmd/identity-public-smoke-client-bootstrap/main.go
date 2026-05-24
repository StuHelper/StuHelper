package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/openplatform"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
)

func main() {
	_ = godotenv.Load()

	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}

	database := db.NewDB(pool, 10*time.Second)
	defer database.Close()

	repository := openplatform.NewRepository(database)
	result, err := openplatform.BootstrapIdentityPublicSmokeClient(ctx, repository, openplatform.IdentityPublicSmokeClientBootstrapInput{
		OwnerUserID:        requiredInt64("IDENTITY_PUBLIC_SMOKE_OWNER_USER_ID"),
		ReviewerUserID:     requiredInt64("IDENTITY_PUBLIC_SMOKE_REVIEWER_USER_ID"),
		ClientID:           envDefault("IDENTITY_PUBLIC_SMOKE_CLIENT_ID", "identity-public-smoke"),
		ClientSecret:       os.Getenv("IDENTITY_PUBLIC_SMOKE_CLIENT_SECRET"),
		DisplayName:        envDefault("IDENTITY_PUBLIC_SMOKE_DISPLAY_NAME", "Identity Public Smoke"),
		Description:        envDefault("IDENTITY_PUBLIC_SMOKE_DESCRIPTION", "Dedicated approved client for production public Identity smoke checks."),
		HomepageURL:        requiredString("IDENTITY_PUBLIC_SMOKE_HOMEPAGE_URL"),
		PrivacyPolicyURL:   requiredString("IDENTITY_PUBLIC_SMOKE_PRIVACY_POLICY_URL"),
		RedirectURI:        requiredString("IDENTITY_PUBLIC_SMOKE_REDIRECT_URI"),
		ClientScopes:       strings.Fields(envDefault("IDENTITY_PUBLIC_SMOKE_CLIENT_CREDENTIALS_SCOPE", openplatform.ScopeResourceRead)),
		RequestID:          envDefault("IDENTITY_PUBLIC_SMOKE_BOOTSTRAP_REQUEST_ID", "identity-public-smoke-bootstrap"),
		AllowRevokedRepair: envBool("IDENTITY_PUBLIC_SMOKE_BOOTSTRAP_ALLOW_REVOKED_REPAIR"),
	})
	if err != nil {
		log.Fatalf("bootstrap identity public smoke client: %v", err)
	}

	fmt.Printf("IDENTITY_PUBLIC_SMOKE_CLIENT_ID=%s\n", shellValue(result.App.ClientID))
	fmt.Printf("IDENTITY_PUBLIC_SMOKE_REDIRECT_URI=%s\n", shellValue(result.App.RedirectURIs[0]))
	fmt.Printf("IDENTITY_PUBLIC_SMOKE_CLIENT_CREDENTIALS_SCOPE=%s\n", shellValue(strings.Join(result.ClientScopes, " ")))
	fmt.Printf("IDENTITY_PUBLIC_SMOKE_CLIENT_SECRET=%s\n", shellValue(result.ClientSecret))
	fmt.Println("IDENTITY_PUBLIC_SMOKE_RESOURCE_ACCESS_EXPECT_ALLOWED=false")
}

func requiredString(key string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		log.Fatalf("%s is required", key)
	}
	return value
}

func requiredInt64(key string) int64 {
	raw := requiredString(key)
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		log.Fatalf("%s must be a positive integer", key)
	}
	return value
}

func envDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envBool(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "y":
		return true
	default:
		return false
	}
}

func shellValue(value string) string {
	return strings.ReplaceAll(value, "\n", "")
}
