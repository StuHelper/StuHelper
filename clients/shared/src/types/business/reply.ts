/**
 * 回复相关类型定义 — 直接别名到 OpenAPI 生成类型
 */
import type { components } from '../api.gen'

export type Reply = components['schemas']['Reply']

/** 发表回复参数 */
export interface PostReplyParams {
  reviewID: string
  parentID?: string | null
  content: string
}
