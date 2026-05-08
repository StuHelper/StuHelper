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

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
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
			_ = json.NewEncoder(w).Encode(openfga.ListObjectsResponse{Objects: []string{"school:1002", "school:1001"}})
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
	assert.Equal(t, []string{"school:1001", "school:1002"}, objects)

	require.NoError(t, client.WriteReviewRelations(context.Background(), "r1", "u1", "c1", "s1"))
	require.NoError(t, client.WriteReportRelations(context.Background(), "rep1", "u2", "r1", "s1"))
	require.NoError(t, client.DeleteTuples(context.Background(), []Tuple{{User: "user:u1", Relation: "author", Object: "review:r1"}}))

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, writes, 4)
	assert.Contains(t, writes[0], "writes")
	assert.Contains(t, writes[1], "writes")
	assert.Contains(t, writes[2], "writes")
	assert.Contains(t, writes[3], "deletes")

	firstJSON, err := json.Marshal(writes[0])
	require.NoError(t, err)
	assert.Contains(t, string(firstJSON), "school:s1")
	assert.Contains(t, string(firstJSON), "school")
	assert.Contains(t, string(firstJSON), "section:school_s1_review_moderation")

	secondJSON, err := json.Marshal(writes[1])
	require.NoError(t, err)
	assert.Contains(t, string(secondJSON), "user:u1")
	assert.Contains(t, string(secondJSON), "author")
	assert.Contains(t, string(secondJSON), "review:r1")
	assert.Contains(t, string(secondJSON), "section:school_s1_review_moderation")
}

func TestRecordSpanError_NoPanicOnNil(t *testing.T) {
	assert.NotPanics(t, func() { recordSpanError(nil, nil) })
}
