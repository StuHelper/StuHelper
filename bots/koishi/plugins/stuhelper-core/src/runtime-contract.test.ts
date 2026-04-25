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
const repoRoot = join(workspaceRoot, '../..')

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
  assert.match(content, /\n\s+auth:[^\n]*:\n/, `${relativePath} 必须启用 @koishijs\/plugin-auth。`)
  assert.match(content, /password:\s*\$\{\{\s*env\.STUHELPER_CONSOLE_ADMIN_PASSWORD\s*\}\}/, `${relativePath} 必须通过环境变量提供控制台管理员密码。`)
}

test('stuhelper-core 控制台 API 注册必须依赖 auth 服务', async () => {
  const sourcePath = join(workspaceRoot, 'plugins/stuhelper-core/src/index.ts')
  const content = await readFile(sourcePath, 'utf8')

  assert.match(
    content,
    /ctx\.inject\(\['console', 'database', 'stuhelperGroupCenter', 'auth'\]/,
    'registerConsoleAPI 必须在 auth 服务存在时才注册控制台 API。',
  )
  assert.match(
    content,
    /validateConsoleAdminPassword\(process\.env\.STUHELPER_CONSOLE_ADMIN_PASSWORD\)/,
    'registerConsoleAPI 必须在启动期显式校验控制台管理员密码。',
  )
})

test('Koishi 控制台管理员密码必须写入环境样板和入口文档', async () => {
  await assertContainsConsoleAdminPassword(join(repoRoot, '.env.example'))
  await assertContainsConsoleAdminPassword(join(repoRoot, '.env.prod.example'))
  await assertContainsConsoleAdminPassword(join(workspaceRoot, 'README.md'))
  await assertContainsConsoleAdminPassword(join(repoRoot, 'docs/QUICKSTART.md'))
  await assertContainsConsoleAdminPassword(join(repoRoot, 'docs/guides/koishi-development.md'))
})

async function assertContainsConsoleAdminPassword(filePath: string) {
  const content = await readFile(filePath, 'utf8')

  assert.match(
    content,
    /STUHELPER_CONSOLE_ADMIN_PASSWORD/,
    `${filePath} 必须说明 STUHELPER_CONSOLE_ADMIN_PASSWORD。`,
  )
}

test('stuhelper-core 不应通过覆盖 ctx.console.addListener 注入 authority', async () => {
  const sourcePath = join(workspaceRoot, 'plugins/stuhelper-core/src/core/api/index.ts')
  const content = await readFile(sourcePath, 'utf8')

  assert.doesNotMatch(
    content,
    /ctx\.console\.addListener\s*=/,
    'registerWebSocketAPI 不应再通过 monkey-patch 覆盖 ctx.console.addListener。',
  )
})
