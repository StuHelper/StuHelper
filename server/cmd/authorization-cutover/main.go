// Command authorization-cutover performs the one-time, fail-closed migration
// from the legacy Casdoor/OpenFGA authorization state into PostgreSQL's
// authorization_grants control-plane ledger.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/StuHelper/StuHelper/server/internal/modules/authorization"
	"github.com/StuHelper/StuHelper/server/internal/modules/user"
	"github.com/StuHelper/StuHelper/server/internal/pkg/config"
	"github.com/StuHelper/StuHelper/server/internal/pkg/db"
	"github.com/StuHelper/StuHelper/server/internal/pkg/fga"
	platformcasdoor "github.com/StuHelper/StuHelper/server/internal/platform/casdoor"
)

const cutoverTimeout = 10 * time.Minute

var legacyScopedRoles = []authorization.Role{
	authorization.RoleSchoolAdmin,
	authorization.RoleSectionAdmin,
	authorization.RoleSectionModerator,
	authorization.RoleSectionReviewer,
}

type settings struct {
	database config.DatabaseConfig
	openFGA  config.OpenFGAConfig
	casdoor  platformcasdoor.Credential
}

type safeResult struct {
	Changed            bool   `json:"changed"`
	SourceDigest       string `json:"sourceDigest"`
	ImportedGrantCount int    `json:"importedGrantCount"`
	SkippedTupleCount  int    `json:"skippedTupleCount"`
}

func main() {
	if err := run(os.Getenv, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "authorization-cutover: %v\n", err)
		os.Exit(1)
	}
}

func run(getenv func(string) string, output io.Writer) error {
	settings, err := loadSettings(getenv)
	if err != nil {
		return err
	}

	pool, err := db.NewPGPool(settings.database)
	if err != nil {
		return fmt.Errorf("connect PostgreSQL: %w", err)
	}
	database := db.NewDB(pool, time.Duration(settings.database.QueryTimeout)*time.Second)
	defer database.Close()

	fgaClient, err := fga.NewClient(settings.openFGA)
	if err != nil {
		return fmt.Errorf("create OpenFGA client: %w", err)
	}
	service := authorization.NewService(
		authorization.NewRepository(database),
		authorization.WithProjectionClient(fgaClient),
	)

	ctx, cancel := context.WithTimeout(context.Background(), cutoverTimeout)
	defer cancel()
	status, err := service.AuthorityCutoverStatus(ctx)
	if err != nil {
		return fmt.Errorf("load durable cutover marker: %w", err)
	}
	if status.Completed {
		return writeResult(output, authorization.AuthorityCutoverResult{
			Changed:            false,
			SourceDigest:       status.SourceDigest,
			ImportedGrantCount: status.ImportedGrantCount,
		})
	}
	linkedIdentities, err := user.NewAuthorityCutoverRepository(database).ListLinkedIdentities(ctx)
	if err != nil {
		return fmt.Errorf("read local authority identity links: %w", err)
	}

	casdoorClient, err := platformcasdoor.NewAuthoritySnapshotClient(settings.casdoor)
	if err != nil {
		return fmt.Errorf("create Casdoor authority snapshot client: %w", err)
	}
	roleNames := make([]string, 0, len(legacyScopedRoles))
	for _, role := range legacyScopedRoles {
		roleNames = append(roleNames, string(role))
	}
	providerSnapshot, err := casdoorClient.Snapshot(ctx, roleNames)
	if err != nil {
		return fmt.Errorf("read Casdoor authority snapshot: %w", err)
	}

	result, err := service.ImportLegacyAuthority(
		ctx,
		toAuthorityCutoverUsers(linkedIdentities),
		toLegacyAuthoritySnapshot(providerSnapshot),
		fgaClient,
	)
	if err != nil {
		return fmt.Errorf("import verified authority snapshot: %w", err)
	}
	return writeResult(output, result)
}

func toAuthorityCutoverUsers(input []user.AuthorityCutoverIdentity) []authorization.AuthorityCutoverUser {
	result := make([]authorization.AuthorityCutoverUser, 0, len(input))
	for _, identity := range input {
		result = append(result, authorization.AuthorityCutoverUser{
			InternalUserID:  identity.InternalUserID,
			ProviderSubject: identity.ProviderSubject,
		})
	}
	return result
}

