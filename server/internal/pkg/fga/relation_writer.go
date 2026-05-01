package fga

import (
	"context"
	"fmt"
)

// WriteReviewRelations 评课发布时写入完整关系链，authorUserID 必须是内部 users.id。
func (c *Client) WriteReviewRelations(ctx context.Context, reviewID, authorUserID, courseID, schoolID string) error {
	return c.WriteMissingTuples(ctx, []Tuple{
		{User: "user:" + authorUserID, Relation: "author", Object: "review:" + reviewID},
		{User: "course:" + courseID, Relation: "course", Object: "review:" + reviewID},
		{User: "school:" + schoolID, Relation: "school", Object: "review:" + reviewID},
	})
}

// WriteReportRelations 举报创建时写入完整关系链，reporterUserID 必须是内部 users.id。
func (c *Client) WriteReportRelations(ctx context.Context, reportID, reporterUserID, reviewID, schoolID string) error {
	return c.WriteMissingTuples(ctx, []Tuple{
		{User: "user:" + reporterUserID, Relation: "reporter", Object: "report:" + reportID},
		{User: "review:" + reviewID, Relation: "review", Object: "report:" + reportID},
		{User: "school:" + schoolID, Relation: "school", Object: "report:" + reportID},
	})
}

// WriteMissingTuples writes only tuples that are not already present in OpenFGA.
func (c *Client) WriteMissingTuples(ctx context.Context, desired []Tuple) error {
	if len(desired) == 0 {
		return nil
	}

	missing := make([]Tuple, 0, len(desired))
	for _, group := range groupTuplesByObjectRelation(desired) {
		existing, err := c.ReadTuples(ctx, group.object, group.relation)
		if err != nil {
			return fmt.Errorf("fga: read before write %s#%s: %w", group.object, group.relation, err)
		}
		missing = append(missing, MissingTuples(existing, group.tuples)...)
	}
	return c.WriteTuples(ctx, missing)
}

// MissingTuples returns desired tuples that are not present in existing.
func MissingTuples(existing, desired []Tuple) []Tuple {
	seen := make(map[Tuple]struct{}, len(existing))
	for _, tuple := range existing {
		seen[tuple] = struct{}{}
	}

	missing := make([]Tuple, 0, len(desired))
	for _, tuple := range desired {
		if _, ok := seen[tuple]; ok {
			continue
		}
		seen[tuple] = struct{}{}
		missing = append(missing, tuple)
	}
	return missing
}

type tupleGroup struct {
	object   string
	relation string
	tuples   []Tuple
}

func groupTuplesByObjectRelation(tuples []Tuple) []tupleGroup {
	indexes := make(map[string]int, len(tuples))
	groups := make([]tupleGroup, 0, len(tuples))
	for _, tuple := range tuples {
		key := tuple.Object + "\x00" + tuple.Relation
		index, ok := indexes[key]
		if !ok {
			index = len(groups)
			indexes[key] = index
			groups = append(groups, tupleGroup{object: tuple.Object, relation: tuple.Relation})
		}
		groups[index].tuples = append(groups[index].tuples, tuple)
	}
	return groups
}
