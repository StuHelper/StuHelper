import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

interface WorkspaceManifest {
  main?: string
  type?: string
  types?: string
  typings?: string
}

const currentDir = dirname(fileURLToPath(import.meta.url))
const workspaceRoot = join(currentDir, '../../..')

test('Koishi 本地工作区包使用构建产物作为运行时入口', async () => {
  await assertPackageRuntimeEntry('packages/shared/package.json')
  await assertPackageRuntimeEntry('plugins/stuhelper-admin/package.json')
  await assertPackageRuntimeEntry('plugins/stuhelper-binding/package.json')
  await assertPackageRuntimeEntry('plugins/stuhelper-console/package.json')
  await assertPackageRuntimeEntry('plugins/stuhelper-core/package.json')
  await assertPackageRuntimeEntry('plugins/stuhelper-group-guard/package.json')
  await assertKoishiRuntimeConfig('koishi.yml')
})

async function assertPackageRuntimeEntry(relativePath: string) {
  const manifest = await readManifest(relativePath)

  assert.equal(
    manifest.main,
    'lib/index.js',
    `${relativePath} 必须把运行时入口指向构建产物`,
  )
  assert.equal(
    manifest.types ?? manifest.typings,
    'lib/index.d.ts',
    `${relativePath} 必须把类型入口指向构建产物`,
  )
  assert.notEqual(
    manifest.type,
    'module',
    `${relativePath} 不应依赖源码态 ESM 入口`,
  )
}

async function readManifest(relativePath: string) {
  const manifestPath = join(workspaceRoot, relativePath)
  const content = await readFile(manifestPath, 'utf8')
  return JSON.parse(content) as WorkspaceManifest
}

async function assertKoishiRuntimeConfig(relativePath: string) {
  const configPath = join(workspaceRoot, relativePath)
  const content = await readFile(configPath, 'utf8')

  assert.match(content, /\n\s+port: 5140\n/, `${relativePath} 必须固定监听 5140 端口。`)
  assert.match(content, /\n\s+maxPort: 5140\n/, `${relativePath} 不应自动漂移到其他端口。`)
  assert.doesNotMatch(content, /\n\s+~stuhelper-core:/, `${relativePath} 不应禁用 stuhelper-core。`)
}
