package postgresfixture

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
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

type sharedPostgresServer struct {
	adminURL string
}

var sharedPostgres = struct {
	once   sync.Once
	server *sharedPostgresServer
	err    error
	seq    atomic.Uint64
}{}

func Start(t *testing.T) *Fixture {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	server := startSharedPostgres(t, ctx)
	dbName := nextDatabaseName()
	createTestDatabase(t, ctx, server.adminURL, dbName)

	connStr := databaseURLForTest(t, server.adminURL, dbName)
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
			return dropTestDatabase(server.adminURL, dbName)
		},
	}
	t.Cleanup(fixturesCleanup(t, fixture))
	return fixture
}

func startSharedPostgres(t *testing.T, ctx context.Context) *sharedPostgresServer {
	t.Helper()

	sharedPostgres.once.Do(func() {
		sharedPostgres.server, sharedPostgres.err = runSharedPostgres(ctx)
	})
	require.NoError(t, sharedPostgres.err)
	return sharedPostgres.server
}

func runSharedPostgres(ctx context.Context) (*sharedPostgresServer, error) {
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
	if err != nil {
		return nil, err
	}

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, err
	}
	return &sharedPostgresServer{adminURL: databaseURLForName(connStr, "postgres")}, nil
}

func nextDatabaseName() string {
	return fmt.Sprintf("stuhelper_test_%d", sharedPostgres.seq.Add(1))
}

func createTestDatabase(t *testing.T, ctx context.Context, adminURL string, dbName string) {
	t.Helper()

	conn, err := pgx.Connect(ctx, adminURL)
	require.NoError(t, err)
	defer conn.Close(ctx)

	_, err = conn.Exec(ctx, "CREATE DATABASE "+quoteIdentifier(dbName))
	require.NoError(t, err)
}

func dropTestDatabase(adminURL string, dbName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	_, execErr := conn.Exec(ctx, "DROP DATABASE IF EXISTS "+quoteIdentifier(dbName)+" WITH (FORCE)")
	return execErr
}

func databaseURLForTest(t *testing.T, rawURL string, dbName string) string {
	t.Helper()

	connStr, err := databaseURLWithName(rawURL, dbName)
	require.NoError(t, err)
	return connStr
}

func databaseURLForName(rawURL string, dbName string) string {
	connStr, err := databaseURLWithName(rawURL, dbName)
	if err != nil {
		panic(err)
	}
	return connStr
}

func databaseURLWithName(rawURL string, dbName string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	parsed.Path = "/" + dbName
	return parsed.String(), nil
}

func quoteIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
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
