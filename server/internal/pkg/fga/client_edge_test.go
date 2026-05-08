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
	assert.NoError(t, validateTupleField("user:018f3f9a-6e8b-7d11_a1", "user"))
	assert.NoError(t, validateTupleField("section:review_moderation#member", "user"))
	assert.Error(t, validateTupleField("user:bad.id", "user"))
	assert.Error(t, validateTupleField("user123", "user"))
	assert.NoError(t, validateTupleField("review:1", "object"))
	assert.NoError(t, validateTupleField("review:018f3f9a-6e8b-7d11_a1", "object"))
	assert.Error(t, validateTupleField("review:bad.id", "object"))
	assert.Error(t, validateTupleField("review", "object"))
	assert.NoError(t, validateTupleField("can_delete", "relation"))
	assert.Error(t, validateTupleField("can-delete", "relation"))
	assert.NoError(t, validateTupleField("review", "object type"))
	assert.Error(t, validateTupleField("review:type", "object type"))
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

	err = c.DeleteTuples(ctx, []Tuple{{User: "user-1", Relation: "author", Object: "review:1"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "type:id")

	_, err = c.ReadTuples(ctx, "", "viewer")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "object")

	_, err = c.ReadTuples(ctx, "review:1", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "relation")

	_, err = c.ListObjects(ctx, "", "viewer", "review")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "user")

	_, err = c.ListObjects(ctx, "user:1", "", "review")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "relation")

	_, err = c.ListObjects(ctx, "user:1", "viewer", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "object type")
}
