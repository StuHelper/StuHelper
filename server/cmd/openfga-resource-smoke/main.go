// Command openfga-resource-smoke verifies Open Platform app-to-resource tuples
// against a configured OpenFGA store/model.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/fga"
)

const (
	relationReadByApp  = "can_read_by_app"
	relationWriteByApp = "can_write_by_app"
)

type resourceFGAClient interface {
	Check(ctx context.Context, user, relation, object string) (bool, error)
	WriteMissingTuples(ctx context.Context, desired []fga.Tuple) error
	DeleteTuples(ctx context.Context, tuples []fga.Tuple) error
	ListObjects(ctx context.Context, user, relation, objectType string) ([]string, error)
}

type smokeConfig struct {
	APIURL     string `json:"apiURL"`
	AppID      string `json:"appID"`
	ModelID    string `json:"modelID"`
	ResourceID string `json:"resourceID"`
	StoreID    string `json:"storeID"`
}

type smokeEvidence struct {
	APIURL                string `json:"apiURL"`
	AppObject             string `json:"appObject"`
	ListedReadAfterRevoke bool   `json:"listedReadAfterRevoke"`
	ListedReadGrant       bool   `json:"listedReadGrant"`
	ModelID               string `json:"modelID"`
	ReadAfterGrant        bool   `json:"readAfterGrant"`
	ReadAfterRevoke       bool   `json:"readAfterRevoke"`
	ResourceObject        string `json:"resourceObject"`
	StoreID               string `json:"storeID"`
	WriteAfterGrant       bool   `json:"writeAfterGrant"`
	WriteAfterRevoke      bool   `json:"writeAfterRevoke"`
}

func main() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Printf("warning: failed to load .env: %v", err)
	}

	cfg, err := smokeConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	client, err := fga.NewClient(config.OpenFGAConfig{
		APIUrl:               cfg.APIURL,
		StoreID:              cfg.StoreID,
		AuthorizationModelID: cfg.ModelID,
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), envDuration("OPENFGA_RESOURCE_SMOKE_TIMEOUT", 20*time.Second))
	defer cancel()

	evidence, err := runSmoke(ctx, client, cfg)
	if err != nil {
		log.Fatal(err)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(evidence); err != nil {
		log.Fatal(err)
	}
}

func smokeConfigFromEnv() (smokeConfig, error) {
	now := strconv.FormatInt(time.Now().UnixNano(), 10)
	cfg := smokeConfig{
		APIURL:     envOrDefault("OPENFGA_API_URL", "http://localhost:8081"),
		AppID:      envOrDefault("OPENFGA_RESOURCE_SMOKE_APP_ID", "smoke"+now),
		ModelID:    os.Getenv("OPENFGA_MODEL_ID"),
		ResourceID: envOrDefault("OPENFGA_RESOURCE_SMOKE_RESOURCE_ID", "resource"+now),
		StoreID:    os.Getenv("OPENFGA_STORE_ID"),
	}
	var missing []string
	if cfg.StoreID == "" {
		missing = append(missing, "OPENFGA_STORE_ID")
	}
	if cfg.ModelID == "" {
		missing = append(missing, "OPENFGA_MODEL_ID")
	}
	if len(missing) > 0 {
		return smokeConfig{}, fmt.Errorf("missing required env: %s", strings.Join(missing, ", "))
	}
	return cfg, nil
}

func runSmoke(ctx context.Context, client resourceFGAClient, cfg smokeConfig) (smokeEvidence, error) {
	app := "open_platform_app:" + cfg.AppID
	resource := "resource_item:" + cfg.ResourceID
	readTuple := fga.Tuple{User: app, Relation: relationReadByApp, Object: resource}
	writeTuple := fga.Tuple{User: app, Relation: relationWriteByApp, Object: resource}

	if err := client.WriteMissingTuples(ctx, []fga.Tuple{readTuple, writeTuple}); err != nil {
		return smokeEvidence{}, fmt.Errorf("grant resource tuples: %w", err)
	}

	readAllowed, err := client.Check(ctx, app, relationReadByApp, resource)
	if err != nil {
		return smokeEvidence{}, fmt.Errorf("check read grant: %w", err)
	}
	writeAllowed, err := client.Check(ctx, app, relationWriteByApp, resource)
	if err != nil {
		return smokeEvidence{}, fmt.Errorf("check write grant: %w", err)
	}
	readObjects, err := client.ListObjects(ctx, app, relationReadByApp, "resource_item")
	if err != nil {
		return smokeEvidence{}, fmt.Errorf("list read grants: %w", err)
	}

	if err := client.DeleteTuples(ctx, []fga.Tuple{readTuple}); err != nil {
		return smokeEvidence{}, fmt.Errorf("revoke read grant: %w", err)
	}
	readAfterRevoke, err := client.Check(ctx, app, relationReadByApp, resource)
	if err != nil {
		return smokeEvidence{}, fmt.Errorf("check read after revoke: %w", err)
	}
	readObjectsAfterRevoke, err := client.ListObjects(ctx, app, relationReadByApp, "resource_item")
	if err != nil {
		return smokeEvidence{}, fmt.Errorf("list read grants after revoke: %w", err)
	}
	writeStillAllowed, err := client.Check(ctx, app, relationWriteByApp, resource)
	if err != nil {
		return smokeEvidence{}, fmt.Errorf("check write after read revoke: %w", err)
	}

	if err := client.DeleteTuples(ctx, []fga.Tuple{writeTuple}); err != nil {
		return smokeEvidence{}, fmt.Errorf("revoke write grant: %w", err)
	}
	writeAfterRevoke, err := client.Check(ctx, app, relationWriteByApp, resource)
	if err != nil {
		return smokeEvidence{}, fmt.Errorf("check write after revoke: %w", err)
	}

	evidence := smokeEvidence{
		APIURL:                cfg.APIURL,
		AppObject:             app,
		ListedReadAfterRevoke: containsString(readObjectsAfterRevoke, resource),
		ListedReadGrant:       containsString(readObjects, resource),
		ModelID:               cfg.ModelID,
		ReadAfterGrant:        readAllowed,
		ReadAfterRevoke:       readAfterRevoke,
		ResourceObject:        resource,
		StoreID:               cfg.StoreID,
		WriteAfterGrant:       writeAllowed && writeStillAllowed,
		WriteAfterRevoke:      writeAfterRevoke,
	}
	if !evidence.ReadAfterGrant || !evidence.WriteAfterGrant || !evidence.ListedReadGrant {
		return evidence, fmt.Errorf("resource tuple grant did not become effective: %+v", evidence)
	}
	if evidence.ReadAfterRevoke || evidence.WriteAfterRevoke || evidence.ListedReadAfterRevoke {
		return evidence, fmt.Errorf("resource tuple revoke did not become effective: %+v", evidence)
	}
	return evidence, nil
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		log.Printf("warning: invalid %s=%q, using %s", key, value, fallback)
		return fallback
	}
	return parsed
}
