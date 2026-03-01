/**
 * 用户中心 API
 */
import api from './index'
import type { Review } from '@/types/review'
import type { FavoriteCourse } from '@/types/course'
import type { PaginatedResponse } from '@/types/api'

// 获取我的评论
export function getMyReviews(page = 1, pageSize = 10) {
  return api.get<PaginatedResponse<Review>>('/user/reviews', {
    params: { page, pageSize }
  })
}

// 获取我的点赞
export function getMyVotes(page = 1, pageSize = 10) {
  return api.get<PaginatedResponse<Review>>('/user/votes', {
    params: { page, pageSize }
  })
}

// 获取我的收藏
export function getMyFavorites(page = 1, pageSize = 10) {
  return api.get<PaginatedResponse<FavoriteCourse>>('/user/favorites', {
    params: { page, pageSize }
  })
}

// 收藏课程
export function addFavorite(courseID: number) {
  return api.post<{ success: boolean }>(`/courses/${courseID}/favorites`)
}

// 取消收藏
export function removeFavorite(courseID: number) {
  return api.delete<{ success: boolean }>(`/courses/${courseID}/favorites`)
}
