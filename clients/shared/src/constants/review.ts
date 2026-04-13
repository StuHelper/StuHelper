/**
 * 测评表单验证常量
 * 与后端 reviews.title CHECK(LENGTH <= 100) / reviews.content CHECK(LENGTH BETWEEN 10 AND 5000) 对齐
 */
export const REVIEW_GRADES = ['A+', 'A', 'A-', 'B+', 'B', 'B-', 'C+', 'C', 'C-', 'D', 'F'] as const
export type ReviewGrade = typeof REVIEW_GRADES[number]

export function isReviewGrade(value: unknown): value is ReviewGrade {
  return typeof value === 'string' && REVIEW_GRADES.includes(value as ReviewGrade)
}

export const REVIEW_TITLE_MAX_LENGTH = 100
export const REVIEW_CONTENT_MIN_LENGTH = 10
export const REVIEW_CONTENT_MAX_LENGTH = 5000
