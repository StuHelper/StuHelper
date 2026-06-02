package fga

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMissingTuplesSkipsExistingAndDuplicateDesired(t *testing.T) {
	t.Parallel()

	existing := []Tuple{
		{User: "user:1", Relation: "author", Object: "review:1"},
	}
	desired := []Tuple{
		{User: "user:1", Relation: "author", Object: "review:1"},
		{User: "course:1", Relation: "course", Object: "review:1"},
		{User: "course:1", Relation: "course", Object: "review:1"},
	}

	missing := MissingTuples(existing, desired)

	assert.Equal(t, []Tuple{
		{User: "course:1", Relation: "course", Object: "review:1"},
	}, missing)
}

func TestGroupTuplesByObjectRelationPreservesFirstSeenOrder(t *testing.T) {
	t.Parallel()

	groups := groupTuplesByObjectRelation([]Tuple{
		{User: "user:1", Relation: "author", Object: "review:1"},
		{User: "user:2", Relation: "author", Object: "review:1"},
		{User: "school:4111010001", Relation: "school", Object: "review:1"},
	})

	assert.Len(t, groups, 2)
	assert.Equal(t, "review:1", groups[0].object)
	assert.Equal(t, "author", groups[0].relation)
	assert.Len(t, groups[0].tuples, 2)
	assert.Equal(t, "school", groups[1].relation)
}
