/**
 * 用户中心状态管理
 */
import { defineStore } from 'pinia'
import { ref, type Ref } from 'vue'
import type { FavoriteCourse } from '@/types/course'
import { api } from '@/api'


// 通用分页获取辅助函数
async function fetchPaginated<T>(
  apiFn: (page: number, pageSize: number) => Promise<{ data?: { data?: { list: T[]; total: number } } }>,
  listRef: Ref<T[]>,
  totalRef: Ref<number>,
  loadingRef: Ref<boolean>,
  errorRef: Ref<string | null>,
  page: number,
  pageSize: number
) {
  loadingRef.value = true
  errorRef.value = null
  try {
    const res = await apiFn(page, pageSize)
    const items = res.data?.data?.list || []
    if (page === 1) {
      listRef.value = items
    } else {
      listRef.value = [...listRef.value, ...items]
    }
    totalRef.value = res.data?.data?.total || 0
  } catch (err) {
    errorRef.value = err instanceof Error ? err.message : String(err)
    throw err
  } finally {
    loadingRef.value = false
  }
}

export const useUserStore = defineStore('user', () => {
  // 我的收藏
  const myFavorites = ref<FavoriteCourse[]>([])
  const myFavoritesTotal = ref(0)
  const myFavoritesLoading = ref(false)
  const myFavoritesError = ref<string | null>(null)

  // 收藏状态缓存：true=已收藏, false=未收藏, undefined=未知（未加载）
  const favoriteStatus = ref<Partial<Record<number, boolean>>>({})

  // 获取我的收藏
  const fetchMyFavorites = async (page = 1, pageSize = 10) => {
    await fetchPaginated(api.user.getMyFavorites, myFavorites, myFavoritesTotal, myFavoritesLoading, myFavoritesError, page, pageSize)
    // 更新收藏状态缓存
    const updated = { ...favoriteStatus.value }
    for (const c of myFavorites.value) {
      updated[c.id] = true
    }
    favoriteStatus.value = updated
  }

  // 切换收藏状态
  const toggleFavorite = async (courseID: number) => {
    const current = favoriteStatus.value[courseID] ?? false

    // 乐观更新
    favoriteStatus.value = { ...favoriteStatus.value, [courseID]: !current }

    try {
      if (current) {
        await api.user.removeFavorite(courseID)
      } else {
        await api.user.addFavorite(courseID)
      }
    } catch (err) {
      // 回滚
      favoriteStatus.value = { ...favoriteStatus.value, [courseID]: current }
      throw err
    }
  }

  // 返回值：true=已收藏, false=未收藏, undefined=未知
  const isFavorited = (courseID: number): boolean | undefined => {
    return favoriteStatus.value[courseID]
  }

  // 设置服务端已知的收藏状态（由 course detail / list 响应推送）
  const setFavoriteStatus = (courseID: number, status: boolean) => {
    favoriteStatus.value = { ...favoriteStatus.value, [courseID]: status }
  }

  // 按需加载未知的收藏状态
  const ensureFavoriteStatus = async (courseID: number) => {
    if (favoriteStatus.value[courseID] !== undefined) return
    try {
      const res = await api.user.getFavoriteStatus(courseID)
      const favorited = res.data?.data?.favorited ?? false
      favoriteStatus.value = { ...favoriteStatus.value, [courseID]: favorited }
    } catch {
      // 加载失败保持未知状态
    }
  }

  // 重置状态（setup store 不支持 $reset）
  const reset = () => {
    myFavorites.value = []
    myFavoritesTotal.value = 0
    myFavoritesLoading.value = false
    myFavoritesError.value = null
    favoriteStatus.value = {}
  }

  return {
    myFavorites,
    myFavoritesTotal,
    myFavoritesLoading,
    myFavoritesError,
    favoriteStatus,
    fetchMyFavorites,
    toggleFavorite,
    isFavorited,
    setFavoriteStatus,
    ensureFavoriteStatus,
    reset
  }
})
