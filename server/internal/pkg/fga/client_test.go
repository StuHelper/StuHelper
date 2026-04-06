package fga

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
)

func TestNewClient_NilWhenStoreIDEmpty(t *testing.T) {
	cfg := config.OpenFGAConfig{
		APIUrl:  "http://localhost:8081",
		StoreID: "",
	}
	client, err := NewClient(cfg)
	assert.NoError(t, err)
	assert.Nil(t, client, "client should be nil when StoreID is empty")
}

func TestNewClient_ReturnsClientWhenConfigured(t *testing.T) {
	cfg := config.OpenFGAConfig{
		APIUrl:               "http://localhost:8081",
		StoreID:              "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		AuthorizationModelID: "01ARZ3NDEKTSV4RRFFQ69G5FAV",
	}
	client, err := NewClient(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, client)
}

func TestTuple_BuildsCorrectOpenFGATupleKey(t *testing.T) {
	tuple := Tuple{
		User:     "user:123",
		Relation: "author",
		Object:   "review:456",
	}
	assert.Equal(t, "user:123", tuple.User)
	assert.Equal(t, "author", tuple.Relation)
	assert.Equal(t, "review:456", tuple.Object)
}
