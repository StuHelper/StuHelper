import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

interface PlatformManifest {
  main?: string
  types?: string
  typings?: string
  files?: string[]
  koishi?: {
    browser?: boolean
  }
}

const currentDir = dirname(fileURLToPath(import.meta.url))
const pluginRoot = join(currentDir, '..')
const workspaceRoot = join(pluginRoot, '../..')

test('platform package uses runtime build outputs and browser assets', async () => {
  const manifest = await readPackageManifest()

  assert.equal(manifest.main, 'lib/index.js')
  assert.equal(manifest.types ?? manifest.typings, 'lib/index.d.ts')
  assert.ok(manifest.files?.includes('lib'))
  assert.ok(manifest.files?.includes('dist'))
  assert.equal(manifest.koishi?.browser, true)
})

test('koishi config restores the StuHelper group center entry', async () => {
  const content = await readWorkspaceFile('koishi.yml')

  assert.match(content, /\n\s+stuhelper-core:[^\n]*: \{\}/)
  assert.doesNotMatch(content, /\n\s+stuhelper-platform:[^\n]*:/)
  assert.doesNotMatch(content, /\n\s+stuhelper-group-guard:[^\n]*:/)
  assert.doesNotMatch(content, /\n\s+stuhelper-console:[^\n]*:/)
})

test('platform plugin exposes no Koishi configuration schema', async () => {
  const content = await readFile(join(currentDir, 'index.ts'), 'utf8')

  assert.doesNotMatch(content, /export\s+const\s+Config/)
  assert.doesNotMatch(content, /Schema\./)
})

async function readPackageManifest(): Promise<PlatformManifest> {
  const content = await readFile(join(pluginRoot, 'package.json'), 'utf8')
  return JSON.parse(content) as PlatformManifest
}

async function readWorkspaceFile(relativePath: string): Promise<string> {
  return readFile(join(workspaceRoot, relativePath), 'utf8')
}
