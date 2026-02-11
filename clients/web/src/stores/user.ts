/**
 * 用户中心状态管理
 */
import { defineStore } from 'pinia'
import { ref, type Ref } from 'vue'
import type { Review } from '@/types/review'
import type { Course } from '@/types/course'
import * as userApi from '@/api/user'


// 通用分页获取辅助函数
async function fetchPaginated<T>(
  apiFn: (page: number, pageSize: number) => Promise<{ data?: { list: T[]; total: number } }>,
  listRef: Ref<T[]>,
  totalRef: Ref<number>,
  loadingRef: Ref<boolean>,
  page: number,
  pageSize: number
) {
  loadingRef.value = true
  try {
    const res = await apiFn(page, pageSize)
    const items = res.data?.list || []
    if (page === 1) {
      listRef.value = items
    } else {
      listRef.value = [...listRef.value, ...items]
    }
    totalRef.value = res.data?.total || 0
  } finally {
    loadingRef.value = false
  }
}

export const useUserStore = defineStore('user', () => {
  // 我的评论
  const myReviews = ref<Review[]>([])
  const myReviewsTotal = ref(0)
  const myReviewsLoading = ref(false)

  // 我的点赞
  const myVotes = ref<Review[]>([])
  const myVotesTotal = ref(0)
  const myVotesLoading = ref(false)

  // 我的收藏
  const myFavorites = ref<Course[]>([])
  const myFavoritesTotal = ref(0)
  const myFavoritesLoading = ref(false)

  // 收藏状态缓存
  const favoriteIDs = ref<Set<number>>(new Set())

  // 获取我的评论
  const fetchMyReviews = (page = 1, pageSize = 10) =>
    fetchPaginated(userApi.getMyReviews, myReviews, myReviewsTotal, myReviewsLoading, page, pageSize)

  // 获取我的点赞
  const fetchMyVotes = (page = 1, pageSize = 10) =>
    fetchPaginated(userApi.getMyVotes, myVotes, myVotesTotal, myVotesLoading, page, pageSize)

  // 获取我的收藏
  const fetchMyFavorites = async (page = 1, pageSize = 10) => {
    await fetchPaginated(userApi.getMyFavorites, myFavorites, myFavoritesTotal, myFavoritesLoading, page, pageSize)
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
        await userApi.removeFavorite(courseID)
      } else {
        await userApi.addFavorite(courseID)
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

  return {
    myReviews,
    myReviewsTotal,
    myReviewsLoading,
    myVotes,
    myVotesTotal,
    myVotesLoading,
    myFavorites,
    myFavoritesTotal,
    myFavoritesLoading,
    favoriteIDs,
    fetchMyReviews,
    fetchMyVotes,
    fetchMyFavorites,
    toggleFavorite,
    isFavorited
  }
})
