package cache

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/redisfixture"
)

// setupTestRedis creates a miniredis instance and returns a connected client.
func setupTestRedis(t *testing.T) (*redis.Client, *redisfixture.Fixture) {
	t.Helper()
	fixture := redisfixture.Start(t)
	return fixture.Client, fixture
}

// ---------------------------------------------------------------------------
// NewHelper / NewHelperWithMaxVersions
// ---------------------------------------------------------------------------

func TestNewHelper(t *testing.T) {
	client, _ := setupTestRedis(t)
	h := NewHelper(client)

	assert.NotNil(t, h)
	assert.Equal(t, client, h.Client())
	assert.Equal(t, defaultMaxVersionEntries, h.maxVersionEntries)
}

func TestNewHelperWithMaxVersions(t *testing.T) {
	client, _ := setupTestRedis(t)

	t.Run("positive limit", func(t *testing.T) {
		h := NewHelperWithMaxVersions(client, 50)
		assert.Equal(t, 50, h.maxVersionEntries)
	})

	t.Run("zero falls back to default", func(t *testing.T) {
		h := NewHelperWithMaxVersions(client, 0)
		assert.Equal(t, defaultMaxVersionEntries, h.maxVersionEntries)
	})

	t.Run("negative falls back to default", func(t *testing.T) {
		h := NewHelperWithMaxVersions(client, -1)
		assert.Equal(t, defaultMaxVersionEntries, h.maxVersionEntries)
	})
}

// ---------------------------------------------------------------------------
// Set / GetAs round-trip
// ---------------------------------------------------------------------------

type testPayload struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func TestSetAndGetAs_RoundTrip(t *testing.T) {
	client, _ := setupTestRedis(t)
	h := NewHelper(client)
	ctx := context.Background()

	want := testPayload{Name: "hello", Count: 42}
	require.NoError(t, h.Set(ctx, "key:1", want, time.Minute))

	got, ok := GetAs[testPayload](h, ctx, "key:1")
	assert.True(t, ok)
	assert.Equal(t, want, got)
}

func TestGetAs_Miss(t *testing.T) {
	client, _ := setupTestRedis(t)
	h := NewHelper(client)
	ctx := context.Background()

	got, ok := GetAs[testPayload](h, ctx, "nonexistent")
	assert.False(t, ok)
	assert.Equal(t, testPayload{}, got)
}

func TestGetAs_UnmarshalError(t *testing.T) {
	client, _ := setupTestRedis(t)
	h := NewHelper(client)
	ctx := context.Background()

	// Store a string that cannot unmarshal into testPayload
	require.NoError(t, client.Set(ctx, "bad-json", "not-json", 0).Err())

	got, ok := GetAs[testPayload](h, ctx, "bad-json")
	assert.False(t, ok)
	assert.Equal(t, testPayload{}, got)
}

// ---------------------------------------------------------------------------
// Set / GetRaw round-trip
// ---------------------------------------------------------------------------

func TestSetAndGetRaw_RoundTrip(t *testing.T) {
	client, _ := setupTestRedis(t)
	h := NewHelper(client)
	ctx := context.Background()

	want := testPayload{Name: "raw", Count: 7}
	require.NoError(t, h.Set(ctx, "raw:1", want, time.Minute))

	raw, ok := h.GetRaw(ctx, "raw:1")
	assert.True(t, ok)

	var got testPayload
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Equal(t, want, got)
}

func TestGetRaw_Miss(t *testing.T) {
	client, _ := setupTestRedis(t)
	h := NewHelper(client)
	ctx := context.Background()

	raw, ok := h.GetRaw(ctx, "nonexistent")
	assert.False(t, ok)
	assert.Nil(t, raw)
}

// ---------------------------------------------------------------------------
// SetInt / GetInt
// ---------------------------------------------------------------------------

func TestSetIntAndGetInt_RoundTrip(t *testing.T) {
	client, _ := setupTestRedis(t)
	h := NewHelper(client)
	ctx := context.Background()

	require.NoError(t, h.SetInt(ctx, "count:1", 99, time.Minute))

	got, ok := h.GetInt(ctx, "count:1")
	assert.True(t, ok)
	assert.Equal(t, 99, got)
}

func TestGetInt_Miss(t *testing.T) {
	client, _ := setupTestRedis(t)
	h := NewHelper(client)
	ctx := context.Background()

	got, ok := h.GetInt(ctx, "no-int")
	assert.False(t, ok)
	assert.Equal(t, 0, got)
}

// ---------------------------------------------------------------------------
// TTL expiry (via miniredis FastForward)
// ---------------------------------------------------------------------------

func TestSet_TTLExpiry(t *testing.T) {
	client, fixture := setupTestRedis(t)
	h := NewHelper(client)
	ctx := context.Background()

	require.NoError(t, h.Set(ctx, "ttl:1", "data", 2*time.Second))

	// Immediately available
	_, ok := h.GetRaw(ctx, "ttl:1")
	assert.True(t, ok)

	// Fast-forward past TTL
	fixture.Server.FastForward(3 * time.Second)

	_, ok = h.GetRaw(ctx, "ttl:1")
	assert.False(t, ok)
}

// ---------------------------------------------------------------------------
// InvalidateByVersion / GetVersion / BuildVersionedKey
// ---------------------------------------------------------------------------

func TestVersionKey(t *testing.T) {
	assert.Equal(t, "cache:version:reviews", VersionKey("reviews"))
}

func TestGetVersion_DefaultsToZero(t *testing.T) {
	client, _ := setupTestRedis(t)
	h := NewHelper(client)
	ctx := context.Background()

	v := h.GetVersion(ctx, "unset-prefix")
	assert.Equal(t, "0", v)
}

