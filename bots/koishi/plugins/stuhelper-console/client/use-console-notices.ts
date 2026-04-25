import { ref } from 'vue'

import { resolveNoticeMessage } from './ui-state'

export interface NoticeItem {
  id: string
  kind: 'success' | 'error'
  message: string
}

const NOTICE_TTL_MS = 4000

export function useConsoleNotices() {
  const loading = ref(false)
  const notices = ref<NoticeItem[]>([])

  function pushNotice(kind: NoticeItem['kind'], message: string) {
    const id = `${Date.now()}-${Math.random().toString(36).slice(2, 7)}`
    notices.value = [...notices.value, { id, kind, message }]
    window.setTimeout(() => dismissNotice(id), NOTICE_TTL_MS)
  }

  function dismissNotice(id: string) {
    notices.value = notices.value.filter((item) => item.id !== id)
  }

  async function runTask(task: () => Promise<unknown>) {
    loading.value = true
    try {
      const result = await task()
      pushNotice('success', resolveNoticeMessage(result))
      return result
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error)
      pushNotice('error', message)
      throw error
    } finally {
      loading.value = false
    }
  }

  return {
    loading,
    notices,
    pushNotice,
    dismissNotice,
    runTask,
  }
}
