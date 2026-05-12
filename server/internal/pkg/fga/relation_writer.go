package fga

import (
	"context"
	"fmt"
)

func (c *Client) WriteReviewModerationSection(ctx context.Context, schoolID string) error {
	return c.WriteMissingTuples(ctx, []Tuple{
		{User: "school:" + schoolID, Relation: "school", Object: "section:" + ReviewModerationSectionID(schoolID)},
	})
}

// WriteReviewRelations 评课发布或对账时写入 admin Check 会读取的动态关系。
// authorUserID 是内部 users.id；历史导入数据可能没有可解析作者，此时只写管理授权所需的 school/section 关系。
func (c *Client) WriteReviewRelations(ctx context.Context, reviewID, authorUserID, schoolID string) error {
	sectionID := ReviewModerationSectionID(schoolID)
	tuples := make([]Tuple, 0, 3)
	if authorUserID != "" {
		tuples = append(tuples, Tuple{User: "user:" + authorUserID, Relation: "author", Object: "review:" + reviewID})
	}
	tuples = append(tuples,
		Tuple{User: "school:" + schoolID, Relation: "school", Object: "review:" + reviewID},
		Tuple{User: "section:" + sectionID, Relation: "section", Object: "review:" + reviewID},
	)
	return c.WriteMissingTuples(ctx, tuples)
}

// WriteReportRelations 举报创建时写入 admin Check 会读取的动态关系。
func (c *Client) WriteReportRelations(ctx context.Context, reportID, schoolID string) error {
	sectionID := ReviewModerationSectionID(schoolID)
	return c.WriteMissingTuples(ctx, []Tuple{
		{User: "school:" + schoolID, Relation: "school", Object: "report:" + reportID},
		{User: "section:" + sectionID, Relation: "section", Object: "report:" + reportID},
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
