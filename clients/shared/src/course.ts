export type {
  Course,
  CourseCategory,
  Department,
  FavoriteCourse,
  RatingDimension,
  Term,
} from './types/business/course'

export type {
  CourseRatingStatsResponse,
  DimensionStats,
  RadarChartData,
  RadarChartDataset,
  RatingValue,
  TeacherStats,
  TermRatingStats,
} from './presentation/course'

export { RATING_VALUES, isValidRating } from './presentation/course'
