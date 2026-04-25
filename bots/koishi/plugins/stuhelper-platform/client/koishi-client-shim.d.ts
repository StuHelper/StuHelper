declare module '@koishijs/client' {
  import type { Component } from 'vue'

  export interface Context {
    page(input: {
      readonly name: string
      readonly path: string
      readonly component: Component
    }): void
  }

  export const store: Record<string, unknown>

  export function send(type: string, ...args: unknown[]): Promise<unknown> | undefined
}
