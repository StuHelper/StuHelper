package review

import "strconv"

type paginatedReviewListData struct {
	List  []Review `json:"list"`
	Total int      `json:"total"`
}

type groupedReviewListData map[string]paginatedReviewListData

func normalizeReviewList(list []Review) []Review {
	if list == nil {
		return []Review{}
	}
	return list
}

func buildPaginatedReviewListData(list []Review, total int) paginatedReviewListData {
	return paginatedReviewListData{
		List:  normalizeReviewList(list),
		Total: total,
	}
}

func buildGroupedReviewListData(courseIDs []int64, result *BatchCourseReviewsResult, facts ReviewAccessFacts) groupedReviewListData {
	grouped := make(groupedReviewListData, len(courseIDs))
	for _, courseID := range courseIDs {
		reviews := stripReviewsForResponse(normalizeReviewList(result.Reviews[courseID]), facts)
		grouped[strconv.FormatInt(courseID, 10)] = buildPaginatedReviewListData(reviews, result.Totals[courseID])
	}
	return grouped
}
