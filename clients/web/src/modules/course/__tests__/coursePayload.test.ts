import { describe, expect, it } from 'vitest'

import {
  readCourseListPayload,
  readCoursePagePayload,
  readDepartmentArrayPayload,
  readTeacherStatsArrayPayload,
  readTeacherSummaryListPayload,
  readTeacherSummaryPagePayload,
  readTermArrayPayload,
} from '../coursePayload'

const course = {
  id: 1,
  schoolID: 1,
  departmentID: 1,
  departmentName: '计算机科学与技术学院',
  code: 'CS101',
  name: '数据结构',
  credits: 3,
  category: '专业课',
  reviewCount: 12,
  isFavorited: false,
}

const department = {
  id: 1,
  schoolID: 1,
  name: '计算机科学与技术学院',
  shortName: '计算机学院',
  category: 'engineering',
  sortOrder: 10,
}

const term = {
  id: '2026-spring',
  schoolID: 1,
  name: '2026 春',
  isCurrent: true,
}

const teacher = {
  teacherID: 1,
  teacherName: '张老师',
  departmentName: '计算机科学与技术学院',
  avgRating: 4.6,
  courseCount: 3,
  reviewCount: 18,
  tags: ['认真负责'],
}

const teacherSummary = {
  teacherID: 2,
  teacherName: '李老师',
  departmentName: '软件学院',
  avgRating: 4.3,
  reviewCount: 9,
  courseCount: 2,
}

describe('course payload readers', () => {
  it('reads valid course, department, term, and teacher payloads', () => {
    expect(readCourseListPayload({ list: [course] })).toEqual([course])
    expect(readCoursePagePayload({ list: [course], total: 1 })).toEqual({
      list: [course],
      total: 1,
    })
    expect(readDepartmentArrayPayload([department])).toEqual([department])
    expect(readTermArrayPayload([term])).toEqual([term])
    expect(readTeacherStatsArrayPayload([teacher])).toEqual([teacher])
    expect(readTeacherSummaryListPayload({ list: [teacherSummary] })).toEqual([
      teacherSummary,
    ])
    expect(readTeacherSummaryPagePayload({ list: [teacherSummary], total: 1 })).toEqual({
      list: [teacherSummary],
      total: 1,
    })
  })

  it('fails closed when course fields are malformed', () => {
    expect(() =>
      readCourseListPayload({
        list: [{ ...course, credits: '3' }],
      }),
    ).toThrow('Invalid courses response')
  })

  it('fails closed when department fields are malformed', () => {
    expect(() =>
      readDepartmentArrayPayload([{ ...department, category: undefined }]),
    ).toThrow('Invalid departments response')
  })

  it('fails closed when term fields are malformed', () => {
    expect(() =>
      readTermArrayPayload([{ ...term, isCurrent: undefined }]),
    ).toThrow('Invalid terms response')
  })

  it('fails closed when teacher fields are malformed', () => {
    expect(() =>
      readTeacherStatsArrayPayload([{ ...teacher, courseCount: -1 }]),
    ).toThrow('Invalid course teachers response')
  })

  it('fails closed when teacher summary fields are malformed', () => {
    expect(() =>
      readTeacherSummaryPagePayload({
        list: [{ ...teacherSummary, reviewCount: '9' }],
        total: 1,
      }),
    ).toThrow('Invalid teachers response')
  })
})
