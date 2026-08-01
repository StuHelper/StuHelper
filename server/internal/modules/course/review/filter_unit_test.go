package review

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seededFilter(words []SensitiveWord) *Filter {
	f := &Filter{}
	f.applyWords(words)
	f.refreshTTL = time.Hour
	return f
}

func TestNewFilterRequiresRepo(t *testing.T) {
	assert.PanicsWithValue(t, "review.NewFilter: repo must not be nil", func() {
		NewFilter(nil)
	})
}

func TestReviewFilterInvalidateMarksWarmSnapshotStale(t *testing.T) {
	f := seededFilter([]SensitiveWord{{Word: "danger", Level: "block"}})
	require.False(t, f.lastRefresh.IsZero())

	f.Invalidate()

	assert.True(t, f.lastRefresh.IsZero())
	assert.Len(t, f.blockMatchers, 1, "invalidation should not erase the last known snapshot before reload")
}

func TestReviewFilterInvalidateSerializesWithRefresh(t *testing.T) {
	f := seededFilter([]SensitiveWord{{Word: "danger", Level: "block"}})
	started := make(chan struct{})
	done := make(chan struct{})

	f.refreshMu.Lock()
	go func() {
		close(started)
		f.Invalidate()
		close(done)
	}()
	<-started

	completedWhileRefreshLocked := false
	select {
	case <-done:
		completedWhileRefreshLocked = true
	case <-time.After(50 * time.Millisecond):
	}
	f.refreshMu.Unlock()

	assert.False(t, completedWhileRefreshLocked, "invalidation must wait for an in-progress refresh")
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("invalidation did not complete after refresh released its critical section")
	}
	assert.True(t, f.lastRefresh.IsZero())
}

func TestReviewFilterHelpers(t *testing.T) {
	assert.True(t, isASCIIWord("hello"))
	assert.True(t, isASCIIWord("Hello"))
	assert.False(t, isASCIIWord("hello1"))
	assert.False(t, isASCIIWord("中文"))
	assert.False(t, isASCIIWord(""))

	ascii := buildMatcher("ass")
	require.NotNil(t, ascii.regex)
	assert.True(t, matchWord(ascii, "bad ass move", "bad ass move"))
	assert.False(t, matchWord(ascii, "classroom", "classroom"))

	zh := buildMatcher("作弊")
	assert.Nil(t, zh.regex)
	assert.True(t, matchWord(zh, "这是一条作弊样例", "这是一条作弊样例"))
}

func TestReviewFilterCheckContentLevels(t *testing.T) {
	ctx := context.Background()
	f := seededFilter([]SensitiveWord{
		{Word: "作弊", Level: "block"},
		{Word: "foo", Level: "warn"},
		{Word: "bar", Level: "review"},
	})

	block, err := f.CheckContent(ctx, "这里存在作弊行为")
	require.NoError(t, err)
	assert.False(t, block.IsValid)
	assert.Equal(t, "block", block.Level)
	assert.Equal(t, 1, block.MatchCount)

	warn, err := f.CheckContent(ctx, "foo should only warn")
	require.NoError(t, err)
	assert.True(t, warn.IsValid)
	assert.Equal(t, "warn", warn.Level)
	assert.Equal(t, 1, warn.MatchCount)

	review, err := f.CheckContent(ctx, "bar should be reviewed")
	require.NoError(t, err)
	assert.True(t, review.IsValid)
	assert.Equal(t, "review", review.Level)
	assert.Equal(t, 1, review.MatchCount)

	clean, err := f.CheckContent(ctx, "totally clean content")
	require.NoError(t, err)
	assert.True(t, clean.IsValid)
	assert.Empty(t, clean.Level)
	assert.Zero(t, clean.MatchCount)
}

func TestReviewFilterContainsBlockedWord(t *testing.T) {
	ctx := context.Background()
	f := seededFilter([]SensitiveWord{{Word: "danger", Level: "block"}})

	blocked, err := f.ContainsBlockedWord(ctx, "danger appears here")
	require.NoError(t, err)
	assert.True(t, blocked)

	blocked, err = f.ContainsBlockedWord(ctx, "safe content")
	require.NoError(t, err)
	assert.False(t, blocked)
}
