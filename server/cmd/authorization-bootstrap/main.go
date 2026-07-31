// Command authorization-bootstrap creates the first PostgreSQL-managed
// StuHelper super administrator grants. It is a one-time, fail-closed
// break-glass tool and never writes Casdoor role membership or OpenFGA tuples
// directly.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/StuHelper/StuHelper/server/internal/modules/authorization"
	"github.com/StuHelper/StuHelper/server/internal/pkg/db"
)

const bootstrapTimeout = 30 * time.Second
const bootstrapReason = "initial PostgreSQL authorization control-plane bootstrap"

func main() {
	if err := run(os.Args[1:], os.Getenv); err != nil {
		fmt.Fprintf(os.Stderr, "authorization-bootstrap: %v\n", err)
		os.Exit(1)
	}
}

type envReader func(string) string

func run(args []string, getenv envReader) error {
	flags := flag.NewFlagSet("authorization-bootstrap", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	apply := flags.Bool("apply", false, "create missing initial super-admin grants")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}

	usernames, err := parseUsernames(getenv("STUHELPER_INITIAL_SUPER_ADMINS"))
	if err != nil {
		return err
	}
	if strings.EqualFold(strings.TrimSpace(getenv("APP_ENV")), "production") &&
		len(usernames) < 2 {
		return errors.New("production bootstrap requires at least two initial super administrators")
	}
	databaseURL := strings.TrimSpace(getenv("DATABASE_URL"))
	if databaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), bootstrapTimeout)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("connect PostgreSQL: %w", err)
	}
	database := db.NewDB(pool, 5*time.Second)
	defer database.Close()

	service := authorization.NewService(authorization.NewRepository(database))
	hasSuperAdmin, err := service.HasDesiredSuperAdmin(ctx)
	if err != nil {
		return err
	}
	if hasSuperAdmin {
		fmt.Println("authorization_bootstrap skipped=true reason=desired_super_admin_exists")
		return nil
	}

	userIDs := make([]int64, 0, len(usernames))
	for _, username := range usernames {
		userID, resolveErr := service.ResolveInternalUserIDByUsername(ctx, username)
		if resolveErr != nil {
			return fmt.Errorf("resolve initial super admin %q: %w", username, resolveErr)
		}
		userIDs = append(userIDs, userID)
	}
	if !*apply {
		fmt.Printf("authorization_bootstrap apply=false candidates=%d\n", len(userIDs))
		return nil
	}

	result, err := service.BootstrapSuperAdmins(ctx, authorization.BootstrapSuperAdminsInput{
		SubjectUserIDs: userIDs,
		Reason:         bootstrapReason,
	})
	if err != nil {
		return err
	}
	if result.Skipped {
		fmt.Println("authorization_bootstrap skipped=true reason=desired_super_admin_exists")
		return nil
	}
	fmt.Printf(
		"authorization_bootstrap apply=true candidates=%d changed=%d projection=pending\n",
		len(userIDs),
		len(result.Grants),
	)
	return nil
}

func parseUsernames(raw string) ([]string, error) {
	seen := make(map[string]struct{})
	usernames := make([]string, 0)
	for _, value := range strings.Split(raw, ",") {
		username := strings.TrimSpace(value)
		if username == "" {
			continue
		}
		if strings.HasPrefix(username, "REPLACE_WITH_") {
			return nil, errors.New("STUHELPER_INITIAL_SUPER_ADMINS contains a placeholder")
		}
		key := strings.ToLower(username)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		usernames = append(usernames, username)
	}
	if len(usernames) == 0 {
		return nil, errors.New("STUHELPER_INITIAL_SUPER_ADMINS is required")
	}
	return usernames, nil
}
