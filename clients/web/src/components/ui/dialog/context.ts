import type { ComputedRef, InjectionKey } from 'vue'

export interface DialogContext {
  descriptionId: string
  open: ComputedRef<boolean>
  setOpen: (open: boolean) => void
  titleId: string
}

export const dialogContextKey = Symbol('DialogContext') as InjectionKey<DialogContext>
