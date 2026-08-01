import type { OxlintConfig } from 'oxlint';

const vue: OxlintConfig = {
  rules: {
    // Existing components intentionally reuse platform names such as Transition.
    'vue/no-reserved-component-names': 'off',
    'vue/prefer-import-from-vue': 'error',
  },
};

export { vue };
