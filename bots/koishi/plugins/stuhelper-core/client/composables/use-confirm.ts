import { shallowRef } from 'vue'

export type ConfirmTone = 'normal' | 'danger'

export interface ConfirmOptions {
  title: string
  message: string
  tone?: ConfirmTone
  confirmText?: string
  cancelText?: string
}

export interface ConfirmDialogState {
  open: boolean
  title: string
  message: string
  tone: ConfirmTone
  confirmText: string
  cancelText: string
}

const CLOSED_STATE: ConfirmDialogState = {
  open: false,
  title: '确认操作',
  message: '',
  tone: 'normal',
  confirmText: '确认',
  cancelText: '取消',
}

export function useConfirm() {
  const state = shallowRef<ConfirmDialogState>({ ...CLOSED_STATE })
  let resolvePending: ((confirmed: boolean) => void) | null = null

  function confirm(options: ConfirmOptions): Promise<boolean> {
    if (resolvePending) {
      return Promise.resolve(false)
    }

    state.value = {
      open: true,
      title: options.title,
      message: options.message,
      tone: options.tone ?? 'normal',
      confirmText: options.confirmText ?? '确认',
      cancelText: options.cancelText ?? '取消',
    }

    return new Promise((resolve) => {
      resolvePending = resolve
    })
  }

  function close(confirmed: boolean) {
    const resolve = resolvePending
    resolvePending = null
    state.value = { ...state.value, open: false }
    resolve?.(confirmed)
  }

  return {
    state,
    confirm,
    accept: () => close(true),
    cancel: () => close(false),
  }
}
