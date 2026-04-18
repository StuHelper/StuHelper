package fga

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/config"
)

func TestNewClient_RequiresAPIURLWhenStoreConfigured(t *testing.T) {
	client, err := NewClient(config.OpenFGAConfig{StoreID: "store-1"})
	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "APIUrl is required")
}

func TestValidateTupleField(t *testing.T) {
	assert.Error(t, validateTupleField("", "user"))
	assert.Error(t, validateTupleField("bad\nuser", "user"))
	assert.NoError(t, validateTupleField("user:123", "user"))
}

func TestClientInputValidation(t *testing.T) {
	c := &Client{}
	ctx := context.Background()

	_, err := c.Check(ctx, "", "viewer", "review:1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user")

	err = c.WriteTuples(ctx, nil)
	require.NoError(t, err)

	err = c.WriteTuples(ctx, []Tuple{{User: "", Relation: "author", Object: "review:1"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user")

	err = c.DeleteTuples(ctx, nil)
	require.NoError(t, err)

	_, err = c.ReadTuples(ctx, "", "viewer")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "object")

	_, err = c.ReadTuples(ctx, "review:1", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "relation")
}
