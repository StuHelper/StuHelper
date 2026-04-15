import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import { fileURLToPath, URL } from 'node:url'

const devProxyTarget = process.env.VITE_DEV_PROXY_TARGET || 'http://localhost:8080'

// source-first：monorepo 内部直接解析 shared 源码，不依赖预构建 dist
const sharedSrc = fileURLToPath(new URL('../shared/src', import.meta.url))

export default defineConfig({
  plugins: [
    vue(),
    tailwindcss(),
    AutoImport({
      imports: ['vue', 'vue-router', 'pinia', '@vueuse/core'],
      dts: 'src/auto-imports.d.ts'
    }),
    Components({
      dts: 'src/components.d.ts'
    })
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      // @stuhelper/shared 及其子路径统一解析到 shared 源码（source-first，D-15）
      '@stuhelper/shared': sharedSrc,
    }
  },
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: devProxyTarget,
        changeOrigin: true
      }
    }
  }
})
