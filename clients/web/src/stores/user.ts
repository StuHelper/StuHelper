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

  // 收藏状态缓存
  const favoriteIDs = ref<Set<number>>(new Set())

  // 获取我的收藏
  const fetchMyFavorites = async (page = 1, pageSize = 10) => {
    await fetchPaginated(api.user.getMyFavorites, myFavorites, myFavoritesTotal, myFavoritesLoading, myFavoritesError, page, pageSize)
    // 更新收藏ID缓存
    favoriteIDs.value = new Set(myFavorites.value.map(c => c.id))
  }

  // 切换收藏状态
  const toggleFavorite = async (courseID: number) => {
    const isFavorited = favoriteIDs.value.has(courseID)

    // 乐观更新（创建新 Set 触发响应性）
    const next = new Set(favoriteIDs.value)
    if (isFavorited) {
      next.delete(courseID)
    } else {
      next.add(courseID)
    }
    favoriteIDs.value = next

    try {
      if (isFavorited) {
        await api.user.removeFavorite(courseID)
      } else {
        await api.user.addFavorite(courseID)
      }
    } catch (err) {
      // 回滚（创建新 Set 触发响应性）
      const rollback = new Set(favoriteIDs.value)
      if (isFavorited) {
        rollback.add(courseID)
      } else {
        rollback.delete(courseID)
      }
      favoriteIDs.value = rollback
      throw err
    }
  }

  // 检查是否已收藏
  const isFavorited = (courseID: number) => {
    return favoriteIDs.value.has(courseID)
  }

  // 重置状态（setup store 不支持 $reset）
  const reset = () => {
    myFavorites.value = []
    myFavoritesTotal.value = 0
    myFavoritesLoading.value = false
    myFavoritesError.value = null
    favoriteIDs.value = new Set()
  }

  return {
    myFavorites,
    myFavoritesTotal,
    myFavoritesLoading,
    myFavoritesError,
    favoriteIDs,
    fetchMyFavorites,
    toggleFavorite,
    isFavorited,
    reset
  }
})
