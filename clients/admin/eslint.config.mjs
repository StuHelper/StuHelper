import { defineConfig } from '@vben/eslint-config';

import pluginVueA11y from 'eslint-plugin-vuejs-accessibility';

const applicationVueFiles = ['apps/web-ele/src/**/*.vue'];
const accessibilityConfigs = pluginVueA11y.configs['flat/recommended'].map(
  (config) => ({
    ...config,
    files: applicationVueFiles,
  }),
);

export default defineConfig([
  ...accessibilityConfigs,
  {
    files: applicationVueFiles,
    rules: {
      'vue/html-button-has-type': 'error',
      'vuejs-accessibility/label-has-for': [
        'error',
        {
          allowChildren: true,
          required: { some: ['nesting', 'id'] },
        },
      ],
      'vuejs-accessibility/no-static-element-interactions': 'off',
    },
  },
  {
    files: ['**/*.test.ts'],
    rules: {
      'vue/one-component-per-file': 'off',
    },
  },
]);
