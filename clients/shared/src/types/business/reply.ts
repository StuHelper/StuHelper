/**
 * 回复相关类型定义
 */

// 回复
export interface Reply {
  id: string
  reviewID: string
  parentID?: string | null
  content: string
  likeCount: number
  status: string
  createdAt: string
  updatedAt: string
  isOwner: boolean
}

// 发表回复参数
export interface PostReplyParams {
  reviewID: string
  parentID?: string | null
  content: string
}
