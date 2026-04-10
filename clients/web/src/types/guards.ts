/**
 * 类型守卫函数
 * 用于运行时类型验证，替代类型断言
 */

// 成绩等级
export const GRADES = ['A+', 'A', 'A-', 'B+', 'B', 'B-', 'C+', 'C', 'C-', 'D', 'F'] as const
export type Grade = typeof GRADES[number]

export function isValidGrade(value: unknown): value is Grade {
  return typeof value === 'string' && GRADES.includes(value as Grade)
}

// 排序类型
export const SORT_TYPES = ['time', 'likes', 'rating'] as const
export type SortType = typeof SORT_TYPES[number]

export function isValidSortType(value: unknown): value is SortType {
  return typeof value === 'string' && SORT_TYPES.includes(value as SortType)
}

// 评论状态
export const REVIEW_STATUSES = ['published', 'pending_review', 'hidden', 'deleted', 'all'] as const
export type ReviewStatus = typeof REVIEW_STATUSES[number]

export function isValidReviewStatus(value: unknown): value is ReviewStatus {
  return typeof value === 'string' && REVIEW_STATUSES.includes(value as ReviewStatus)
}

// 举报状态
export const REPORT_STATUSES = ['pending', 'resolved', 'rejected', 'all'] as const
export type ReportStatus = typeof REPORT_STATUSES[number]

export function isValidReportStatus(value: unknown): value is ReportStatus {
  return typeof value === 'string' && REPORT_STATUSES.includes(value as ReportStatus)
}

// 管理员操作
export const ADMIN_ACTIONS = ['hide', 'restore', 'delete'] as const
export type AdminAction = typeof ADMIN_ACTIONS[number]

export function isValidAdminAction(value: unknown): value is AdminAction {
  return typeof value === 'string' && ADMIN_ACTIONS.includes(value as AdminAction)
}

// 敏感词级别
export const SENSITIVE_WORD_LEVELS = ['block', 'warn', 'review'] as const
export type SensitiveWordLevel = typeof SENSITIVE_WORD_LEVELS[number]

export function isValidSensitiveWordLevel(value: unknown): value is SensitiveWordLevel {
  return typeof value === 'string' && SENSITIVE_WORD_LEVELS.includes(value as SensitiveWordLevel)
}
