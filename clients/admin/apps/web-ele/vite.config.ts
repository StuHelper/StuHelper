import path from 'node:path';

import { defineConfig } from '@vben/vite-config';

import ElementPlus from 'unplugin-element-plus/vite';

const devProxyTarget = process.env.VITE_DEV_PROXY_TARGET || 'http://localhost:8080';

export default defineConfig(async () => {
  return {
    application: {},
    vite: {
      plugins: [
        ElementPlus({
          format: 'esm',
        }),
      ],
      resolve: {
        alias: {
          '@stuhelper/shared': path.resolve(__dirname, '../../../shared/src'),
        },
      },
      server: {
        proxy: {
          '/api': {
            changeOrigin: true,
            // Go 后端
            target: devProxyTarget,
            ws: true,
          },
        },
      },
    },
  };
});
