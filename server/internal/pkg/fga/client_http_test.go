package fga

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	openfga "github.com/openfga/go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/config"
)

func TestClient_HTTPWrappers(t *testing.T) {
	var mu sync.Mutex
	var writes []map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/check"):
			require.Equal(t, http.MethodPost, r.Method)
			_ = json.NewEncoder(w).Encode(openfga.CheckResponse{Allowed: openfga.PtrBool(true)})
		case strings.HasSuffix(r.URL.Path, "/read"):
			require.Equal(t, http.MethodPost, r.Method)
			_ = json.NewEncoder(w).Encode(openfga.ReadResponse{Tuples: []openfga.Tuple{{Key: openfga.TupleKey{User: "user:1", Relation: "author", Object: "review:1"}, Timestamp: time.Now()}}, ContinuationToken: ""})
		case strings.HasSuffix(r.URL.Path, "/list-objects"):
			require.Equal(t, http.MethodPost, r.Method)
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, "user:1", body["user"])
			assert.Equal(t, "effective_admin", body["relation"])
			assert.Equal(t, "school", body["type"])
			_ = json.NewEncoder(w).Encode(openfga.ListObjectsResponse{Objects: []string{"school:4111010002", "school:4111010001"}})
		case strings.HasSuffix(r.URL.Path, "/write"):
			require.Equal(t, http.MethodPost, r.Method)
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			mu.Lock()
			writes = append(writes, body)
			mu.Unlock()
			_, _ = w.Write([]byte(`{}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(config.OpenFGAConfig{APIUrl: server.URL, StoreID: "01ARZ3NDEKTSV4RRFFQ69G5FAV", AuthorizationModelID: "01ARZ3NDEKTSV4RRFFQ69G5FAV"})
	require.NoError(t, err)
	require.NotNil(t, client)

	allowed, err := client.Check(context.Background(), "user:1", "viewer", "review:1")
	require.NoError(t, err)
	assert.True(t, allowed)

	tuples, err := client.ReadTuples(context.Background(), "review:1", "author")
	require.NoError(t, err)
	require.Len(t, tuples, 1)
	assert.Equal(t, Tuple{User: "user:1", Relation: "author", Object: "review:1"}, tuples[0])

	objects, err := client.ListObjects(context.Background(), "user:1", "effective_admin", "school")
	require.NoError(t, err)
	assert.Equal(t, []string{"school:4111010001", "school:4111010002"}, objects)

	require.NoError(t, client.WriteReviewRelations(context.Background(), "r1", "u1", "s1"))
	require.NoError(t, client.WriteReviewRelations(context.Background(), "legacy", "", "s1"))
	require.NoError(t, client.WriteReportRelations(context.Background(), "rep1", "s1"))
	require.NoError(t, client.DeleteTuples(context.Background(), []Tuple{{User: "user:u1", Relation: "author", Object: "review:r1"}}))
	require.NoError(t, client.DeleteTuplesIgnoringMissing(context.Background(), []Tuple{{User: "user:u2", Relation: "author", Object: "review:r2"}}))

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, writes, 5)
	assert.Contains(t, writes[0], "writes")
	assert.Contains(t, writes[1], "writes")
	assert.Contains(t, writes[2], "writes")
	assert.Contains(t, writes[3], "deletes")
	assert.Contains(t, writes[4], "deletes")

	firstJSON, err := json.Marshal(writes[0])
	require.NoError(t, err)
	assert.Contains(t, string(firstJSON), "user:u1")
	assert.Contains(t, string(firstJSON), "author")
	assert.Contains(t, string(firstJSON), "review:r1")
	assert.Contains(t, string(firstJSON), "section:school_s1_review_moderation")

	legacyJSON, err := json.Marshal(writes[1])
	require.NoError(t, err)
	assert.NotContains(t, string(legacyJSON), `"relation":"author"`)
	assert.NotContains(t, string(legacyJSON), "user:u")
	assert.Contains(t, string(legacyJSON), "school:s1")
	assert.Contains(t, string(legacyJSON), "section:school_s1_review_moderation")

	deleteJSON, err := json.Marshal(writes[3])
	require.NoError(t, err)
	assert.Contains(t, string(deleteJSON), "deletes")

	idempotentDeleteJSON, err := json.Marshal(writes[4])
	require.NoError(t, err)
	assert.Contains(t, string(idempotentDeleteJSON), `"on_missing":"ignore"`)
}

func TestTupleExistsUsesExactDirectTupleRead(t *testing.T) {
	var requestBodies []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.True(t, strings.HasSuffix(r.URL.Path, "/read"))
		require.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")

		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "HIGHER_CONSISTENCY", body["consistency"])
		tupleKey, ok := body["tuple_key"].(map[string]any)
		require.True(t, ok)
		requestBodies = append(requestBodies, tupleKey)

		responseBody := openfga.ReadResponse{ContinuationToken: ""}
		if tupleKey["user"] == "user:42" {
			responseBody.Tuples = []openfga.Tuple{{
				Key: openfga.TupleKey{
					User:     "user:42",
					Relation: "super_admin",
					Object:   "ecosystem:stuhelper",
				},
				Timestamp: time.Now(),
			}}
		}
		require.NoError(t, json.NewEncoder(w).Encode(responseBody))
	}))
	defer server.Close()

	client, err := NewClient(config.OpenFGAConfig{
		APIUrl:               server.URL,
		StoreID:              "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		AuthorizationModelID: "01ARZ3NDEKTSV4RRFFQ69G5FAV",
	})
	require.NoError(t, err)

	tuple := Tuple{User: "user:42", Relation: "super_admin", Object: "ecosystem:stuhelper"}
	exists, err := client.TupleExists(t.Context(), tuple)
	require.NoError(t, err)
	assert.True(t, exists)

	tuple.User = "user:43"
	exists, err = client.TupleExists(t.Context(), tuple)
	require.NoError(t, err)
	assert.False(t, exists)

	require.Len(t, requestBodies, 2)
	assert.Equal(t, map[string]any{
		"user":     "user:42",
		"relation": "super_admin",
		"object":   "ecosystem:stuhelper",
	}, requestBodies[0])
	assert.Equal(t, "user:43", requestBodies[1]["user"])
}

func TestReadTuplesFollowsContinuationTokensWithHigherConsistency(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.True(t, strings.HasSuffix(r.URL.Path, "/read"))
		require.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "HIGHER_CONSISTENCY", body["consistency"])
		requestCount++
		switch requestCount {
		case 1:
			assert.Empty(t, body["continuation_token"])
			require.NoError(t, json.NewEncoder(w).Encode(openfga.ReadResponse{
				Tuples: []openfga.Tuple{{Key: openfga.TupleKey{
					User: "user:1", Relation: "admin", Object: "school:1",
				}}},
				ContinuationToken: "next-page",
			}))
		case 2:
			assert.Equal(t, "next-page", body["continuation_token"])
			require.NoError(t, json.NewEncoder(w).Encode(openfga.ReadResponse{
				Tuples: []openfga.Tuple{{Key: openfga.TupleKey{
					User: "user:2", Relation: "admin", Object: "school:1",
				}}},
				ContinuationToken: "",
			}))
		default:
			t.Fatalf("unexpected read request %d", requestCount)
		}
	}))
	defer server.Close()

	client, err := NewClient(config.OpenFGAConfig{
		APIUrl:               server.URL,
		StoreID:              "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		AuthorizationModelID: "01ARZ3NDEKTSV4RRFFQ69G5FAV",
	})
	require.NoError(t, err)

	tuples, err := client.ReadTuples(t.Context(), "school:1", "admin")

	require.NoError(t, err)
	assert.Equal(t, []Tuple{
		{User: "user:1", Relation: "admin", Object: "school:1"},
		{User: "user:2", Relation: "admin", Object: "school:1"},
	}, tuples)
	assert.Equal(t, 2, requestCount)
}

func TestRecordSpanError_NoPanicOnNil(t *testing.T) {
	assert.NotPanics(t, func() { recordSpanError(nil, nil) })
}
