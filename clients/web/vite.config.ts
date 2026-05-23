import { defineConfig, type Plugin } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import AutoImport from 'unplugin-auto-import/vite'
import Components from 'unplugin-vue-components/vite'
import type { IncomingMessage, ServerResponse } from 'node:http'
import { createRequire } from 'node:module'
import { fileURLToPath, URL } from 'node:url'

const devProxyTarget = process.env.VITE_DEV_PROXY_TARGET || 'http://localhost:8080'
const e2eAPIStubEnabled = process.env.VITE_E2E_API_STUB === '1'

// source-first：monorepo 内部直接解析 shared 源码，不依赖预构建 dist
const sharedSrc = fileURLToPath(new URL('../shared/src', import.meta.url))
const require = createRequire(import.meta.url)
const vueEntry = require.resolve('vue/dist/vue.runtime.esm-bundler.js')
const vuePackageJSON = require.resolve('vue/package.json')
const vueRequire = createRequire(vuePackageJSON)
const runtimeDomEntry = vueRequire.resolve('@vue/runtime-dom/dist/runtime-dom.esm-bundler.js')
const runtimeDomPackageJSON = vueRequire.resolve('@vue/runtime-dom/package.json')
const runtimeDomRequire = createRequire(runtimeDomPackageJSON)

function resolveVueRuntimeEntry(packageName: string, entry: string) {
  return runtimeDomRequire.resolve(`${packageName}/dist/${entry}`)
}

function e2eAPIStubPlugin(): Plugin {
  return {
    name: 'stuhelper-e2e-api-stub',
    configureServer(server) {
      server.middlewares.use((req: IncomingMessage, res: ServerResponse, next) => {
        const requestURL = req.url ?? ''
        if (!requestURL.startsWith('/api/')) {
          next()
          return
        }

        if (requestURL.includes('/notifications/stream')) {
          res.statusCode = 200
          res.setHeader('Content-Type', 'text/event-stream')
          res.setHeader('Cache-Control', 'no-cache')
          res.end()
          return
        }

        if (requestURL === '/api/v1/metrics/vitals' || requestURL.startsWith('/api/v1/metrics/vitals?')) {
          res.statusCode = 204
          res.end()
          return
        }

        res.statusCode = 200
        res.setHeader('Content-Type', 'application/json')
        res.end(JSON.stringify({ data: null }))
      })
    },
  }
}

export default defineConfig({
  plugins: [
    e2eAPIStubEnabled && e2eAPIStubPlugin(),
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
    dedupe: ['vue', '@vue/runtime-core', '@vue/runtime-dom', '@vue/reactivity', '@vue/shared'],
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      // @stuhelper/shared 及其子路径统一解析到 shared 源码（source-first，D-15）
      '@stuhelper/shared': sharedSrc,
      // 显式绑定同一条 Vue 运行时依赖链，避免 pnpm 多版本提升导致构建阶段混用。
      vue: vueEntry,
      '@vue/runtime-dom': runtimeDomEntry,
      '@vue/runtime-core': resolveVueRuntimeEntry('@vue/runtime-core', 'runtime-core.esm-bundler.js'),
      '@vue/reactivity': resolveVueRuntimeEntry('@vue/reactivity', 'reactivity.esm-bundler.js'),
      '@vue/shared': resolveVueRuntimeEntry('@vue/shared', 'shared.esm-bundler.js'),
    }
  },
  server: {
    port: 3000,
    proxy: e2eAPIStubEnabled
      ? undefined
      : {
          '/api': {
            target: devProxyTarget,
            changeOrigin: true
          },
          '/.well-known': {
            target: devProxyTarget,
            changeOrigin: true
          },
          '/oauth2': {
            target: devProxyTarget,
            changeOrigin: true
          },
          '/oidc': {
            target: devProxyTarget,
            changeOrigin: true
          }
        }
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('/node_modules/echarts/') || id.includes('/node_modules/zrender/')) {
            return 'echarts'
          }
          return undefined
        }
      }
    }
  }
})
