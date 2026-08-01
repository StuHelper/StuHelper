package review

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/reviewaccess"
	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
)

func TestSensitiveWordMutationsInvalidateWarmFilter(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(
		fixture.DB,
		repo,
		noopReviewSender2{},
		&recordingReviewFGAWriter{},
		fakeAccessReader{
			schools: []reviewaccess.SchoolConfig{},
		},
	)
	ctx := context.Background()

	require.NoError(t, svc.filter.Refresh(ctx))
	require.False(t, svc.filter.lastRefresh.IsZero())

	word, err := svc.CreateSensitiveWord(ctx, "立即拦截词", "custom", "block")
	require.NoError(t, err)
	assert.True(t, svc.filter.lastRefresh.IsZero())

	createdResult, err := svc.CheckContent(ctx, "这条内容含有立即拦截词")
	require.NoError(t, err)
	require.NotNil(t, createdResult)
	assert.False(t, createdResult.IsValid)
	assert.Equal(t, "block", createdResult.Level)

	updatedWord := "立即警告词"
	updatedLevel := ContentFlagWarn
	require.NoError(t, svc.UpdateSensitiveWord(
		ctx,
		word.ID,
		&updatedWord,
		nil,
		&updatedLevel,
		nil,
	))
	assert.True(t, svc.filter.lastRefresh.IsZero())

	oldResult, err := svc.CheckContent(ctx, "旧的立即拦截词不应继续命中")
	require.NoError(t, err)
	require.NotNil(t, oldResult)
	assert.True(t, oldResult.IsValid)
	assert.Empty(t, oldResult.Level)

	updatedResult, err := svc.CheckContent(ctx, "新的立即警告词应立即命中")
	require.NoError(t, err)
	require.NotNil(t, updatedResult)
	assert.True(t, updatedResult.IsValid)
	assert.Equal(t, ContentFlagWarn, updatedResult.Level)

	require.NoError(t, svc.DeleteSensitiveWord(ctx, word.ID))
	assert.True(t, svc.filter.lastRefresh.IsZero())

	deletedResult, err := svc.CheckContent(ctx, "删除后的立即警告词不应命中")
	require.NoError(t, err)
	require.NotNil(t, deletedResult)
	assert.True(t, deletedResult.IsValid)
	assert.Empty(t, deletedResult.Level)
}

func TestSensitiveWordMutationFailureKeepsWarmFilter(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(
		fixture.DB,
		repo,
		noopReviewSender2{},
		&recordingReviewFGAWriter{},
		fakeAccessReader{},
	)
	ctx := context.Background()

	require.NoError(t, svc.filter.Refresh(ctx))
	lastRefresh := svc.filter.lastRefresh
	inactive := false

	err := svc.UpdateSensitiveWord(ctx, "missing-sensitive-word", nil, nil, nil, &inactive)
	require.Error(t, err)
	assert.Equal(t, lastRefresh, svc.filter.lastRefresh)

	err = svc.DeleteSensitiveWord(ctx, "missing-sensitive-word")
	require.Error(t, err)
	assert.Equal(t, lastRefresh, svc.filter.lastRefresh)
}

func TestInvalidatedSensitiveWordFilterFailsClosedWhenReloadFails(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	filter := NewFilter(repo)
	ctx := context.Background()

	require.NoError(t, filter.Refresh(ctx))
	filter.Invalidate()
	fixture.DB.Close()

	result, err := filter.CheckContent(ctx, "content after dependency failure")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrModerationUnavailable)
	assert.Nil(t, result)
	assert.True(t, filter.lastRefresh.IsZero())
}
