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
 *
 * 默认会立即执行一次加载；仅当调用点明确需要延迟触发时传入
 * `immediate: false`，避免把“懒加载”需求错误地塞进默认行为。
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

  // scope 销毁后标记为非活跃，异步回调中跳过状态写入
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
