/**
 * 管理后台 API
 */
import api from './index'
import type {
  Report,
  ProcessReportParams,
  AdminStats,
  OperationLog,
  BatchUpdateParams
} from '@/types/admin'

export type { Report, AdminStats, OperationLog }
import type { Review } from '@/types/review'
import type { PaginatedResponse } from '@/types/api'

// 获取管理统计
export function getAdminStats() {
  return api.get<AdminStats>('/admin/stats')
}

// 获取举报列表
export function getReports(status = 'pending', page = 1, pageSize = 20) {
  return api.get<PaginatedResponse<Report>>('/admin/reports', {
    params: { status, page, pageSize }
  })
}

// 处理举报
export function processReport(reportID: string, params: ProcessReportParams) {
  return api.put<{ success: boolean }>(`/admin/reports/${reportID}`, params)
}

// 获取所有评论（管理员）
export function getAllReviews(
  status = 'all',
  page = 1,
  pageSize = 20
) {
  return api.get<PaginatedResponse<Review>>('/admin/reviews', {
    params: { status, page, pageSize }
  })
}

// 更新评论状态
export function updateReviewStatus(reviewID: string, action: string) {
  return api.put<{ success: boolean }>(`/admin/reviews/${reviewID}`, { action })
}

// 批量更新评论
export function batchUpdateReviews(params: BatchUpdateParams) {
  return api.post<{ affected: number }>('/admin/reviews/batch', params)
}

// 获取操作日志
export function getOperationLogs(page = 1, pageSize = 20) {
  return api.get<PaginatedResponse<OperationLog>>('/admin/logs', {
    params: { page, pageSize }
  })
}

// 导出数据
export function exportData(format: 'json' | 'csv' = 'json', status = 'all') {
  return api.get<Blob>('/admin/export', {
    params: { format, status },
    responseType: 'blob'
  })
}
