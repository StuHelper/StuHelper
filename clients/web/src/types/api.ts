/**
 * 通用 API 类型定义
 */

// 分页响应类型
export interface PaginatedResponse<T> {
  list: T[]
  total: number
}
