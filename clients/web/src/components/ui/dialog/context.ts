import type { ComputedRef, InjectionKey } from 'vue'

export interface DialogContext {
  open: ComputedRef<boolean>
  setOpen: (open: boolean) => void
}

export const dialogContextKey = Symbol('DialogContext') as InjectionKey<DialogContext>
