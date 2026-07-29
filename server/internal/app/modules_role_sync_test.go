package app

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/StuHelper/StuHelper/server/internal/pkg/config"
)

func TestObjectStorageConfiguredTreatsBlankEndpointAsMissing(t *testing.T) {
	assert.False(t, objectStorageConfigured(config.ObjectStorageConfig{Endpoint: "  "}))
	assert.True(t, objectStorageConfigured(config.ObjectStorageConfig{Endpoint: "https://s3.example.com"}))
}

func TestCasdoorRoleSyncConfiguredTreatsBlankValuesAsMissing(t *testing.T) {
	cfg := config.CasdoorConfig{
		RoleSyncClientID:       "  ",
		RoleSyncClientSecret:   "  ",
		RoleSyncApplication:    "  ",
		UserLookupClientID:     "  ",
		UserLookupClientSecret: "  ",
		UserLookupApplication:  "  ",
	}

	assert.False(t, casdoorRoleSyncConfigured(cfg))
}

func TestCasdoorRoleSyncConfiguredIncludesApplicationFields(t *testing.T) {
	assert.True(t, casdoorRoleSyncConfigured(config.CasdoorConfig{
		RoleSyncApplication: "casdoor-admin-role-sync",
	}))
	assert.True(t, casdoorRoleSyncConfigured(config.CasdoorConfig{
		UserLookupApplication: "casdoor-admin-user-lookup",
	}))
}
