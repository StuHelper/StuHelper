/**
 * 测评相关类型定义
 *
 * 仅保留 OpenAPI wire contract 别名和 API 补充类型。
 * normalizer / guard / UI 接口已迁至 presentation/review.ts。
 */
import type { components } from '../api.gen'

// ---- OpenAPI 别名 ----

export type Review = components['schemas']['Review']
export type PostReviewRequest = components['schemas']['PostReviewRequest']

// ---- 前端补充类型 ----

/** 动态评分（维度 key → 分值），与 OpenAPI ReviewRatings 一致 */
export type ReviewRatings = components['schemas']['ReviewRatings']