func TestInvalidateByVersion_IncrementsVersion(t *testing.T) {
	client, _ := setupTestRedis(t)
	h := NewHelper(client)
	ctx := context.Background()

	v0 := h.GetVersion(ctx, "pf")
	assert.Equal(t, "0", v0)

	require.NoError(t, h.InvalidateByVersion(ctx, "pf"))

	// Local cache was cleared, next call fetches from Redis
	v1 := h.GetVersion(ctx, "pf")
	assert.Equal(t, "1", v1)

	require.NoError(t, h.InvalidateByVersion(ctx, "pf"))
	v2 := h.GetVersion(ctx, "pf")
	assert.Equal(t, "2", v2)
}

func TestBuildVersionedKey(t *testing.T) {
	client, _ := setupTestRedis(t)
	h := NewHelper(client)
	ctx := context.Background()

	// Before any invalidation the version is "0"
	key := h.BuildVersionedKey(ctx, "reviews", "course:123")
	assert.Equal(t, "reviews:v0:course:123", key)

	// After incrementing version the key changes
	require.NoError(t, h.InvalidateByVersion(ctx, "reviews"))
	key = h.BuildVersionedKey(ctx, "reviews", "course:123")
	assert.Equal(t, "reviews:v1:course:123", key)
}

// ---------------------------------------------------------------------------
// GetVersion local-cache eviction (maxVersionEntries)
// ---------------------------------------------------------------------------

func TestGetVersion_EvictsWhenOverLimit(t *testing.T) {
	client, _ := setupTestRedis(t)
	h := NewHelperWithMaxVersions(client, 3)
	ctx := context.Background()

	// Fill up to the limit
	for i := 0; i < 4; i++ {
		_ = h.GetVersion(ctx, "pfx"+string(rune('a'+i)))
	}

	h.vmu.RLock()
	count := len(h.versions)
	h.vmu.RUnlock()

	assert.LessOrEqual(t, count, 3, "local version cache should not exceed maxVersionEntries")
}

func TestGetVersion_OverflowKeepsNewestEntry(t *testing.T) {
	client, _ := setupTestRedis(t)
	h := NewHelperWithMaxVersions(client, 1)
	ctx := context.Background()

	require.NoError(t, client.Set(ctx, VersionKey("alpha"), "7", time.Minute).Err())
	require.NoError(t, client.Set(ctx, VersionKey("beta"), "9", time.Minute).Err())

	assert.Equal(t, "7", h.GetVersion(ctx, "alpha"))
	assert.Equal(t, "9", h.GetVersion(ctx, "beta"))

	h.vmu.RLock()
	defer h.vmu.RUnlock()
	assert.Len(t, h.versions, 1)
	assert.Equal(t, "9", h.versions[VersionKey("beta")].version)
}

// ---------------------------------------------------------------------------
// GetOrSet
// ---------------------------------------------------------------------------

func TestGetOrSet_LoaderCalledOnMiss(t *testing.T) {
	client, _ := setupTestRedis(t)
	h := NewHelper(client)
	ctx := context.Background()

	calls := 0
	loader := func(_ context.Context) (testPayload, error) {
		calls++
		return testPayload{Name: "loaded", Count: 1}, nil
	}

	got, err := GetOrSet(h, ctx, "gos:1", time.Minute, loader)
	require.NoError(t, err)
	assert.Equal(t, testPayload{Name: "loaded", Count: 1}, got)
	assert.Equal(t, 1, calls)

	// Second call should hit cache, loader not called again
	got, err = GetOrSet(h, ctx, "gos:1", time.Minute, loader)
	require.NoError(t, err)
	assert.Equal(t, testPayload{Name: "loaded", Count: 1}, got)
	assert.Equal(t, 1, calls) // still 1
}

func TestGetOrSet_LoaderError(t *testing.T) {
	client, _ := setupTestRedis(t)
	h := NewHelper(client)
	ctx := context.Background()

	wantErr := errors.New("db down")
	loader := func(_ context.Context) (testPayload, error) {
		return testPayload{}, wantErr
	}

	_, err := GetOrSet(h, ctx, "gos:err", time.Minute, loader)
	assert.ErrorIs(t, err, wantErr)
}

// ---------------------------------------------------------------------------
// JitteredTTL
// ---------------------------------------------------------------------------

func TestJitteredTTL_StaysWithinBounds(t *testing.T) {
	base := 10 * time.Second
	lo := time.Duration(float64(base) * (1 - jitterFraction))
	hi := time.Duration(float64(base) * (1 + jitterFraction))

	for i := 0; i < 200; i++ {
		got := JitteredTTL(base)
		assert.GreaterOrEqual(t, got, lo, "jittered TTL below lower bound")
		assert.LessOrEqual(t, got, hi, "jittered TTL above upper bound")
	}
}

// ---------------------------------------------------------------------------
// Nil-client safety (all methods should be no-ops)
// ---------------------------------------------------------------------------

func TestNilClient_NoOps(t *testing.T) {
	h := NewHelper(nil)
	ctx := context.Background()

	// Set is a no-op
	assert.NoError(t, h.Set(ctx, "k", "v", time.Minute))

	// GetRaw returns miss
	_, ok := h.GetRaw(ctx, "k")
	assert.False(t, ok)

	// GetAs returns miss
	_, ok = GetAs[string](h, ctx, "k")
	assert.False(t, ok)

	// GetInt returns miss
	_, ok = h.GetInt(ctx, "k")
	assert.False(t, ok)

	// SetInt is a no-op
	assert.NoError(t, h.SetInt(ctx, "k", 1, time.Minute))

	// InvalidateByVersion is a no-op
	assert.NoError(t, h.InvalidateByVersion(ctx, "prefix"))

	// GetVersion returns "0"
	assert.Equal(t, "0", h.GetVersion(ctx, "prefix"))
}
