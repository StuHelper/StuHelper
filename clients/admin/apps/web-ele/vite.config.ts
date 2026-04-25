import process from 'node:process';

import { defineConfig } from '@vben/vite-config';

import ElementPlus from 'unplugin-element-plus/vite';

const devProxyTarget =
  process.env.VITE_DEV_PROXY_TARGET || 'http://localhost:8080';

export default defineConfig(async () => {
  return {
    application: {},
    vite: {
      plugins: [
        ElementPlus({
          format: 'esm',
        }),
      ],
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
