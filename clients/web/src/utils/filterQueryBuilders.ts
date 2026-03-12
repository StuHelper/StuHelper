export function buildDepartmentListQuery(category: string) {
  const normalized = category.trim()
  return normalized ? { category: normalized } : {}
}

export function buildDepartmentCoursesQuery(departmentID: number, category: string) {
  const normalized = category.trim()
  return {
    page: 1,
    pageSize: 100,
    departmentID,
    ...(normalized ? { category: normalized } : {}),
  }
}

export function buildTeachersQuery(input: {
  page: number
  pageSize: number
  searchQuery: string
  filterDeptID: number
}) {
  const normalizedSearch = input.searchQuery.trim()
  return {
    page: input.page,
    pageSize: input.pageSize,
    ...(normalizedSearch ? { search: normalizedSearch } : {}),
    ...(input.filterDeptID > 0 ? { departmentID: input.filterDeptID } : {}),
  }
}

export function buildSensitiveWordsQuery(input: {
  page: number
  pageSize: number
  filterCategory: string
  filterLevel: '' | 'block' | 'warn' | 'review'
}): {
  page: number
  pageSize: number
  category?: string
  level?: 'block' | 'warn' | 'review'
} {
  const normalizedCategory = input.filterCategory.trim()
  return {
    page: input.page,
    pageSize: input.pageSize,
    ...(normalizedCategory ? { category: normalizedCategory } : {}),
    ...(input.filterLevel ? { level: input.filterLevel } : {}),
  }
}
