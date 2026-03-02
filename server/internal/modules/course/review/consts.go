package review

// 评论状态常量
const (
	StatusPublished = "published"
	StatusHidden    = "hidden"
	StatusDeleted   = "deleted"
	StatusAll       = "all"
)

// 举报状态常量
const (
	ReportStatusPending  = "pending"
	ReportStatusResolved = "resolved"
	ReportStatusRejected = "rejected"
)

// 排序选项常量
const (
	SortTime   = "time"
	SortLikes  = "likes"
	SortRating = "rating"
)

// isValidReviewStatus 校验评论状态参数是否合法（含 "all"）
func isValidReviewStatus(s string) bool {
	switch s {
	case StatusPublished, StatusHidden, StatusDeleted, StatusAll:
		return true
	}
	return false
}

// isValidReportStatus 校验举报状态参数是否合法（含 "all"）
func isValidReportStatus(s string) bool {
	switch s {
	case ReportStatusPending, ReportStatusResolved, ReportStatusRejected, StatusAll:
		return true
	}
	return false
}

// isValidSort 校验排序参数是否合法
func isValidSort(s string) bool {
	switch s {
	case SortTime, SortLikes, SortRating:
		return true
	}
	return false
}
