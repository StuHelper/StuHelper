// Command fga-setup bootstraps the OpenFGA authorization engine.
//
// Usage:
//
//	go run ./cmd/fga-setup
//
// Actions:
//  1. Creates an OpenFGA Store (if OPENFGA_STORE_ID is empty)
//  2. Imports the authorization model from infra/openfga/model.fga
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

	openfga "github.com/openfga/go-sdk"
	"github.com/openfga/go-sdk/client"
)

func main() {
	apiURL := envOrDefault("OPENFGA_API_URL", "http://localhost:8081")
	storeID := os.Getenv("OPENFGA_STORE_ID")
	modelPath := envOrDefault("FGA_MODEL_PATH", "infra/openfga/model.fga")

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
		log.Printf("Using existing store: %s", storeID)
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
	log.Printf("Importing model from %s...", modelPath)
	modelContent, err := os.ReadFile(modelPath)
	if err != nil {
		log.Fatalf("Failed to read model file: %v", err)
	}

	modelJSON, err := dslToJSON(string(modelContent))
	if err != nil {
		log.Fatalf("Failed to convert DSL to JSON: %v\nNote: if this fails, use the OpenFGA CLI instead:\n  openfga model write --store-id %s --file %s", err, storeID, modelPath)
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

// dslToJSON is a simplified DSL-to-JSON converter for the OpenFGA model.
// For production use, prefer the OpenFGA CLI: `openfga model write`
func dslToJSON(dsl string) (client.ClientWriteAuthorizationModelRequest, error) {
	// This is a placeholder — the OpenFGA Go SDK doesn't include DSL parsing.
	// The recommended approach is to use the OpenFGA CLI to write models.
	// For automated setup, we parse the JSON equivalent directly.
	_ = dsl

	// Return a hardcoded model matching infra/openfga/model.fga
	// This is kept in sync manually. When the model changes, update here too.
	model := buildModelFromCode()
	return model, nil
}

func buildModelFromCode() client.ClientWriteAuthorizationModelRequest {
	var req client.ClientWriteAuthorizationModelRequest
	modelJSON := `{
		"schema_version": "1.1",
		"type_definitions": [
			{"type": "user"},
			{
				"type": "ecosystem",
				"relations": {
					"super_admin": {"this": {}}
				},
				"metadata": {
					"relations": {
						"super_admin": {"directly_related_user_types": [{"type": "user"}]}
					}
				}
			},
			{
				"type": "school",
				"relations": {
					"parent": {"this": {}},
					"admin": {"this": {}},
					"reviewer": {"this": {}},
					"volunteer": {"this": {}},
					"effective_admin": {"union": {"child": [{"this": {}}, {"tupleToUserset": {"tupleset": {"relation": "parent"}, "computedUserset": {"relation": "super_admin"}}}]}}
				},
				"metadata": {
					"relations": {
						"parent": {"directly_related_user_types": [{"type": "ecosystem"}]},
						"admin": {"directly_related_user_types": [{"type": "user"}]},
						"reviewer": {"directly_related_user_types": [{"type": "user"}]},
						"volunteer": {"directly_related_user_types": [{"type": "user"}]},
						"effective_admin": {"directly_related_user_types": [{"type": "user"}]}
					}
				}
			},
			{
				"type": "course",
				"relations": {
					"school": {"this": {}},
					"owner": {"this": {}},
					"teaching_assistant": {"this": {}},
					"can_edit": {"union": {"child": [{"computedUserset": {"relation": "owner"}}, {"computedUserset": {"relation": "teaching_assistant"}}, {"tupleToUserset": {"tupleset": {"relation": "school"}, "computedUserset": {"relation": "effective_admin"}}}]}},
					"can_view": {"union": {"child": [{"computedUserset": {"relation": "owner"}}, {"computedUserset": {"relation": "teaching_assistant"}}, {"tupleToUserset": {"tupleset": {"relation": "school"}, "computedUserset": {"relation": "effective_admin"}}}]}}
				},
				"metadata": {
					"relations": {
						"school": {"directly_related_user_types": [{"type": "school"}]},
						"owner": {"directly_related_user_types": [{"type": "user"}]},
						"teaching_assistant": {"directly_related_user_types": [{"type": "user"}]}
					}
				}
			},
			{
				"type": "review",
				"relations": {
					"course": {"this": {}},
					"school": {"this": {}},
					"author": {"this": {}},
					"can_edit": {"computedUserset": {"relation": "author"}},
					"can_delete": {"union": {"child": [{"computedUserset": {"relation": "author"}}, {"tupleToUserset": {"tupleset": {"relation": "school"}, "computedUserset": {"relation": "volunteer"}}}, {"tupleToUserset": {"tupleset": {"relation": "school"}, "computedUserset": {"relation": "effective_admin"}}}]}},
					"can_hide": {"union": {"child": [{"tupleToUserset": {"tupleset": {"relation": "school"}, "computedUserset": {"relation": "volunteer"}}}, {"tupleToUserset": {"tupleset": {"relation": "school"}, "computedUserset": {"relation": "effective_admin"}}}]}},
					"can_view_author_identity": {"tupleToUserset": {"tupleset": {"relation": "school"}, "computedUserset": {"relation": "effective_admin"}}}
				},
				"metadata": {
					"relations": {
						"course": {"directly_related_user_types": [{"type": "course"}]},
						"school": {"directly_related_user_types": [{"type": "school"}]},
						"author": {"directly_related_user_types": [{"type": "user"}]}
					}
				}
			},
			{
				"type": "report",
				"relations": {
					"review": {"this": {}},
					"school": {"this": {}},
					"reporter": {"this": {}},
					"can_process": {"union": {"child": [{"tupleToUserset": {"tupleset": {"relation": "school"}, "computedUserset": {"relation": "volunteer"}}}, {"tupleToUserset": {"tupleset": {"relation": "school"}, "computedUserset": {"relation": "effective_admin"}}}]}}
				},
				"metadata": {
					"relations": {
						"review": {"directly_related_user_types": [{"type": "review"}]},
						"school": {"directly_related_user_types": [{"type": "school"}]},
						"reporter": {"directly_related_user_types": [{"type": "user"}]}
					}
				}
			},
			{
				"type": "user_profile",
				"relations": {
					"owner": {"this": {}},
					"school": {"this": {}},
					"can_view_own": {"computedUserset": {"relation": "owner"}},
					"can_view_identity": {"tupleToUserset": {"tupleset": {"relation": "school"}, "computedUserset": {"relation": "effective_admin"}}},
					"can_review_verification": {"tupleToUserset": {"tupleset": {"relation": "school"}, "computedUserset": {"relation": "effective_admin"}}}
				},
				"metadata": {
					"relations": {
						"owner": {"directly_related_user_types": [{"type": "user"}]},
						"school": {"directly_related_user_types": [{"type": "school"}]}
					}
				}
			}
		]
	}`
	if err := json.Unmarshal([]byte(modelJSON), &req); err != nil {
		log.Fatalf("Failed to parse hardcoded model JSON: %v", err)
	}
	return req
}

