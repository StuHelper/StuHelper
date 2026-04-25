/**
 * 乐观更新工具
 * 先更新 UI，再发 API，失败时回滚
 */
import { ref } from 'vue'
import i18n from '@/i18n'
import { getErrorMessage } from '@/api/errors'
import { useToast } from './useToast'

// 返回结果对象，调用方可区分成功与失败
export interface OptimisticResult<T> {
  success: boolean
  data?: T
  error?: unknown
}

export function useOptimisticUpdate() {
  const t = i18n.global.t
  const toast = useToast()
  const pending = ref(false)

  async function execute<T>(options: {
    optimistic: () => void
    request: () => Promise<T>
    rollback: () => void
    errorMessage?: string
  }): Promise<OptimisticResult<T>> {
    pending.value = true
    options.optimistic()

    try {
      const result = await options.request()
      return { success: true, data: result }
    } catch (err) {
      options.rollback()
      toast.error(getErrorMessage(err, options.errorMessage || t('common.actions.operationFailed')))
      return { success: false, error: err }
    } finally {
      pending.value = false
    }
  }

  return { execute, pending }
}
