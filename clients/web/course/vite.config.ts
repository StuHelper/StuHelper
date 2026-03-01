import { defineConfig, type PluginOption } from 'vite'
import tailwindcss from '@tailwindcss/vite'
import vue from '@vitejs/plugin-vue'
import { visualizer } from 'rollup-plugin-visualizer'
import { resolve } from 'path'

export default defineConfig(({ mode }) => {
  const plugins: PluginOption[] = [tailwindcss(), vue()]

  // npm run analyze 时启用 bundle 可视化（生成 dist/stats.html）
  if (mode === 'analyze') {
    plugins.push(
      visualizer({
        open: true,
        filename: 'dist/stats.html',
        gzipSize: true
      })
    )
  }

  return {
    plugins,
    resolve: {
      alias: {
        '@': resolve(__dirname, 'src')
      }
    },
    build: {
      target: 'es2021',
      sourcemap: false,
      chunkSizeWarningLimit: 300,
      rollupOptions: {
        output: {
          manualChunks(id) {
            if (id.includes('node_modules/vue-router') || id.includes('node_modules/pinia') || id.includes('node_modules/vue-i18n')) {
              return 'vue-vendor'
            }
            if (id.includes('node_modules/echarts') || id.includes('node_modules/vue-echarts') || id.includes('node_modules/zrender')) {
              return 'echarts'
            }
            if (id.includes('node_modules/axios')) {
              return 'axios'
            }
            if (id.includes('node_modules/element-plus')) {
              return 'element-plus'
            }
          }
        }
      }
    },
    server: {
      port: 3000,
      // M-30: 开发环境 CSP headers，限制 style 注入来源
      headers: {
        'Content-Security-Policy': "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; connect-src 'self' http://localhost:8080; font-src 'self'"
      },
      proxy: {
        '/api': {
          target: 'http://localhost:8080',
          changeOrigin: true
        }
      }
    }
  }
})
