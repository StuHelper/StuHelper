import { defineConfig } from 'vite'
import uni from '@dcloudio/vite-plugin-uni'

export default defineConfig({
  plugins: [uni()],
  resolve: {
    alias: {
      vue: '@dcloudio/uni-h5-vue',
      '@vue/server-renderer': '@dcloudio/uni-h5-vue/server-renderer',
    },
  },
})
