import type { IncomingMessage, ServerResponse } from 'node:http';
import type { Plugin } from 'vite';

import process from 'node:process';

import { defineConfig } from '@vben/vite-config';

import ElementPlus from 'unplugin-element-plus/vite';

const devProxyTarget =
  process.env.VITE_DEV_PROXY_TARGET || 'http://localhost:8080';
const e2eAPIStubEnabled = process.env.VITE_E2E_API_STUB === '1';

function e2eAPIStubPlugin(): Plugin {
  return {
    name: 'stuhelper-admin-e2e-api-stub',
    configureServer(server) {
      server.middlewares.use(
        (req: IncomingMessage, res: ServerResponse, next) => {
          const requestURL = req.url ?? '';
          if (!requestURL.startsWith('/api/')) {
            next();
            return;
          }

          res.statusCode = 200;
          res.setHeader('Content-Type', 'application/json');
          res.end(JSON.stringify({ success: true, data: null }));
        },
      );
    },
  };
}

export default defineConfig(async () => {
  return {
    application: {},
    vite: {
      plugins: [
        ...(e2eAPIStubEnabled ? [e2eAPIStubPlugin()] : []),
        ElementPlus({
          format: 'esm',
        }),
      ],
      server: {
        proxy: e2eAPIStubEnabled
          ? undefined
          : {
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
