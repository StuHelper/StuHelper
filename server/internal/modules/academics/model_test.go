package academics

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSourceJSONContract(t *testing.T) {
	payload, err := json.Marshal(Source{
		ID:       1,
		Key:      "buaa-fixture",
		Name:     "北航教务夹具",
		Provider: "fixture",
		Config:   []byte(`{"secret":"token"}`),
		Enabled:  true,
	})
	require.NoError(t, err)

	var body map[string]any
	require.NoError(t, json.Unmarshal(payload, &body))

	assert.Contains(t, body, "id")
	assert.Contains(t, body, "key")
	assert.Contains(t, body, "name")
	assert.Contains(t, body, "provider")
	assert.Contains(t, body, "enabled")
	assert.NotContains(t, body, "config")
	assert.NotContains(t, body, "Config")
	assert.NotContains(t, body, "ID")
	assert.NotContains(t, body, "Key")
	assert.NotContains(t, body, "Name")
	assert.NotContains(t, body, "Provider")
	assert.NotContains(t, body, "Enabled")
}
