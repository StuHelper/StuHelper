import { defineConfig } from '@vben/eslint-config';

export default defineConfig([
  {
    files: ['**/*.test.ts'],
    rules: {
      'vue/one-component-per-file': 'off',
    },
  },
]);
