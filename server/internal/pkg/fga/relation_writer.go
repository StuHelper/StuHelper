package fga

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

const reviewModerationSectionSuffix = "_review_moderation"

var ensuredTupleCache sync.Map

func ReviewModerationSectionID(schoolID string) string {
	return "school_" + strings.TrimSpace(schoolID) + reviewModerationSectionSuffix
}

// WriteReviewRelations 评课发布时写入完整关系链，authorUserID 必须是内部 users.id。
func (c *Client) WriteReviewRelations(ctx context.Context, reviewID, authorUserID, courseID, schoolID string) error {
	sectionID := ReviewModerationSectionID(schoolID)
	if err := c.ensureTupleOnce(ctx, Tuple{
		User: "school:" + schoolID, Relation: "school", Object: "section:" + sectionID,
	}); err != nil {
		return err
	}
	return c.WriteMissingTuples(ctx, []Tuple{
		{User: "user:" + authorUserID, Relation: "author", Object: "review:" + reviewID},
		{User: "course:" + courseID, Relation: "course", Object: "review:" + reviewID},
		{User: "school:" + schoolID, Relation: "school", Object: "review:" + reviewID},
		{User: "section:" + sectionID, Relation: "section", Object: "review:" + reviewID},
	})
}

// WriteReportRelations 举报创建时写入完整关系链，reporterUserID 必须是内部 users.id。
func (c *Client) WriteReportRelations(ctx context.Context, reportID, reporterUserID, reviewID, schoolID string) error {
	sectionID := ReviewModerationSectionID(schoolID)
	if err := c.ensureTupleOnce(ctx, Tuple{
		User: "school:" + schoolID, Relation: "school", Object: "section:" + sectionID,
	}); err != nil {
		return err
	}
	return c.WriteMissingTuples(ctx, []Tuple{
		{User: "user:" + reporterUserID, Relation: "reporter", Object: "report:" + reportID},
		{User: "review:" + reviewID, Relation: "review", Object: "report:" + reportID},
		{User: "school:" + schoolID, Relation: "school", Object: "report:" + reportID},
		{User: "section:" + sectionID, Relation: "section", Object: "report:" + reportID},
	})
}

func (c *Client) ensureTupleOnce(ctx context.Context, tuple Tuple) error {
	cacheKey := c.storeID + "\x00" + c.modelID + "\x00" + tuple.Object + "\x00" + tuple.Relation + "\x00" + tuple.User
	if _, ok := ensuredTupleCache.Load(cacheKey); ok {
		return nil
	}
	if err := c.WriteMissingTuples(ctx, []Tuple{tuple}); err != nil {
		return err
	}
	ensuredTupleCache.Store(cacheKey, struct{}{})
	return nil
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
