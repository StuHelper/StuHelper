/**
 * 草稿状态管理
 */
import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Draft, SaveDraftParams } from '@/types/draft'
import { api } from '@/api'

// M-120: 草稿缓存最大条目数
const MAX_DRAFT_ENTRIES = 100

export const useDraftStore = defineStore('draft', () => {
  // L-49: 使用 Record 替代 Map，确保完整的 Pinia 响应式支持
  // H-20: reactive Record 的属性变更能正确触发响应式更新
  const drafts = ref<Record<number, Draft>>({})
  // 缓存时间戳（毫秒）
  const cacheTimestamps = new Map<number, number>()
  // 缓存 TTL：5 分钟
  const CACHE_TTL_MS = 5 * 60 * 1000

  // 保存状态
  const saving = ref(false)
  const lastSavedAt = ref<Date | null>(null)

  function isCacheStale(courseID: number): boolean {
    const ts = cacheTimestamps.get(courseID)
    if (ts == null) return true
    return Date.now() - ts > CACHE_TTL_MS
  }

  // M-120: 超过上限时清除最旧条目
  function evictIfNeeded() {
    const keys = Object.keys(drafts.value).map(Number)
    if (keys.length <= MAX_DRAFT_ENTRIES) return
    // 按缓存时间戳排序，清除最旧的
    keys.sort((a, b) => (cacheTimestamps.get(a) ?? 0) - (cacheTimestamps.get(b) ?? 0))
    const toRemove = keys.slice(0, keys.length - MAX_DRAFT_ENTRIES)
    for (const key of toRemove) {
      delete drafts.value[key]
      cacheTimestamps.delete(key)
    }
  }

  // M-06: 校验草稿字段
  function validateDraftParams(data: SaveDraftParams): boolean {
    if (!data.courseID || !Number.isFinite(data.courseID)) return false
    if (data.title !== undefined && typeof data.title !== 'string') return false
    if (data.content !== undefined && typeof data.content !== 'string') return false
    return true
  }

  // 保存草稿
  const saveDraft = async (data: SaveDraftParams) => {
    if (!validateDraftParams(data)) return undefined
    saving.value = true
    try {
      const res = await api.draft.saveDraft(data)
      const draft = res.data?.data as Draft | undefined
      if (draft) {
        drafts.value[data.courseID] = draft
        cacheTimestamps.set(data.courseID, Date.now())
        evictIfNeeded()
        lastSavedAt.value = new Date()
      }
      return draft
    } finally {
      saving.value = false
    }
  }

  // 获取草稿
  const loadDraft = async (courseID: number, forceRefresh = false) => {
    if (!courseID || !Number.isFinite(courseID)) return null
    // 先检查缓存（未过期且非强制刷新）
    if (!forceRefresh && courseID in drafts.value && !isCacheStale(courseID)) {
      return drafts.value[courseID]
    }

    try {
      const res = await api.draft.getDraft(courseID)
      const draft = res.data?.data as Draft | undefined
      if (draft) {
        drafts.value[courseID] = draft
        cacheTimestamps.set(courseID, Date.now())
        evictIfNeeded()
      }
      return draft
    } catch {
      return null
    }
  }

  // 删除草稿
  const deleteDraft = async (courseID: number) => {
    await api.draft.deleteDraft(courseID)
    delete drafts.value[courseID]
    cacheTimestamps.delete(courseID)
  }

  // 检查是否有草稿
  const hasDraft = (courseID: number) => {
    return courseID in drafts.value
  }

  // 获取缓存的草稿
  const getCachedDraft = (courseID: number) => {
    return drafts.value[courseID] ?? undefined
  }

  // M-39: 重置状态（登出时调用，防止用户数据泄漏）
  const reset = () => {
    drafts.value = {}
    cacheTimestamps.clear()
    saving.value = false
    lastSavedAt.value = null
  }

  return {
    drafts,
    saving,
    lastSavedAt,
    saveDraft,
    loadDraft,
    deleteDraft,
    hasDraft,
    getCachedDraft,
    reset
  }
})
