/**
 * 排行榜 API
 */
import api from './index'

// 热门课程（对齐后端 review.HotCourse）
export interface HotCourse {
  courseID: number
  courseName: string
  reviewCount: number
  avgRating: number
}

// 获取热门课程排行
export function getHotRankings(period: 'week' | 'month' | 'all' = 'all', limit = 10) {
  return api.get<HotCourse[]>('/rankings/hot', {
    params: { period, limit }
  })
}
