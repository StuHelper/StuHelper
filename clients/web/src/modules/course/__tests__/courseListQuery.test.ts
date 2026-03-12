import { describe, expect, it } from 'vitest'
import { buildCourseListQuery } from '../courseListQuery'

describe('buildCourseListQuery', () => {
  it('builds a service-driven query from filters and pagination', () => {
    expect(buildCourseListQuery({
      page: 2,
      pageSize: 12,
      searchQuery: '线性代数',
      selectedDepartment: 5,
      sortBy: 'reviewCount'
    })).toEqual({
      page: 2,
      pageSize: 12,
      q: '线性代数',
      departmentID: 5,
      sort: 'reviewCount'
    })
  })

  it('omits empty search and department filters', () => {
    expect(buildCourseListQuery({
      page: 1,
      pageSize: 20,
      searchQuery: '   ',
      selectedDepartment: '',
      sortBy: 'name'
    })).toEqual({
      page: 1,
      pageSize: 20,
      sort: 'name'
    })
  })
})
