import assert from 'node:assert/strict'
import { access, readFile } from 'node:fs/promises'
import { constants } from 'node:fs'
import { dirname, join } from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

interface CorePackageManifest {
  files?: string[]
  dependencies?: Record<string, string>
  koishi?: {
    browser?: boolean
  }
}

const currentDir = dirname(fileURLToPath(import.meta.url))
const pluginRoot = join(currentDir, '..')

test('stuhelper-core 迁移后必须承载新的浏览器群管中心入口', async () => {
  const manifest = await readPackageManifest()

  assert.equal(
    manifest.koishi?.browser,
    true,
    'stuhelper-core 必须声明 browser 插件能力以承载新的群管中心 UI',
  )
  assert.ok(
    manifest.files?.includes('dist'),
    'stuhelper-core 必须分发 dist 构建产物',
  )
  assert.ok(
    manifest.dependencies?.['@koishijs/client'],
    'stuhelper-core 必须显式依赖 @koishijs/client',
  )
  assert.ok(
    manifest.dependencies?.['@koishijs/console'],
    'stuhelper-core 必须显式依赖 @koishijs/console',
  )

  await assertFileExists('client/index.ts')
  await assertFileExists('client/pages/index.vue')

  const clientEntry = await readFile(join(pluginRoot, 'client/index.ts'), 'utf8')
  assert.match(clientEntry, /StuHelper 群管中心/, '客户端页面名称必须改为 StuHelper 群管中心')
  assert.doesNotMatch(clientEntry, /grouphelper/i, '客户端入口不应再暴露 grouphelper 品牌')
})

async function readPackageManifest() {
  const manifestPath = join(pluginRoot, 'package.json')
  const content = await readFile(manifestPath, 'utf8')
  return JSON.parse(content) as CorePackageManifest
}

async function assertFileExists(relativePath: string) {
  await access(join(pluginRoot, relativePath), constants.F_OK)
}
