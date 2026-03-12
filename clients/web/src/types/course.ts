/**
 * 课程相关类型定义
 * Re-export from shared to maintain single source of truth
 */
export type {
  RatingValue,
  RatingDimension,
  Department,
  Course,
  FavoriteCourse,
  CourseCategory,
  Term,
  DimensionStats,
  TermRatingStats,
  RadarChartDataset,
  RadarChartData,
  TeacherStats,
  CourseRatingStatsResponse,
} from '@stuhelper/shared'

export { RATING_VALUES, isValidRating } from '@stuhelper/shared'
