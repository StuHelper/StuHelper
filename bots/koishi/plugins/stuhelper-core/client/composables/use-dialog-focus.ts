import {
  nextTick,
  onBeforeUnmount,
  watch,
  type Ref,
} from 'vue'

const FOCUSABLE_SELECTOR = [
  'a[href]',
  'button:not([disabled])',
  'input:not([disabled]):not([type="hidden"])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  '[contenteditable="true"]',
  '[tabindex]:not([tabindex="-1"])',
].join(',')

function focusableElements(dialog: HTMLElement): HTMLElement[] {
  return Array.from(dialog.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR))
    .filter((element) => {
      if (element.hidden || element.getAttribute('aria-hidden') === 'true') return false
      const style = window.getComputedStyle(element)
      return style.display !== 'none' && style.visibility !== 'hidden'
    })
}

export function useDialogFocus(
  open: Ref<boolean>,
  dialogRef: Ref<HTMLElement | null>,
  close: () => void,
) {
  let restoreTarget: HTMLElement | null = null

  function restoreFocus() {
    const target = restoreTarget
    restoreTarget = null
    if (!target?.isConnected) return
    void nextTick(() => target.focus({ preventScroll: true }))
  }

  watch(
    open,
    (isOpen, wasOpen) => {
      if (isOpen && !wasOpen) {
        restoreTarget = document.activeElement instanceof HTMLElement
          ? document.activeElement
          : null
        void nextTick(() => {
          const dialog = dialogRef.value
          if (!dialog) return
          const autofocusTarget = dialog.querySelector<HTMLElement>('[autofocus]')
          ;(autofocusTarget ?? dialog).focus({ preventScroll: true })
        })
      } else if (!isOpen && wasOpen) {
        restoreFocus()
      }
    },
    { immediate: true, flush: 'post' },
  )

  function handleKeydown(event: KeyboardEvent) {
    if (event.key === 'Escape') {
      event.preventDefault()
      event.stopPropagation()
      close()
      return
    }
    if (event.key !== 'Tab') return

    const dialog = dialogRef.value
    if (!dialog) return
    const focusable = focusableElements(dialog)
    if (focusable.length === 0) {
      event.preventDefault()
      dialog.focus({ preventScroll: true })
      return
    }

    const first = focusable[0]
    const last = focusable[focusable.length - 1]
    const active = document.activeElement
    if (event.shiftKey && (active === first || !dialog.contains(active))) {
      event.preventDefault()
      last.focus()
    } else if (!event.shiftKey && (active === last || !dialog.contains(active))) {
      event.preventDefault()
      first.focus()
    }
  }

  onBeforeUnmount(() => {
    if (open.value) restoreFocus()
  })

  return { handleKeydown }
}
