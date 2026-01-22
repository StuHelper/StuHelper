import api from './index'
import type { Review, PostReviewParams } from '@/types/review'

// 获取课程测评
export const getCourseReviews = (courseId: number, page = 1, pageSize = 10) => {
  return api.get<{ list: Review[]; total: number }>(`/courses/${courseId}/reviews`, {
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
export const postReview = (data: PostReviewParams) => {
  return api.post('/reviews', {
    course_id: data.courseId,
    teacher_id: data.teacherId,
    term_id: data.termId,
    title: data.title,
    content: data.content,
    grade: data.grade,
    ratings: data.ratings
  })
}

// 投票
export const voteReview = (reviewId: number, voteType: 'like' | 'dislike') => {
  return api.post(`/reviews/${reviewId}/vote`, { vote_type: voteType })
}
