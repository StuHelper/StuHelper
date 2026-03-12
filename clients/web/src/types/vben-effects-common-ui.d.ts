declare module '@cimom/vben-effects-common-ui' {
  import type { DefineComponent } from 'vue'

  export const CountTo: DefineComponent<{
    endVal: number
    duration?: number
    separator?: string
  }>
}
