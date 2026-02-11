/**
 * 回复相关类型定义
 */

// 回复
export interface Reply {
  id: number
  reviewID: number
  parentID?: number
  content: string
  likeCount: number
  status: string
  createdAt: string
  updatedAt: string
  isOwner: boolean
}

// 发表回复参数
export interface PostReplyParams {
  reviewID: number
  parentID?: number
  content: string
}
