import api from './index'
import type { Review } from '@/types/review'

// 获取课程测评
export const getCourseReviews = (courseId: number, page = 1, pageSize = 10) => {
  return api.get(`/courses/${courseId}/reviews`, {
    params: { page, page_size: pageSize }
  })
}

// 获取最新测评
export const getLatestReviews = (page = 1, pageSize = 10) => {
  return api.get<{ list: Review[]; total: number }>('/reviews/latest', {
    params: { page, page_size: pageSize }
  })
}

// 发布测评
export interface PostReviewParams {
  courseId: number
  title?: string
  content: string
  ratingRecommend: number
  ratingContent: number
  ratingWorkload: number
  ratingExam: number
}

export const postReview = (data: PostReviewParams) => {
  return api.post('/reviews', {
    course_id: data.courseId,
    title: data.title,
    content: data.content,
    rating_recommend: data.ratingRecommend,
    rating_content: data.ratingContent,
    rating_workload: data.ratingWorkload,
    rating_exam: data.ratingExam
  })
}

// 投票
export const voteReview = (reviewId: number, voteType: 'like' | 'dislike') => {
  return api.post(`/reviews/${reviewId}/vote`, { vote_type: voteType })
}
