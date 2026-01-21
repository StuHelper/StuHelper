import type { RatingLevel } from './course'

// 测评
export interface Review {
  id: string
  courseId: number
  courseName?: string
  teacherName?: string
  termName?: string
  title: string
  content: string
  grade?: string
  ratingRecommend: RatingLevel
  ratingContent: RatingLevel
  ratingWorkload: RatingLevel
  ratingExam: RatingLevel
  likeCount: number
  dislikeCount: number
  createdAt: string
}
