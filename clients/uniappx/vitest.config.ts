import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vitest/config'

export default defineConfig({
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      '@stuhelper/shared': fileURLToPath(new URL('../shared/src', import.meta.url)),
      '@stuhelper/shared/': fileURLToPath(new URL('../shared/src/', import.meta.url)),
    },
  },
  test: {
    environment: 'node',
  },
})
