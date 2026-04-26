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
const legacyBaseClassName = 'Base' + 'Module'
const legacyAdapterPath = 'base-module-' + 'adapter'
const legacyModuleTypeKey = 'Module' + 'Type'
const legacyBasePath = 'base' + '.module'

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
    /export function apply\(ctx: Context, config: Config\) {\s+registerCoreService\(ctx\)\s+registerConsoleEntry\(ctx\)\s+registerConsoleApi\(ctx, config\)\s+registerBackgroundJobs\(ctx\)\s+registerRuntimeModules\(ctx\)\s+}/,
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

function legacyAdapterPattern(id: string, className: string): RegExp {
  return new RegExp(`id: '${id}', ${legacyModuleTypeKey}: ${className}`)
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
  assert.doesNotMatch(
    runtime,
    new RegExp(`new ${legacyModuleTypeKey}\\(`),
    'registerRuntimeModules 不应再直接 new 旧模块类，必须从 runtime registry 装配。',
  )
})

test('stuhelper-core 运行时模块注册顺序不变', async () => {
  const registry = await readWorkspaceFile('plugins/stuhelper-core/src/runtime/registry.ts')
  const match = registry.match(/const MODULE_REGISTRATIONS:[^=]+ = \[([\s\S]*?)\]/)
  const moduleBody = match?.[1] ?? ''
  const modules = Array.from(
    moduleBody.matchAll(/(helpRuntimeModule|diceRuntimeModule|banmeRuntimeModule|configRuntimeModule|memberManageRuntimeModule|messageManageRuntimeModule|orderManageRuntimeModule|keywordRuntimeModule|aiRuntimeModule|warnRuntimeModule|reportRuntimeModule|getauthRuntimeModule|authRuntimeModule|crossGroupRuntimeModule|antirepeatRuntimeModule|welcomeRuntimeModule|repeatRuntimeModule|logRuntimeModule|antirecallRuntimeModule|subscriptionRuntimeModule|eventRuntimeModule|statusRuntimeModule)/g),
    ([, runtimeModule]) => toModuleClassName(runtimeModule),
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

test('P4 原生 RuntimeModule 收尾后不保留旧 adapter 兼容层', async () => {
  const registry = await readWorkspaceFile('plugins/stuhelper-core/src/runtime/registry.ts')
  const modulesIndex = await readWorkspaceFile('plugins/stuhelper-core/src/core/modules/index.ts')
  const registryLegacyPattern = new RegExp(`${legacyAdapterPath}|adapt${legacyBaseClassName}|${legacyModuleTypeKey}`)
  const moduleIndexLegacyPattern = new RegExp(`${legacyBaseClassName}|${legacyBasePath}`)

  assert.doesNotMatch(registry, registryLegacyPattern)
  assert.doesNotMatch(modulesIndex, moduleIndexLegacyPattern)
  await assert.rejects(readWorkspaceFile(`plugins/stuhelper-core/src/core/modules/${legacyBasePath}.ts`), /ENOENT/)
  await assert.rejects(readWorkspaceFile(`plugins/stuhelper-core/src/runtime/${legacyAdapterPath}.ts`), /ENOENT/)
})

const nativeModuleContracts = [
  ['P4b-1', 'HelpModule', 'help.module.ts', legacyAdapterPattern('help', 'HelpModule')],
  ['P4b-2', 'DiceModule', 'dice.module.ts', legacyAdapterPattern('dice', 'DiceModule')],
  ['P4b-3', 'BanmeModule', 'banme.module.ts', legacyAdapterPattern('banme', 'BanmeModule')],
  ['P4b-4', 'ConfigModule', 'config.module.ts', legacyAdapterPattern('config', 'ConfigModule')],
  ['P4b-5', 'StatusModule', 'status.module.ts', legacyAdapterPattern('status', 'StatusModule')],
  ['P4b-6', 'EventModule', 'event.module.ts', legacyAdapterPattern('event', 'EventModule')],
  ['P4b-7', 'AntirepeatModule', 'antirepeat.module.ts', legacyAdapterPattern('antirepeat', 'AntirepeatModule')],
  ['P4b-8', 'MessageManageModule', 'messageManage.module.ts', legacyAdapterPattern('manage-message', 'MessageManageModule')],
  ['P4b-9', 'GetAuthModule', 'getauth.module.ts', legacyAdapterPattern('getauth', 'GetAuthModule')],
  ['P4b-10', 'crossGroupModule', 'crossGroupManage.module.ts', legacyAdapterPattern('manage-cross-group', 'crossGroupModule')],
  ['P4b-11', 'AuthModule', 'auth.module.ts', legacyAdapterPattern('auth', 'AuthModule')],
  ['P4b-12', 'RepeatModule', 'repeat.module.ts', legacyAdapterPattern('repeat', 'RepeatModule')],
  ['P4b-13', 'WelcomeModule', 'welcome.module.ts', legacyAdapterPattern('welcome', 'WelcomeModule')],
  ['P4b-14', 'LogModule', 'log.module.ts', legacyAdapterPattern('log', 'LogModule')],
  ['P4b-15', 'AntiRecallModule', 'antirecall.module.ts', legacyAdapterPattern('antirecall', 'AntiRecallModule')],
  ['P4b-16', 'SubscriptionModule', 'subscription.module.ts', legacyAdapterPattern('subscription', 'SubscriptionModule')],
  ['P4b-17', 'MemberManageModule', 'memberManage.module.ts', legacyAdapterPattern('manage-member', 'MemberManageModule')],
  ['P4b-18', 'OrderManageModule', 'orderManage.module.ts', legacyAdapterPattern('manage-order', 'OrderManageModule')],
  ['P4b-19', 'KeywordModule', 'keyword.module.ts', legacyAdapterPattern('keyword', 'KeywordModule')],
  ['P4b-20', 'AIModule', 'ai.module.ts', legacyAdapterPattern('ai', 'AIModule')],
  ['P4b-21', 'WarnModule', 'warn.module.ts', legacyAdapterPattern('warn', 'WarnModule')],
  ['P4b-22', 'ReportModule', 'report.module.ts', legacyAdapterPattern('report', 'ReportModule')],
] as const

for (const [phase, className, fileName, adapterPattern] of nativeModuleContracts) {
  test(`${phase} ${className} must be native runtime module`, async () => {
    const moduleSource = await readWorkspaceFile(`plugins/stuhelper-core/src/core/modules/${fileName}`)
    const registry = await readWorkspaceFile('plugins/stuhelper-core/src/runtime/registry.ts')

    assert.doesNotMatch(
      moduleSource,
      new RegExp(`extends ${legacyBaseClassName}`),
      `${className} 已进入 ${phase}，不能继续继承旧模块基类。`,
    )
    assert.doesNotMatch(
      moduleSource,
      new RegExp(`from '\\./${legacyBasePath}'`),
      `${className} 已进入 ${phase}，不能继续依赖旧模块基类文件。`,
    )
    assert.doesNotMatch(
      registry,
      adapterPattern,
      `${className} 必须作为原生 RuntimeModule 注册，不能再通过旧 adapter 注册。`,
    )
  })
}

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

function toModuleClassName(runtimeModule: string): string {
  return runtimeModule
    .replace('helpRuntimeModule', 'HelpModule')
    .replace('diceRuntimeModule', 'DiceModule')
    .replace('banmeRuntimeModule', 'BanmeModule')
    .replace('configRuntimeModule', 'ConfigModule')
    .replace('memberManageRuntimeModule', 'MemberManageModule')
    .replace('messageManageRuntimeModule', 'MessageManageModule')
    .replace('orderManageRuntimeModule', 'OrderManageModule')
    .replace('keywordRuntimeModule', 'KeywordModule')
    .replace('aiRuntimeModule', 'AIModule')
    .replace('warnRuntimeModule', 'WarnModule')
    .replace('reportRuntimeModule', 'ReportModule')
    .replace('getauthRuntimeModule', 'GetAuthModule')
    .replace('authRuntimeModule', 'AuthModule')
    .replace('crossGroupRuntimeModule', 'crossGroupModule')
    .replace('antirepeatRuntimeModule', 'AntirepeatModule')
    .replace('welcomeRuntimeModule', 'WelcomeModule')
    .replace('repeatRuntimeModule', 'RepeatModule')
    .replace('logRuntimeModule', 'LogModule')
    .replace('antirecallRuntimeModule', 'AntiRecallModule')
    .replace('subscriptionRuntimeModule', 'SubscriptionModule')
    .replace('eventRuntimeModule', 'EventModule')
    .replace('statusRuntimeModule', 'StatusModule')
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
