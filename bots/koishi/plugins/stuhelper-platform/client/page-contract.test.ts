import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

import {
  compileScript,
  compileTemplate,
  parse,
} from '@vue/compiler-sfc'

const COMPONENT_ID = 'stuhelper-platform-page'
const currentDir = dirname(fileURLToPath(import.meta.url))

test('platform WebUI page compiles as a Vue SFC', async () => {
  const source = await readFile(join(currentDir, 'page.vue'), 'utf8')
  const { descriptor, errors } = parse(source, { filename: 'page.vue' })

  assert.deepEqual(formatErrors(errors), [])
  assert.ok(descriptor.scriptSetup)

  const script = compileScript(descriptor, { id: COMPONENT_ID })
  const template = compileTemplate({
    id: COMPONENT_ID,
    source: descriptor.template?.content ?? '',
    filename: 'page.vue',
    compilerOptions: {
      bindingMetadata: script.bindings,
    },
  })

  assert.deepEqual(formatErrors(template.errors), [])
})

function formatErrors(errors: readonly unknown[]): string[] {
  return errors.map((error) => error instanceof Error ? error.message : String(error))
}
