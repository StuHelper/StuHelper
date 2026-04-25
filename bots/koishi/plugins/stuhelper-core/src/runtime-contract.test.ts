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
  await assertPackageRuntimeEntry('plugins/stuhelper-core/package.json')
  await assertPackageRuntimeEntry('plugins/stuhelper-group-guard/package.json')
  await assertKoishiRuntimeConfig('koishi.yml')
})

test('stuhelper-core 入口只表达装配顺序', async () => {
  const content = await readWorkspaceFile('plugins/stuhelper-core/src/index.ts')

  assert.ok(
    content.split(/\r?\n/).length <= 50,
    'src/index.ts 应保持在 50 行以内，只表达装配顺序。',
  )
  assert.match(
    content,
    /export function apply\(ctx: Context, config: Config\) {\s+registerCoreService\(ctx\)\s+registerConsoleEntry\(ctx\)\s+registerConsoleApi\(ctx, config\)\s+registerBackgroundJobs\(ctx\)\s+registerRuntimeModules\(ctx\)\s+registerLegacyPlugins\(ctx, config\)\s+}/,
    'apply() 必须保持 P3 约定的装配顺序。',
  )
})

test('stuhelper-core 装配函数保持统一签名', async () => {
  const setupFunctions = [
    ['setup/register-core-service.ts', 'registerCoreService'],
    ['setup/register-console-entry.ts', 'registerConsoleEntry'],
    ['setup/register-console-api.ts', 'registerConsoleApi'],
    ['setup/register-background-jobs.ts', 'registerBackgroundJobs'],
    ['setup/register-runtime-modules.ts', 'registerRuntimeModules'],
    ['setup/register-legacy-plugins.ts', 'registerLegacyPlugins'],
  ]

  for (const [relativePath, functionName] of setupFunctions) {
    const content = await readWorkspaceFile(`plugins/stuhelper-core/src/${relativePath}`)
    assert.match(
      content,
      new RegExp(`export function ${functionName}\\(ctx: Context, _?config\\?: Config\\)`),
      `${relativePath} 必须导出统一的装配函数签名。`,
    )
  }
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
  const content = await readWorkspaceFile(relativePath)
  return JSON.parse(content) as WorkspaceManifest
}

async function assertKoishiRuntimeConfig(relativePath: string) {
  const content = await readWorkspaceFile(relativePath)

  assert.match(content, /\n\s+port: 5140\n/, `${relativePath} 必须固定监听 5140 端口。`)
  assert.match(content, /\n\s+maxPort: 5140\n/, `${relativePath} 不应自动漂移到其他端口。`)
  assert.doesNotMatch(content, /\n\s+~stuhelper-core:/, `${relativePath} 不应禁用 stuhelper-core。`)
  assert.match(content, /\n\s+auth:[^\n]*:\n/, `${relativePath} 必须启用 @koishijs\/plugin-auth。`)
  assert.match(content, /password:\s*\$\{\{\s*env\.STUHELPER_CONSOLE_ADMIN_PASSWORD\s*\}\}/, `${relativePath} 必须通过环境变量提供控制台管理员密码。`)
}

async function readWorkspaceFile(relativePath: string) {
  return readFile(join(workspaceRoot, relativePath), 'utf8')
}

test('stuhelper-core 控制台密码校验必须与 API 注册共享失败边界', async () => {
  const api = await readWorkspaceFile('plugins/stuhelper-core/src/setup/register-console-api.ts')

  assert.match(
    api,
    /ctx\.inject\(\['console', 'database', 'stuhelperGroupCenter', 'auth'\]/,
    'registerConsoleAPI 必须在 auth 服务存在时才注册控制台 API。',
  )
  assert.match(
    api,
    /validateConsoleAdminPassword\(process\.env\.STUHELPER_CONSOLE_ADMIN_PASSWORD\)/,
    'registerConsoleAPI 必须在启动期显式校验控制台管理员密码。',
  )
  assert.ok(
    api.indexOf('validateConsoleAdminPassword(process.env.STUHELPER_CONSOLE_ADMIN_PASSWORD)') <
      api.indexOf('registerWebSocketAPI(apiCtx, apiCtx.stuhelperGroupCenter)'),
    '控制台管理员密码校验必须先于所有 Console API 注册，保持 fail-closed 行为。',
  )
})

test('stuhelper-core 后台任务和运行时模块保持 database/service 注入路径', async () => {
  const background = await readWorkspaceFile('plugins/stuhelper-core/src/setup/register-background-jobs.ts')
  const runtime = await readWorkspaceFile('plugins/stuhelper-core/src/setup/register-runtime-modules.ts')

  assert.match(
    background,
    /ctx\.inject\(\['database', 'stuhelperGroupCenter'\]/,
    'registerBackgroundJobs 必须保持 database + stuhelperGroupCenter 注入集合。',
  )
  assert.match(
    background,
    /registerReviewClaimRecovery\(moduleCtx, new ModerationStore\(moduleCtx\)\)/,
    'registerBackgroundJobs 必须保留 review claim recovery 注册。',
  )
  assert.match(
    runtime,
    /ctx\.inject\(\['database', 'stuhelperGroupCenter'\]/,
    'registerRuntimeModules 必须保持 database + stuhelperGroupCenter 注入集合。',
  )
  assert.match(
    runtime,
    /moduleCtx\.on\('ready', async \(\) => \{/,
    'registerRuntimeModules 必须继续在 ready 事件里初始化模块。',
  )
})

test('stuhelper-core 运行时模块注册顺序不变', async () => {
  const runtime = await readWorkspaceFile('plugins/stuhelper-core/src/setup/register-runtime-modules.ts')
  const match = runtime.match(/export const MODULE_CLASSES: ModuleClass\[] = \[([\s\S]*?)\]/)
  const moduleBody = match?.[1] ?? ''
  const modules = Array.from(
    moduleBody.matchAll(/^\s+([A-Za-z0-9]+Module|crossGroupModule)(?:\s+as.*)?,/gm),
    ([, moduleName]) => moduleName,
  )

  assert.deepEqual(modules, [
    'WarnModule',
    'KeywordModule',
    'MemberManageModule',
    'MessageManageModule',
    'OrderManageModule',
    'AntirepeatModule',
    'WelcomeModule',
    'RepeatModule',
    'DiceModule',
    'BanmeModule',
    'AntiRecallModule',
    'AIModule',
    'ConfigModule',
    'LogModule',
    'SubscriptionModule',
    'HelpModule',
    'ReportModule',
    'GetAuthModule',
    'AuthModule',
    'EventModule',
    'StatusModule',
    'crossGroupModule',
  ])
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
