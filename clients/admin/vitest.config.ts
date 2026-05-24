import { fileURLToPath, URL } from 'node:url';

import Vue from '@vitejs/plugin-vue';
import VueJsx from '@vitejs/plugin-vue-jsx';
import { configDefaults, defineConfig } from 'vitest/config';

export default defineConfig({
  plugins: [Vue(), VueJsx()],
  resolve: {
    alias: [
      {
        find: /^@stuhelper\/shared\/api$/,
        replacement: fileURLToPath(
          new URL('../shared/src/api/index.ts', import.meta.url),
        ),
      },
      {
        find: /^@stuhelper\/shared\/constants$/,
        replacement: fileURLToPath(
          new URL('../shared/src/constants/index.ts', import.meta.url),
        ),
      },
      {
        find: /^@stuhelper\/shared\/types$/,
        replacement: fileURLToPath(
          new URL('../shared/src/types/index.ts', import.meta.url),
        ),
      },
    ],
  },
  test: {
    environment: 'happy-dom',
    environmentOptions: {
      happyDOM: {
        settings: {
          // happy-dom v20+ disables JS evaluation by default (security fix).
          // Treat disabled script loading as success to preserve test behavior.
          handleDisabledFileLoadingAsSuccess: true,
        },
      },
    },
    exclude: [
      ...configDefaults.exclude,
      '**/e2e/**',
      '**/dist/**',
      '**/.{idea,git,cache,output,temp}/**',
      '**/node_modules/**',
      '**/{stylelint,eslint}.config.*',
      '**/{oxfmt,oxlint}.config.*',
    ],
  },
});
