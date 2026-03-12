import { shallowRef, ref, onScopeDispose, type Ref } from 'vue'

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
  const data = shallowRef<T | null>(options?.initialData ?? null)
  const loading = ref(false)
  const error = ref<Error | null>(null)

  // S-3: 组件/scope 销毁后不再写入 refs，防止异步操作写入已卸载组件的响应式状态
  let isActive = true
  onScopeDispose(() => {
    isActive = false
  })

  const execute = async () => {
    loading.value = true
    error.value = null
    try {
      const result = await fetcher()
      if (isActive) {
        data.value = result
      }
    } catch (e) {
      if (isActive) {
        error.value = e instanceof Error ? e : new Error(String(e))
      }
    } finally {
      if (isActive) {
        loading.value = false
      }
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
