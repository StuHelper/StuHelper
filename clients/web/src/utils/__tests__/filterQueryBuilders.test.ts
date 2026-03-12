import { describe, expect, it } from 'vitest'
import {
  buildDepartmentCoursesQuery,
  buildDepartmentListQuery,
  buildSensitiveWordsQuery,
  buildTeachersQuery,
} from '../filterQueryBuilders'

describe('filter query builders', () => {
  it('builds department list query from selected category', () => {
    expect(buildDepartmentListQuery('')).toEqual({})
    expect(buildDepartmentListQuery('通识')).toEqual({ category: '通识' })
  })

  it('builds department course query with department and optional category', () => {
    expect(buildDepartmentCoursesQuery(12, '')).toEqual({
      page: 1,
      pageSize: 100,
      departmentID: 12,
    })
    expect(buildDepartmentCoursesQuery(12, '专业课')).toEqual({
      page: 1,
      pageSize: 100,
      departmentID: 12,
      category: '专业课',
    })
  })

  it('builds teacher admin query with search and department filter', () => {
    expect(buildTeachersQuery({ page: 2, pageSize: 20, searchQuery: '张', filterDeptID: 3 })).toEqual({
      page: 2,
      pageSize: 20,
      search: '张',
      departmentID: 3,
    })
  })

  it('omits empty teacher filters from query', () => {
    expect(buildTeachersQuery({ page: 1, pageSize: 20, searchQuery: '   ', filterDeptID: 0 })).toEqual({
      page: 1,
      pageSize: 20,
    })
  })

  it('builds sensitive word query with category and level', () => {
    expect(buildSensitiveWordsQuery({ page: 3, pageSize: 50, filterCategory: 'review', filterLevel: 'warn' })).toEqual({
      page: 3,
      pageSize: 50,
      category: 'review',
      level: 'warn',
    })
  })

  it('omits empty sensitive word filters', () => {
    expect(buildSensitiveWordsQuery({ page: 1, pageSize: 20, filterCategory: '', filterLevel: '' })).toEqual({
      page: 1,
      pageSize: 20,
    })
  })
})
