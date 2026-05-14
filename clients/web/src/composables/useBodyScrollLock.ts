import { onUnmounted, watch, type Ref } from 'vue'

let lockCount = 0
let savedBodyOverflow = ''

function lockBodyScroll() {
  if (lockCount === 0) {
    savedBodyOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
  }
  lockCount += 1
}

function unlockBodyScroll() {
  if (lockCount === 0) return
  lockCount -= 1
  if (lockCount === 0) {
    document.body.style.overflow = savedBodyOverflow
    savedBodyOverflow = ''
  }
}

export function useBodyScrollLock(locked: Ref<boolean>) {
  let active = false

  function apply(nextLocked: boolean) {
    if (nextLocked === active) return
    active = nextLocked
    if (active) {
      lockBodyScroll()
      return
    }
    unlockBodyScroll()
  }

  watch(locked, apply, { immediate: true })

  onUnmounted(() => {
    if (!active) return
    active = false
    unlockBodyScroll()
  })
}
