/**
 * 草稿状态管理
 */
import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Draft, SaveDraftParams } from '@/types/draft'
import * as draftApi from '@/api/draft'

export const useDraftStore = defineStore('draft', () => {
  // 草稿缓存
  const drafts = ref<Map<number, Draft>>(new Map())

  // 保存状态
  const saving = ref(false)
  const lastSavedAt = ref<Date | null>(null)

  // 保存草稿
  const saveDraft = async (data: SaveDraftParams) => {
    saving.value = true
    try {
      const res = await draftApi.saveDraft(data)
      if (res.data) {
        drafts.value.set(data.courseID, res.data)
        lastSavedAt.value = new Date()
      }
      return res.data
    } finally {
      saving.value = false
    }
  }

  // 获取草稿
  const loadDraft = async (courseID: number) => {
    // 先检查缓存
    if (drafts.value.has(courseID)) {
      return drafts.value.get(courseID)!
    }

    try {
      const res = await draftApi.getDraft(courseID)
      if (res.data) {
        drafts.value.set(courseID, res.data)
      }
      return res.data
    } catch {
      return null
    }
  }

  // 删除草稿
  const deleteDraft = async (courseID: number) => {
    await draftApi.deleteDraft(courseID)
    drafts.value.delete(courseID)
  }

  // 检查是否有草稿
  const hasDraft = (courseID: number) => {
    return drafts.value.has(courseID)
  }

  // 获取缓存的草稿
  const getCachedDraft = (courseID: number) => {
    return drafts.value.get(courseID)
  }

  return {
    drafts,
    saving,
    lastSavedAt,
    saveDraft,
    loadDraft,
    deleteDraft,
    hasDraft,
    getCachedDraft
  }
})
