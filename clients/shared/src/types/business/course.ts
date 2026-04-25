/**
 * 课程相关类型定义
 *
 * 仅保留 OpenAPI wire contract 别名。
 * 评分守卫 / UI 展示型接口已迁至 presentation/course.ts。
 */
import type { components } from '../api.gen'

// ---- OpenAPI 别名（唯一事实源） ----

export type Course = components['schemas']['Course']
export type Department = components['schemas']['Department']
export type Term = components['schemas']['Term']
export type CourseCategory = components['schemas']['CourseCategory']
export type FavoriteCourse = components['schemas']['FavoriteCourse']
export type RatingDimension = components['schemas']['RatingDimension']
