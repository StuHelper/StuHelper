package resource

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/db"
	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
)

func TestRepositoryBatchAssociationLoadingUsesFixedQueriesAndOneConnection(t *testing.T) {
	fixture := postgresfixture.Start(t)
	ctx := context.Background()

	var mountID int64
	require.NoError(t, fixture.DB.QueryRow(ctx, `
		SELECT id
		FROM storage_mounts
		WHERE key = 'default-s3'
	`).Scan(&mountID))

	seedService := NewService(NewRepository(fixture.DB), &fakeObjectStore{mountID: mountID})
	withAssociations, err := seedService.CreateResource(ctx, "batch-owner", CreateRequest{
		Title:      "With associations",
		Visibility: "public",
		Tags:       []string{"zeta", "alpha"},
		Bindings: []Binding{
			{Type: "term", Value: "2026-SPRING"},
			{Type: "course", Value: "CS101"},
			{Type: "course", Value: "CS001"},
		},
		Filename:    "with-associations.txt",
		ContentType: "text/plain",
		DataBase64:  "d2l0aCBhc3NvY2lhdGlvbnM=",
	})
	require.NoError(t, err)
	withoutAssociations, err := seedService.CreateResource(ctx, "batch-owner", CreateRequest{
		Title:       "Without associations",
		Visibility:  "public",
		Filename:    "without-associations.txt",
		ContentType: "text/plain",
		DataBase64:  "d2l0aG91dCBhc3NvY2lhdGlvbnM=",
	})
	require.NoError(t, err)

	poolConfig, err := pgxpool.ParseConfig(fixture.URL)
	require.NoError(t, err)
	poolConfig.MaxConns = 1
	poolConfig.MinConns = 0
	singlePool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	require.NoError(t, err)
	singleDB := db.NewDB(singlePool, 2*time.Second)
	t.Cleanup(singleDB.Close)
	require.NoError(t, singleDB.Ping(ctx))

	repo := NewRepository(singleDB)
	assertAcquireDelta := func(want int64, run func()) {
		t.Helper()
		before := singlePool.Stat().AcquireCount()
		run()
		assert.Equal(t, want, singlePool.Stat().AcquireCount()-before)
	}

	assertAcquireDelta(3, func() {
		items, total, listErr := repo.ListResources(ctx, ListFilters{Page: 1, PageSize: 1})
		require.NoError(t, listErr)
		assert.Equal(t, 2, total)
		require.Len(t, items, 1)
	})

	var listed []Item
	assertAcquireDelta(3, func() {
		var listErr error
		listed, _, listErr = repo.ListResources(ctx, ListFilters{Page: 1, PageSize: 20})
		require.NoError(t, listErr)
		require.Len(t, listed, 2)
	})

	byID := make(map[int64]Item, len(listed))
	for _, item := range listed {
		byID[item.ID] = item
	}
	assert.Equal(t, []string{"alpha", "zeta"}, byID[withAssociations.ID].Tags)
	assert.Equal(t, []Binding{
		{Type: "course", Value: "CS001"},
		{Type: "course", Value: "CS101"},
		{Type: "term", Value: "2026-SPRING"},
	}, byID[withAssociations.ID].Bindings)
	assert.NotNil(t, byID[withoutAssociations.ID].Tags)
	assert.Empty(t, byID[withoutAssociations.ID].Tags)
	assert.NotNil(t, byID[withoutAssociations.ID].Bindings)
	assert.Empty(t, byID[withoutAssociations.ID].Bindings)

	assertAcquireDelta(3, func() {
		item, getErr := repo.GetResourceByID(ctx, withAssociations.ID)
		require.NoError(t, getErr)
		assert.Equal(t, withAssociations.ID, item.ID)
		assert.Equal(t, []string{"alpha", "zeta"}, item.Tags)
	})

	const workers = 6
	concurrentCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	start := make(chan struct{})
	results := make(chan error, workers)
	for range workers {
		go func() {
			<-start
			items, total, listErr := repo.ListResources(concurrentCtx, ListFilters{Page: 1, PageSize: 20})
			if listErr != nil {
				results <- listErr
				return
			}
			if total != 2 || len(items) != 2 {
				results <- fmt.Errorf("unexpected list result: total=%d items=%d", total, len(items))
				return
			}
			results <- nil
		}()
	}
	close(start)
	for range workers {
		require.NoError(t, <-results)
	}
}
