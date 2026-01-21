// 评分等级类型
export type RatingLevel = -2 | -1 | 0 | 1 | 2

// 院系
export interface Department {
  id: number
  name: string
  shortName?: string
  category: string
}

// 课程
export interface Course {
  id: number
  name: string
  code?: string
  credits: number
  departmentId: number
  reviewCount: number
  avgRecommend: number
  avgContent: number
  avgWorkload: number
  avgExam: number
  department?: Department
}
