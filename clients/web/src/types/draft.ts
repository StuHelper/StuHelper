/**
 * 草稿相关类型定义
 */
import type { ReviewRatings } from './review'

// 草稿
export interface Draft {
  id: string
  courseID: number
  teacherID?: number
  termID?: string
  title?: string
  content?: string
  grade?: string
  ratings: ReviewRatings
  updatedAt: string
}

// 保存草稿参数
export interface SaveDraftParams {
  courseID: number
  teacherID?: number
  termID?: string
  title?: string
  content?: string
  grade?: string
  ratings?: ReviewRatings
}
