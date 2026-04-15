/**
 * 草稿相关类型定义 — 直接别名到 OpenAPI 生成类型
 */
import type { components } from '../api.gen'
import type { ReviewRatings } from './review'

export type Draft = components['schemas']['ReviewDraft']

/** 保存草稿参数 */
export interface SaveDraftParams {
  courseID: number
  teacherID?: number
  termID?: string
  title?: string
  content?: string
  grade?: string
  ratings?: ReviewRatings
}
