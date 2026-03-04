import { ref, type Ref } from 'vue'

export interface AsyncDataState<T> {
  data: Ref<T | null>
  loading: Ref<boolean>
  error: Ref<Error | null>
  execute: () => Promise<void>
  reset: () => void
}

/**
 * 统一的异步数据加载 composable
 * 消除组件中重复的 loading/error 处理逻辑
 */
export function useAsyncData<T>(
  fetcher: () => Promise<T>,
  options?: {
    immediate?: boolean
    initialData?: T
  }
): AsyncDataState<T> {
  const data = ref<T | null>(options?.initialData ?? null) as Ref<T | null>
  const loading = ref(false)
  const error = ref<Error | null>(null)

  const execute = async () => {
    loading.value = true
    error.value = null
    try {
      data.value = await fetcher()
    } catch (e) {
      error.value = e instanceof Error ? e : new Error(String(e))
    } finally {
      loading.value = false
    }
  }

  const reset = () => {
    data.value = options?.initialData ?? null
    loading.value = false
    error.value = null
  }

  if (options?.immediate !== false) {
    execute()
  }

  return { data, loading, error, execute, reset }
}
