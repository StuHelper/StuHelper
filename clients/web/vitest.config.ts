import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

const sharedSrc = fileURLToPath(new URL('../shared/src', import.meta.url))

export default defineConfig({
  plugins: [vue()],
  resolve: {
    dedupe: ['vue', '@vue/runtime-core', '@vue/runtime-dom', '@vue/reactivity', '@vue/shared'],
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      '@stuhelper/shared': sharedSrc,
    }
  },
  test: {
    environment: 'node',
    globals: true,
    exclude: ['tests/e2e/**/*', 'node_modules/**/*'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'html']
    }
  }
})
