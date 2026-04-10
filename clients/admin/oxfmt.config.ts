import { defineConfig } from '@vben/oxfmt-config';

export default defineConfig({
  ignorePatterns: [
    'dist',
    'dev-dist',
    '.local',
    '.claude',
    '.agent',
    '.agents',
    '.codex',
    '.output.js',
    'node_modules',
    '.nvmrc',
    'coverage',
    'CODEOWNERS',
    '.nitro',
    '.output',
    '.pnpm-store',
    'playwright-report',
    'test-results',
    '**/*.svg',
    '**/*.sh',
    'public',
    '.npmrc',
    '*-lock.yaml',
    'skills-lock.json',
  ],
});
