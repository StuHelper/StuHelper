import path from 'node:path'
import { createRequire } from 'node:module'
import { fileURLToPath, URL } from 'node:url'

import { defineConfig, type Plugin } from 'vite'
import uni from '@dcloudio/vite-plugin-uni'

const devProxyTarget = process.env.VITE_DEV_PROXY_TARGET || 'http://localhost:8080'
const sharedSrc = fileURLToPath(new URL('../shared/src', import.meta.url))
const require = createRequire(import.meta.url)
const vueRouterEntry = path.join(
  path.dirname(require.resolve('vue-router/package.json')),
  'dist/vue-router.mjs',
)

function vueRouterESMEntryPlugin(): Plugin {
  return {
    name: 'stuhelper-uniappx-vue-router-esm-entry',
    enforce: 'pre',
    resolveId(id) {
      if (id === 'vue-router') {
        return vueRouterEntry
      }
      return null
    },
  }
}

export default defineConfig({
  plugins: [vueRouterESMEntryPlugin(), uni()],
  resolve: {
    alias: {
      '@stuhelper/shared': sharedSrc,
      vue: '@dcloudio/uni-h5-vue',
      '@vue/server-renderer': '@dcloudio/uni-h5-vue/server-renderer',
    },
  },
  server: {
    proxy: {
      '/api': {
        changeOrigin: true,
        target: devProxyTarget,
      },
    },
  },
})
