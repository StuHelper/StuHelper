package user

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIAMOutboxMigrationRoutesUserProfileProjectionToOpenFGAStream(t *testing.T) {
	up := readServerFile(t, "migrations", "000026_iam_outbox_streams.up.sql")
	assert.Contains(t, up, "SET stream = 'iam_openfga_tuple_sync'\nWHERE stream = 'user_external_sync'\n  AND job_type = 'user_profile_projection';")
	assert.NotContains(t, projectionMigrationBlock(up), "iam_casdoor_user_projection")
}

func TestIAMOutboxRepairMigrationMovesMisroutedUserProfileProjection(t *testing.T) {
	up := readServerFile(t, "migrations", "000032_repair_user_profile_projection_stream.up.sql")
	assert.Contains(t, up, "SET stream = 'iam_openfga_tuple_sync'")
	assert.Contains(t, up, "WHERE stream = 'iam_casdoor_user_projection'\n  AND job_type = 'user_profile_projection';")
}

func projectionMigrationBlock(sql string) string {
	start := strings.Index(sql, "job_type = 'user_profile_projection'")
	if start == -1 {
		return ""
	}
	lineStart := strings.LastIndex(sql[:start], "UPDATE domain_event_outbox")
	if lineStart == -1 {
		return sql[start:]
	}
	lineEnd := strings.Index(sql[start:], ";")
	if lineEnd == -1 {
		return sql[lineStart:]
	}
	return sql[lineStart : start+lineEnd+1]
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
