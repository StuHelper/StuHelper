/**
 * 管理后台相关类型定义
 */

// 举报状态
export type ReportStatus = 'pending' | 'resolved' | 'rejected'

// 举报中嵌套的评论摘要
export interface ReportReview {
  id: string
  courseID: number
  courseName: string
  teacherID?: number
  teacherName?: string
  termID?: string
  title?: string
  content: string
  grade?: string
  ratings: Record<string, number>
  likeCount: number
  dislikeCount: number
  replyCount: number
  status: string
  createdAt: string
}

// 举报
export interface Report {
  id: string
  reviewID: string
  review?: ReportReview
  reason: string
  description?: string
  status: ReportStatus
  resolvedBy?: string
  resolvedAt?: string
  resolutionNote?: string
  createdAt: string
}

// 处理举报参数
export interface ProcessReportParams {
  action: 'reject' | 'hide_review' | 'delete_review'
  note?: string
}

// 管理统计
export interface AdminStats {
  totalReviews: number
  publishedReviews: number
  hiddenReviews: number
  deletedReviews: number
  pendingReports: number
  totalReports: number
  todayReviews: number
  weekReviews: number
}

// 操作日志
export interface OperationLog {
  id: string
  adminUserID: string
  adminUsername: string
  action: string
  resourceType: string
  resourceID: string
  oldValue?: Record<string, unknown>
  newValue?: Record<string, unknown>
  ipAddress?: string
  userAgent?: string
  createdAt: string
}

// 批量操作参数
export interface BatchUpdateParams {
  /** 评论 ID 列表（UUIDv7 格式） */
  ids: string[]
  action: 'hide' | 'restore' | 'delete'
}

// 导出参数
export interface ExportParams {
  format: 'json' | 'csv'
  status?: string
}
