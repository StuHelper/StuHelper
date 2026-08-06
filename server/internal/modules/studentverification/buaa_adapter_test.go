package studentverification

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBUAAAdapterNormalizesAndValidatesMainlandDocument(t *testing.T) {
	adapter := BUAAAdapter{}

	studentID, ok := adapter.NormalizeStudentID(" 20990001 ")
	require.True(t, ok)
	assert.Equal(t, "20990001", studentID)

	name, ok := adapter.NormalizeName(" 张三 ")
	require.True(t, ok)
	assert.Equal(t, "张三", name)

	document, ok := adapter.NormalizeMainlandDocumentNumber("11010519491231002x")
	require.True(t, ok)
	assert.Equal(t, "11010519491231002X", document)
}

func TestBUAAAdapterRejectsMalformedInputAndUnknownDocumentPolicy(t *testing.T) {
	adapter := BUAAAdapter{}

	_, ok := adapter.NormalizeStudentID("23 374368")
	assert.False(t, ok)
	_, ok = adapter.NormalizeMainlandDocumentNumber("110105194912310021")
	assert.False(t, ok)

	assert.False(t, adapter.SupportsMainlandDocumentType("1", nil))
	assert.False(t, adapter.SupportsMainlandDocumentType("1", json.RawMessage(`{"mainlandDocumentTypes":[]}`)))
	assert.True(t, adapter.SupportsMainlandDocumentType(
		"1",
		json.RawMessage(`{"mainlandDocumentTypes":["1"]}`),
	))
}
