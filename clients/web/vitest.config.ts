import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { createRequire } from 'node:module'
import { dirname, join } from 'node:path'
import { fileURLToPath, URL } from 'node:url'

const sharedSrc = fileURLToPath(new URL('../shared/src', import.meta.url))
const require = createRequire(import.meta.url)
const vueEntry = require.resolve('vue/dist/vue.esm-bundler.js')
const vuePackageJSON = require.resolve('vue/package.json')
const vueRequire = createRequire(vuePackageJSON)
const runtimeDomEntry = vueRequire.resolve('@vue/runtime-dom/dist/runtime-dom.esm-bundler.js')
const runtimeDomPackageJSON = vueRequire.resolve('@vue/runtime-dom/package.json')
const runtimeDomRequire = createRequire(runtimeDomPackageJSON)
const vueTestUtilsEntry = join(
  dirname(require.resolve('@vue/test-utils/package.json')),
  'dist/vue-test-utils.esm-bundler.mjs',
)

function resolveVueRuntimeEntry(packageName: string, entry: string) {
  return runtimeDomRequire.resolve(`${packageName}/dist/${entry}`)
}

export default defineConfig({
  plugins: [vue()],
  resolve: {
    dedupe: ['vue', '@vue/runtime-core', '@vue/runtime-dom', '@vue/reactivity', '@vue/shared'],
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      '@stuhelper/shared': sharedSrc,
      vue: vueEntry,
      '@vue/runtime-dom': runtimeDomEntry,
      '@vue/runtime-core': resolveVueRuntimeEntry('@vue/runtime-core', 'runtime-core.esm-bundler.js'),
      '@vue/reactivity': resolveVueRuntimeEntry('@vue/reactivity', 'reactivity.esm-bundler.js'),
      '@vue/shared': resolveVueRuntimeEntry('@vue/shared', 'shared.esm-bundler.js'),
      '@vue/test-utils': vueTestUtilsEntry,
    }
  },
  test: {
    environment: 'node',
    globals: true,
    exclude: ['tests/e2e/**/*', 'node_modules/**/*'],
    server: {
      deps: {
        inline: ['@vue/test-utils'],
      },
    },
    coverage: {
      provider: 'v8',
      reporter: ['text', 'html']
    }
  }
})
