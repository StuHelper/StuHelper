import { ref } from 'vue'

import type { NoticeItem } from '../components/primitives/NoticeStack.vue'
import { errorMessage } from '../utils/error-message'

const DEFAULT_ACTION_ERROR_TITLE = '操作失败'
const NOTICE_DISMISS_DELAY_MS = 4000

export function useActionFeedback() {
  const actionError = ref('')
  const actionErrorTitle = ref(DEFAULT_ACTION_ERROR_TITLE)
  const notices = ref<NoticeItem[]>([])

  function pushSuccess(message: string) {
    pushNotice({ kind: 'success', message })
  }

  function pushError(title: string, message: string) {
    pushNotice({ kind: 'error', title, message })
  }

  function setActionError(title: string, cause: unknown, fallback: string) {
    const message = errorMessage(cause, fallback)
    actionErrorTitle.value = title
    actionError.value = message
    pushError(title, message)
  }

  function clearActionError() {
    actionErrorTitle.value = DEFAULT_ACTION_ERROR_TITLE
    actionError.value = ''
  }

  function dismissNotice(id: string) {
    notices.value = notices.value.filter((item) => item.id !== id)
  }

  function pushNotice(input: Omit<NoticeItem, 'id'>) {
    const id = noticeId()
    notices.value.push({ id, ...input })
    const timer = globalThis.setTimeout(() => dismissNotice(id), NOTICE_DISMISS_DELAY_MS)
    if (typeof timer === 'object' && timer && 'unref' in timer && typeof timer.unref === 'function') {
      timer.unref()
    }
  }

  return {
    actionError,
    actionErrorTitle,
    notices,
    pushSuccess,
    pushError,
    setActionError,
    clearActionError,
    dismissNotice,
    errorMessage,
  }
}

export { errorMessage }

function noticeId(): string {
  return `notice-${Math.random().toString(36).slice(2, 8)}-${Date.now()}`
}
