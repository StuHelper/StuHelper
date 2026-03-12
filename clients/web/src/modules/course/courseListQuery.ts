export type CourseListSort = 'name' | 'credits' | 'reviewCount'

export function buildCourseListQuery(input: {
  page: number
  pageSize: number
  searchQuery: string
  selectedDepartment: number | ''
  sortBy: CourseListSort
}) {
  const normalizedQuery = input.searchQuery.trim()

  return {
    page: input.page,
    pageSize: input.pageSize,
    ...(normalizedQuery ? { q: normalizedQuery } : {}),
    ...(input.selectedDepartment !== '' ? { departmentID: input.selectedDepartment } : {}),
    sort: input.sortBy,
  }
}
