import { ref } from 'vue'

import { errorMessage } from '../utils/error-message'

const DEFAULT_ACTION_ERROR_TITLE = '操作失败'

export interface UseActionErrorOptions {
  defaultTitle?: string
  onError?: (title: string, message: string) => void
}

export function useActionError(options: UseActionErrorOptions = {}) {
  const defaultTitle = options.defaultTitle ?? DEFAULT_ACTION_ERROR_TITLE
  const actionError = ref('')
  const actionErrorTitle = ref(defaultTitle)

  function setActionError(title: string, cause: unknown, fallback: string): string {
    const message = errorMessage(cause, fallback)
    actionErrorTitle.value = title
    actionError.value = message
    options.onError?.(title, message)
    return message
  }

  function clearActionError() {
    actionErrorTitle.value = defaultTitle
    actionError.value = ''
  }

  return {
    actionError,
    actionErrorTitle,
    setActionError,
    clearActionError,
    errorMessage,
  }
}

export { errorMessage }
