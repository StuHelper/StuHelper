/**
 * 乐观更新工具
 * 先更新 UI，再发 API，失败时回滚
 */
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from './useToast'

export function useOptimisticUpdate() {
  const { t } = useI18n()
  const toast = useToast()
  const pending = ref(false)

  async function execute<T>(options: {
    optimistic: () => void
    request: () => Promise<T>
    rollback: () => void
    errorMessage?: string
  }): Promise<T | null> {
    pending.value = true
    options.optimistic()

    try {
      const result = await options.request()
      return result
    } catch {
      options.rollback()
      toast.error(options.errorMessage || t('common.actions.operationFailed'))
      return null
    } finally {
      pending.value = false
    }
  }

  return { execute, pending }
}
