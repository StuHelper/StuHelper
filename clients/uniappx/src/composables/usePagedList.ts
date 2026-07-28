import { computed, ref, type ComputedRef, type Ref } from 'vue'

export interface PageResult<T> {
  list: T[]
  total: number
}

export type PageLoadPhase = 'refresh' | 'more'

export interface PagedListOptions<T> {
  pageSize: number
  fetchPage: (page: number, pageSize: number) => Promise<PageResult<T>>
  onError: (error: unknown, phase: PageLoadPhase) => void
}

export interface PagedListController<T> {
  items: Ref<T[]>
  page: Ref<number>
  total: Ref<number>
  loading: Ref<boolean>
  loadingMore: Ref<boolean>
  hasMore: ComputedRef<boolean>
  refresh: () => Promise<boolean>
  loadMore: () => Promise<boolean>
}

/**
 * 统一列表分页状态机。
 *
 * - 下一页成功后才提交页码，失败重试不会跳页；
 * - 以服务端 total 为准，精确整页时不会多请求一次；
 * - 新刷新会淘汰旧响应，避免搜索/筛选竞态覆盖新结果。
 */
export function usePagedList<T>(options: PagedListOptions<T>): PagedListController<T> {
  const items = ref<T[]>([]) as Ref<T[]>
  const page = ref(1)
  const total = ref(0)
  const loading = ref(false)
  const loadingMore = ref(false)
  const hasMore = computed(() => items.value.length < total.value)
  let generation = 0

  async function refresh(): Promise<boolean> {
    const requestGeneration = ++generation
    loading.value = true
    loadingMore.value = false

    try {
      const result = await options.fetchPage(1, options.pageSize)
      if (requestGeneration !== generation) return false

      items.value = result.list
      page.value = 1
      total.value = result.total
      return true
    } catch (error) {
      if (requestGeneration === generation) {
        options.onError(error, 'refresh')
      }
      return false
    } finally {
      if (requestGeneration === generation) {
        loading.value = false
      }
    }
  }

  async function loadMore(): Promise<boolean> {
    if (loading.value || loadingMore.value || !hasMore.value) return false

    const requestGeneration = generation
    const nextPage = page.value + 1
    loadingMore.value = true

    try {
      const result = await options.fetchPage(nextPage, options.pageSize)
      if (requestGeneration !== generation) return false

      items.value = [...items.value, ...result.list]
      page.value = nextPage
      total.value = result.total
      return true
    } catch (error) {
      if (requestGeneration === generation) {
        options.onError(error, 'more')
      }
      return false
    } finally {
      if (requestGeneration === generation) {
        loadingMore.value = false
      }
    }
  }

  return {
    items,
    page,
    total,
    loading,
    loadingMore,
    hasMore,
    refresh,
    loadMore,
  }
}
