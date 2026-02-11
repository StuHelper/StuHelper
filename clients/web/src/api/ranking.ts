/**
 * 排行榜 API
 */
import api from './index'

// 热门课程
export interface HotCourse {
  id: number
  name: string
  code: string
  departmentName: string
  reviewCount: number
  avgRating: number
  hotScore: number
}

// 获取热门课程排行
export function getHotRankings(period: 'week' | 'month' | 'all' = 'all', limit = 10) {
  return api.get<HotCourse[]>('/rankings/hot', {
    params: { period, limit }
  })
}
