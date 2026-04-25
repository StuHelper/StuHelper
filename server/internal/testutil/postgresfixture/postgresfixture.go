package postgresfixture

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/db"
)

type Fixture struct {
	DB      *db.DB
	Pool    *pgxpool.Pool
	URL     string
	closeFn func() error
}

func Start(t *testing.T) *Fixture {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	container, err := postgrescontainer.Run(ctx,
		"postgres:18-alpine",
		postgrescontainer.WithDatabase("stuhelper"),
		postgrescontainer.WithUsername("stuhelper"),
		postgrescontainer.WithPassword("stuhelper"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(2*time.Minute),
		),
	)
	require.NoError(t, err)

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	applyMigrations(t, connStr)

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)

	wrapped := db.NewDB(pool, 5*time.Second)
	fixture := &Fixture{
		DB:   wrapped,
		Pool: pool,
		URL:  connStr,
		closeFn: func() error {
			wrapped.Close()
			termCtx, termCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer termCancel()
			return container.Terminate(termCtx)
		},
	}
	t.Cleanup(fixturesCleanup(t, fixture))
	return fixture
}

func fixturesCleanup(t *testing.T, f *Fixture) func() {
	return func() {
		t.Helper()
		if f != nil && f.closeFn != nil {
			require.NoError(t, f.closeFn())
		}
	}
}

func applyMigrations(t *testing.T, databaseURL string) {
	t.Helper()

	sourceURL := "file://" + filepath.ToSlash(filepath.Join(serverRoot(), "migrations"))
	m, err := migrate.New(sourceURL, databaseURL)
	require.NoError(t, err)
	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil {
			t.Fatalf("close migrate source: %v", srcErr)
		}
		if dbErr != nil {
			t.Fatalf("close migrate db: %v", dbErr)
		}
	}()

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		require.NoError(t, err)
	}
}

func serverRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("postgresfixture: runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	if root == "." || root == "/" {
		panic(fmt.Sprintf("postgresfixture: unexpected server root resolved from %q", file))
	}
	return root
}
