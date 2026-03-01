/**
 * 管理后台 API
 */
import api from './index'
import type {
  Report,
  ProcessReportParams,
  AdminStats,
  OperationLog,
  BatchUpdateParams,
  AdminEditReviewParams,
  AdminTeacher,
  CreateTeacherParams,
  UpdateTeacherParams,
  AdminSensitiveWord,
  CreateSensitiveWordParams,
  UpdateSensitiveWordParams
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

// 更新评论状态（支持可选的屏蔽原因）
export function updateReviewStatus(reviewID: string, action: string, reason?: string) {
  const body: Record<string, string> = { action }
  if (reason) body.reason = reason
  return api.put<{ success: boolean }>(`/admin/reviews/${reviewID}`, body)
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

// 导出数据（NDJSON 流式导出，responseType: 'blob' 时 axios 返回 AxiosResponse<Blob>）
export function exportData(format: 'json' | 'ndjson' | 'csv' = 'json', status = 'all') {
  return api.get('/admin/export', {
    params: { format, status },
    responseType: 'blob'
  }) as Promise<import('axios').AxiosResponse<Blob>>
}

// 管理员编辑评论内容
export function adminEditReview(reviewID: string, params: AdminEditReviewParams) {
  return api.post<{ message: string }>(`/admin/reviews/${reviewID}/edit`, params)
}

// 获取教师列表（管理员）
export function getAdminTeachers(page = 1, pageSize = 20, search = '', departmentID?: number) {
  return api.get<PaginatedResponse<AdminTeacher>>('/admin/teachers', {
    params: { page, pageSize, search: search || undefined, departmentID: departmentID || undefined }
  })
}

// 创建教师
export function createTeacher(params: CreateTeacherParams) {
  return api.post<AdminTeacher>('/admin/teachers', params)
}

// 更新教师
export function updateTeacher(id: number, params: UpdateTeacherParams) {
  return api.put<{ message: string }>(`/admin/teachers/${id}`, params)
}

// 删除教师
export function deleteTeacher(id: number) {
  return api.delete<{ message: string }>(`/admin/teachers/${id}`)
}

// 获取敏感词列表
export function getSensitiveWords(page = 1, pageSize = 20, category = '', level = '') {
  return api.get<PaginatedResponse<AdminSensitiveWord>>('/admin/sensitive-words', {
    params: { page, pageSize, category: category || undefined, level: level || undefined }
  })
}

// 创建敏感词
export function createSensitiveWord(params: CreateSensitiveWordParams) {
  return api.post<AdminSensitiveWord>('/admin/sensitive-words', params)
}

// 更新敏感词
export function updateSensitiveWord(id: string, params: UpdateSensitiveWordParams) {
  return api.put<{ message: string }>(`/admin/sensitive-words/${id}`, params)
}

// 删除敏感词
export function deleteSensitiveWord(id: string) {
  return api.delete<{ message: string }>(`/admin/sensitive-words/${id}`)
}
