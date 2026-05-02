package review

import (
	"sort"
	"strconv"
	"strings"
)

func adminReportsCacheKey(status string, page, pageSize int, schoolIDs []int64) string {
	return "status=" + status +
		":page=" + strconv.Itoa(page) +
		":size=" + strconv.Itoa(pageSize) +
		":scope=" + moderationScopeCachePart(schoolIDs)
}

func moderationScopeCachePart(schoolIDs []int64) string {
	if schoolIDs == nil {
		return "all"
	}
	if len(schoolIDs) == 0 {
		return "none"
	}

	ids := append([]int64(nil), schoolIDs...)
	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})

	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, strconv.FormatInt(id, 10))
	}
	return "schools:" + strings.Join(parts, ",")
}
