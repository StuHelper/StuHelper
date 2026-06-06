package review

// 评论状态常量
const (
	StatusPublished     = "published"
	StatusPendingReview = "pending_review"
	StatusHidden        = "hidden"
	StatusDeleted       = "deleted"
	StatusAll           = "all"
)

// 内容审核标记常量
const (
	ContentFlagWarn    = "warn"
	ContentFlagReview  = "review"
	ContentFlagCleared = "cleared"
)

// 举报状态常量
const (
	ReportStatusPending  = "pending"
	ReportStatusResolved = "resolved"
	ReportStatusRejected = "rejected"
)

// 举报原因常量
const (
	reportReasonSpam          = "spam"
	reportReasonInappropriate = "inappropriate"
	reportReasonHarassment    = "harassment"
	reportReasonFalseInfo     = "false_info"
	reportReasonOther         = "other"
)

// 投票类型常量
const (
	voteTypeLike    = "like"
	voteTypeDislike = "dislike"
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
	case StatusPublished, StatusPendingReview, StatusHidden, StatusDeleted, StatusAll:
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

// isValidReportReason 校验举报原因是否合法
func isValidReportReason(s string) bool {
	switch s {
	case reportReasonSpam, reportReasonInappropriate, reportReasonHarassment, reportReasonFalseInfo, reportReasonOther:
		return true
	}
	return false
}

// isValidVoteType 校验评价投票类型是否合法
func isValidVoteType(s string) bool {
	switch s {
	case voteTypeLike, voteTypeDislike:
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
