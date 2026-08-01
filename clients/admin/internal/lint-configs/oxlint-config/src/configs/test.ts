import type { OxlintConfig } from 'oxlint';

const test: OxlintConfig = {
  rules: {
    'jest/no-conditional-expect': 'off',
    'jest/require-to-throw-message': 'off',
    'vitest/consistent-test-it': [
      'error',
      {
        fn: 'it',
        withinDescribe: 'it',
      },
    ],
    'vitest/hoisted-apis-on-top': 'off',
    // Keep component names readable in top-level describe blocks and preserve
    // the established Vitest style. These rules became active in Oxlint 1.75
    // and require a separate, repository-wide policy migration.
    'vitest/no-conditional-expect': 'off',
    'vitest/no-focused-tests': 'error',
    'vitest/no-identical-title': 'error',
    'vitest/no-import-node-test': 'error',
    'vitest/prefer-hooks-in-order': 'error',
    'vitest/prefer-lowercase-title': 'off',
    'vitest/require-mock-type-parameters': 'off',
    'vitest/require-to-throw-message': 'off',
    'vitest/valid-expect': 'off',
  },
};

export { test };
