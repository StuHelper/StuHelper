/**
 * 草稿 API
 */
import api from './index'
import type { Draft, SaveDraftParams } from '@/types/draft'

// 保存草稿
export function saveDraft(data: SaveDraftParams) {
  return api.post<Draft>('/drafts', data)
}

// 获取草稿
export function getDraft(courseID: number) {
  return api.get<Draft | null>(`/drafts/${courseID}`)
}

// 删除草稿
export function deleteDraft(courseID: number) {
  return api.delete<{ success: boolean }>(`/drafts/${courseID}`)
}