func loadSettings(getenv func(string) string) (settings, error) {
	required := func(key string) (string, error) {
		value := strings.TrimSpace(getenv(key))
		if value == "" {
			return "", fmt.Errorf("%s is required", key)
		}
		return value, nil
	}
	fallback := func(keys ...string) string {
		for _, key := range keys {
			if value := strings.TrimSpace(getenv(key)); value != "" {
				return value
			}
		}
		return ""
	}

	databaseURL, err := required("DATABASE_URL")
	if err != nil {
		return settings{}, err
	}
	openFGAURL, err := required("OPENFGA_API_URL")
	if err != nil {
		return settings{}, err
	}
	openFGAStoreID, err := required("OPENFGA_STORE_ID")
	if err != nil {
		return settings{}, err
	}
	openFGAModelID, err := required("OPENFGA_MODEL_ID")
	if err != nil {
		return settings{}, err
	}
	casdoorClientID, err := required("CASDOOR_BOOTSTRAP_CLIENT_ID")
	if err != nil {
		return settings{}, err
	}
	casdoorClientSecret, err := required("CASDOOR_BOOTSTRAP_CLIENT_SECRET")
	if err != nil {
		return settings{}, err
	}
	casdoorApplication, err := required("CASDOOR_BOOTSTRAP_APPLICATION")
	if err != nil {
		return settings{}, err
	}
	casdoorEndpoint := fallback("CASDOOR_CUTOVER_ENDPOINT", "CASDOOR_BOOTSTRAP_ENDPOINT", "CASDOOR_ISSUER")
	if casdoorEndpoint == "" {
		return settings{}, fmt.Errorf("CASDOOR_CUTOVER_ENDPOINT or CASDOOR_ISSUER is required")
	}
	casdoorOrganization := fallback("CASDOOR_BOOTSTRAP_ORGANIZATION", "CASDOOR_ORGANIZATION")
	if casdoorOrganization == "" {
		return settings{}, fmt.Errorf("CASDOOR_BOOTSTRAP_ORGANIZATION or CASDOOR_ORGANIZATION is required")
	}

	return settings{
		database: config.DatabaseConfig{
			URL:             databaseURL,
			MaxConns:        envInt32(getenv, "DB_MAX_CONNS", 4),
			MinConns:        envInt32(getenv, "DB_MIN_CONNS", 0),
			MaxConnLifetime: envInt(getenv, "DB_MAX_CONN_LIFETIME", 30),
			MaxConnIdleTime: envInt(getenv, "DB_MAX_CONN_IDLE_TIME", 5),
			QueryTimeout:    envInt(getenv, "DB_QUERY_TIMEOUT", 15),
			SSLMode:         fallback("DB_SSL_MODE"),
			SSLRootCert:     fallback("DB_SSL_ROOT_CERT"),
			SSLCert:         fallback("DB_SSL_CERT"),
			SSLKey:          fallback("DB_SSL_KEY"),
		},
		openFGA: config.OpenFGAConfig{
			APIUrl:               openFGAURL,
			StoreID:              openFGAStoreID,
			AuthorizationModelID: openFGAModelID,
		},
		casdoor: platformcasdoor.Credential{
			Purpose:      platformcasdoor.PurposeAuthorityCutover,
			Endpoint:     casdoorEndpoint,
			ClientID:     casdoorClientID,
			ClientSecret: casdoorClientSecret,
			Certificate:  getenv("CASDOOR_BOOTSTRAP_CERTIFICATE"),
			Organization: casdoorOrganization,
			Application:  casdoorApplication,
		},
	}, nil
}

func toLegacyAuthoritySnapshot(snapshot platformcasdoor.AuthoritySnapshot) authorization.LegacyAuthoritySnapshot {
	result := authorization.LegacyAuthoritySnapshot{
		Organization: snapshot.Organization,
		Users:        make([]authorization.LegacyAuthorityIdentity, 0, len(snapshot.Users)),
		RoleMembers:  make(map[authorization.Role][]string, len(snapshot.RoleMembers)),
	}
	for _, user := range snapshot.Users {
		result.Users = append(result.Users, authorization.LegacyAuthorityIdentity{
			ID:                 user.ID,
			Owner:              user.Owner,
			Name:               user.Name,
			OrganizationAdmin:  user.OrganizationAdmin,
			ForbiddenOrDeleted: user.ForbiddenOrDeleted,
		})
	}
	for roleName, members := range snapshot.RoleMembers {
		result.RoleMembers[authorization.Role(roleName)] = append([]string(nil), members...)
	}
	return result
}

func writeResult(output io.Writer, result authorization.AuthorityCutoverResult) error {
	return json.NewEncoder(output).Encode(safeResult{
		Changed:            result.Changed,
		SourceDigest:       result.SourceDigest,
		ImportedGrantCount: result.ImportedGrantCount,
		SkippedTupleCount:  result.SkippedTupleCount,
	})
}

func envInt(getenv func(string) string, key string, fallback int) int {
	raw := strings.TrimSpace(getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return fallback
	}
	return value
}

func envInt32(getenv func(string) string, key string, fallback int32) int32 {
	raw := strings.TrimSpace(getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || value < 0 {
		return fallback
	}
	// #nosec G115 -- ParseInt with bitSize 32 proves value is within int32 range.
	return int32(value)
}
