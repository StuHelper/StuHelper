export interface PaginationParams {
  page: number
  pageSize: number
}

export interface PaginatedResponse<T> {
  data: T[]
  total: number
  totalPages: number
  totalCount: number
  page: number
  pageSize: number
}
