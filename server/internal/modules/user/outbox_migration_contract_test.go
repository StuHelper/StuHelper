package user

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIAMOutboxBaselineUsesFinalStreams(t *testing.T) {
	up := readServerFile(t, "migrations", "000001_initial_schema.up.sql")
	assert.Contains(t, up, "CREATE TABLE public.domain_event_outbox")
	assert.Contains(t, up, "CONSTRAINT chk_domain_event_outbox_stream CHECK ((stream <> ''::text))")
	assert.Contains(t, up, "'dead_letter'::text")
	assert.Contains(t, up, "domain_event_outbox_stream_pending_idx")
	assert.NotContains(t, up, "user_external_sync_outbox")
	assert.NotContains(t, up, "review_fga_sync_outbox")
	assert.NotContains(t, up, "iam_casdoor_user_projection")
}

func readServerFile(t *testing.T, parts ...string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	serverRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	path := filepath.Join(append([]string{serverRoot}, parts...)...)
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(content)
}
