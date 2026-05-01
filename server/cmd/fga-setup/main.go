// Command fga-setup bootstraps the OpenFGA authorization engine.
//
// Usage:
//
//	go run ./cmd/fga-setup
//
// Actions:
//  1. Creates an OpenFGA Store (if OPENFGA_STORE_ID is empty)
//  2. Imports the authorization model from infra/openfga/model.json
//  3. Writes initial ecosystem and school tuples
//  4. Prints StoreID and ModelID for .env configuration
//
// Prerequisites: OpenFGA must be running (docker compose up openfga).
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	openfga "github.com/openfga/go-sdk"
	"github.com/openfga/go-sdk/client"
)

func main() {
	apiURL := envOrDefault("OPENFGA_API_URL", "http://localhost:8081")
	storeID := os.Getenv("OPENFGA_STORE_ID")
	modelPath := envOrDefault("FGA_MODEL_PATH", "../infra/openfga/model.fga")
	resolvedModelPath, err := resolveModelPath(modelPath)
	if err != nil {
		log.Fatalf("Invalid model path: %v", err)
	}

	ctx := context.Background()

	// 1. Create SDK client (without store initially)
	fgaClient, err := client.NewSdkClient(&client.ClientConfiguration{
		ApiUrl: apiURL,
	})
	if err != nil {
		log.Fatalf("Failed to create FGA client: %v", err)
	}

	// 2. Create or use existing store
	if storeID == "" {
		log.Println("Creating new OpenFGA store...")
		resp, err := fgaClient.CreateStore(ctx).Body(client.ClientCreateStoreRequest{
			Name: "stuhelper",
		}).Execute()
		if err != nil {
			log.Fatalf("Failed to create store: %v", err)
		}
		storeID = resp.Id
		log.Printf("Store created: %s", storeID)
	} else {
		log.Println("Using existing OpenFGA store from environment")
	}

	// Re-create client with store ID
	fgaClient, err = client.NewSdkClient(&client.ClientConfiguration{
		ApiUrl:  apiURL,
		StoreId: storeID,
	})
	if err != nil {
		log.Fatalf("Failed to re-create FGA client: %v", err)
	}

	// 3. Read and import model
	log.Printf("Importing model from %s...", resolvedModelPath)
	modelJSON, err := loadAuthorizationModel(resolvedModelPath)
	if err != nil {
		log.Fatalf("Failed to load authorization model: %v", err)
	}

	writeResp, err := fgaClient.WriteAuthorizationModel(ctx).Body(modelJSON).Execute()
	if err != nil {
		log.Fatalf("Failed to write authorization model: %v", err)
	}
	modelID := writeResp.AuthorizationModelId
	log.Printf("Model imported: %s", modelID)

	// Update client with model ID
	fgaClient, err = client.NewSdkClient(&client.ClientConfiguration{
		ApiUrl:               apiURL,
		StoreId:              storeID,
		AuthorizationModelId: modelID,
	})
	if err != nil {
		log.Fatalf("Failed to re-create FGA client with model: %v", err)
	}

	// 4. Write initial tuples
	log.Println("Writing initial tuples...")
	tuples := []openfga.TupleKey{
		// 学校 → 生态关系
		{User: "ecosystem:stuhelper", Relation: "parent", Object: "school:1"},
	}

	_, err = fgaClient.Write(ctx).Body(client.ClientWriteRequest{
		Writes: tuples,
	}).Execute()
	if err != nil {
		// 可能已存在，记日志但不退出
		log.Printf("Warning: initial tuple write: %v (may already exist)", err)
	} else {
		log.Printf("Written %d initial tuples", len(tuples))
	}

	// 5. Print env config
	fmt.Println()
	fmt.Println("=== Add to your .env file ===")
	fmt.Printf("OPENFGA_STORE_ID=%s\n", storeID)
	fmt.Printf("OPENFGA_MODEL_ID=%s\n", modelID)
	fmt.Println("=============================")
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func resolveModelPath(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", fmt.Errorf("model path is empty")
	}

	workingDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}

	absolutePath, err := filepath.Abs(filepath.Clean(raw))
	if err != nil {
		return "", fmt.Errorf("resolve absolute path: %w", err)
	}

	modelRoot := modelWorkspaceRoot(workingDir)
	relativePath, err := filepath.Rel(modelRoot, absolutePath)
	if err != nil {
		return "", fmt.Errorf("resolve relative path: %w", err)
	}
	if relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("model path %q escapes workspace root %q", raw, modelRoot)
	}

	return absolutePath, nil
}

func modelWorkspaceRoot(workingDir string) string {
	if filepath.Base(workingDir) == "server" {
		return filepath.Dir(workingDir)
	}
	return workingDir
}

func loadAuthorizationModel(modelPath string) (client.ClientWriteAuthorizationModelRequest, error) {
	jsonPath, err := authorizationModelJSONPath(modelPath)
	if err != nil {
		return client.ClientWriteAuthorizationModelRequest{}, err
	}

	// #nosec G304 -- resolveModelPath constrains the base file to the current workspace;
	// authorizationModelJSONPath only switches to a sibling .json file.
	content, err := os.ReadFile(jsonPath)
	if err != nil {
		return client.ClientWriteAuthorizationModelRequest{}, fmt.Errorf("read model JSON: %w", err)
	}
	return parseAuthorizationModelJSON(content)
}

func authorizationModelJSONPath(modelPath string) (string, error) {
	switch ext := filepath.Ext(modelPath); ext {
	case ".json":
		return modelPath, nil
	case ".fga":
		jsonPath := strings.TrimSuffix(modelPath, ext) + ".json"
		if _, err := os.Stat(jsonPath); err != nil {
			return "", fmt.Errorf("OpenFGA Go setup requires JSON companion %q for DSL file %q: %w", jsonPath, modelPath, err)
		}
		return jsonPath, nil
	default:
		return "", fmt.Errorf("unsupported OpenFGA model extension %q; use .fga with a .json companion or pass .json", ext)
	}
}

func parseAuthorizationModelJSON(content []byte) (client.ClientWriteAuthorizationModelRequest, error) {
	var req client.ClientWriteAuthorizationModelRequest
	if err := json.Unmarshal(content, &req); err != nil {
		return client.ClientWriteAuthorizationModelRequest{}, fmt.Errorf("parse model JSON: %w", err)
	}
	if req.SchemaVersion == "" || len(req.TypeDefinitions) == 0 {
		return client.ClientWriteAuthorizationModelRequest{}, fmt.Errorf("model JSON must include schema_version and type_definitions")
	}
	return req, nil
}
