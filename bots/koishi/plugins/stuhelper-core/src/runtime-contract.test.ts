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
  assert.doesNotMatch(
    runtime,
    /new ModuleType\(/,
    'registerRuntimeModules 不应再直接 new BaseModule，必须从 runtime registry 装配。',
  )
})

test('stuhelper-core 运行时模块注册顺序不变', async () => {
  const registry = await readWorkspaceFile('plugins/stuhelper-core/src/runtime/registry.ts')
  const match = registry.match(/const MODULE_REGISTRATIONS: RuntimeModuleRegistration\[] = \[([\s\S]*?)\]/)
  const moduleBody = match?.[1] ?? ''
  const modules = Array.from(
    moduleBody.matchAll(/ModuleType:\s*([A-Za-z0-9]+Module|crossGroupModule)|(helpRuntimeModule|diceRuntimeModule|banmeRuntimeModule|configRuntimeModule|messageManageRuntimeModule|antirepeatRuntimeModule|eventRuntimeModule|statusRuntimeModule)/g),
    ([, moduleName, runtimeModule]) => moduleName ?? toModuleClassName(runtimeModule),
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

test('P4b-1 help module must be native runtime module', async () => {
  const helpModule = await readWorkspaceFile('plugins/stuhelper-core/src/core/modules/help.module.ts')
  const registry = await readWorkspaceFile('plugins/stuhelper-core/src/runtime/registry.ts')

  assert.doesNotMatch(
    helpModule,
    /extends BaseModule/,
    'HelpModule 已进入 P4b-1，不能继续继承 BaseModule。',
  )
  assert.doesNotMatch(
    helpModule,
    /from '\.\/base\.module'/,
    'HelpModule 已进入 P4b-1，不能继续依赖 BaseModule 文件。',
  )
  assert.doesNotMatch(
    registry,
    /id: 'help', ModuleType: HelpModule/,
    'HelpModule 必须作为原生 RuntimeModule 注册，不能再通过 BaseModule adapter 注册。',
  )
})

test('P4b-2 dice module must be native runtime module', async () => {
  const diceModule = await readWorkspaceFile('plugins/stuhelper-core/src/core/modules/dice.module.ts')
  const registry = await readWorkspaceFile('plugins/stuhelper-core/src/runtime/registry.ts')

  assert.doesNotMatch(
    diceModule,
    /extends BaseModule/,
    'DiceModule 已进入 P4b-2，不能继续继承 BaseModule。',
  )
  assert.doesNotMatch(
    diceModule,
    /from '\.\/base\.module'/,
    'DiceModule 已进入 P4b-2，不能继续依赖 BaseModule 文件。',
  )
  assert.doesNotMatch(
    registry,
    /id: 'dice', ModuleType: DiceModule/,
    'DiceModule 必须作为原生 RuntimeModule 注册，不能再通过 BaseModule adapter 注册。',
  )
})

test('P4b-3 banme module must be native runtime module', async () => {
  const banmeModule = await readWorkspaceFile('plugins/stuhelper-core/src/core/modules/banme.module.ts')
  const registry = await readWorkspaceFile('plugins/stuhelper-core/src/runtime/registry.ts')

  assert.doesNotMatch(
    banmeModule,
    /extends BaseModule/,
    'BanmeModule 已进入 P4b-3，不能继续继承 BaseModule。',
  )
  assert.doesNotMatch(
    banmeModule,
    /from '\.\/base\.module'/,
    'BanmeModule 已进入 P4b-3，不能继续依赖 BaseModule 文件。',
  )
  assert.doesNotMatch(
    registry,
    /id: 'banme', ModuleType: BanmeModule/,
    'BanmeModule 必须作为原生 RuntimeModule 注册，不能再通过 BaseModule adapter 注册。',
  )
})

test('P4b-4 config module must be native runtime module', async () => {
  const configModule = await readWorkspaceFile('plugins/stuhelper-core/src/core/modules/config.module.ts')
  const registry = await readWorkspaceFile('plugins/stuhelper-core/src/runtime/registry.ts')

  assert.doesNotMatch(
    configModule,
    /extends BaseModule/,
    'ConfigModule 已进入 P4b-4，不能继续继承 BaseModule。',
  )
  assert.doesNotMatch(
    configModule,
    /from '\.\/base\.module'/,
    'ConfigModule 已进入 P4b-4，不能继续依赖 BaseModule 文件。',
  )
  assert.doesNotMatch(
    registry,
    /id: 'config', ModuleType: ConfigModule/,
    'ConfigModule 必须作为原生 RuntimeModule 注册，不能再通过 BaseModule adapter 注册。',
  )
})

test('P4b-5 status module must be native runtime module', async () => {
  const statusModule = await readWorkspaceFile('plugins/stuhelper-core/src/core/modules/status.module.ts')
  const registry = await readWorkspaceFile('plugins/stuhelper-core/src/runtime/registry.ts')

  assert.doesNotMatch(
    statusModule,
    /extends BaseModule/,
    'StatusModule 已进入 P4b-5，不能继续继承 BaseModule。',
  )
  assert.doesNotMatch(
    statusModule,
    /from '\.\/base\.module'/,
    'StatusModule 已进入 P4b-5，不能继续依赖 BaseModule 文件。',
  )
  assert.doesNotMatch(
    registry,
    /id: 'status', ModuleType: StatusModule/,
    'StatusModule 必须作为原生 RuntimeModule 注册，不能再通过 BaseModule adapter 注册。',
  )
})

test('P4b-6 event module must be native runtime module', async () => {
  const eventModule = await readWorkspaceFile('plugins/stuhelper-core/src/core/modules/event.module.ts')
  const registry = await readWorkspaceFile('plugins/stuhelper-core/src/runtime/registry.ts')

  assert.doesNotMatch(
    eventModule,
    /extends BaseModule/,
    'EventModule 已进入 P4b-6，不能继续继承 BaseModule。',
  )
  assert.doesNotMatch(
    eventModule,
    /from '\.\/base\.module'/,
    'EventModule 已进入 P4b-6，不能继续依赖 BaseModule 文件。',
  )
  assert.doesNotMatch(
    registry,
    /id: 'event', ModuleType: EventModule/,
    'EventModule 必须作为原生 RuntimeModule 注册，不能再通过 BaseModule adapter 注册。',
  )
})

test('P4b-7 antirepeat module must be native runtime module', async () => {
  const antirepeatModule = await readWorkspaceFile('plugins/stuhelper-core/src/core/modules/antirepeat.module.ts')
  const registry = await readWorkspaceFile('plugins/stuhelper-core/src/runtime/registry.ts')

  assert.doesNotMatch(
    antirepeatModule,
    /extends BaseModule/,
    'AntirepeatModule 已进入 P4b-7，不能继续继承 BaseModule。',
  )
  assert.doesNotMatch(
    antirepeatModule,
    /from '\.\/base\.module'/,
    'AntirepeatModule 已进入 P4b-7，不能继续依赖 BaseModule 文件。',
  )
  assert.doesNotMatch(
    registry,
    /id: 'antirepeat', ModuleType: AntirepeatModule/,
    'AntirepeatModule 必须作为原生 RuntimeModule 注册，不能再通过 BaseModule adapter 注册。',
  )
})

test('P4b-8 message manage module must be native runtime module', async () => {
  const messageManageModule = await readWorkspaceFile('plugins/stuhelper-core/src/core/modules/messageManage.module.ts')
  const registry = await readWorkspaceFile('plugins/stuhelper-core/src/runtime/registry.ts')

  assert.doesNotMatch(
    messageManageModule,
    /extends BaseModule/,
    'MessageManageModule 已进入 P4b-8，不能继续继承 BaseModule。',
  )
  assert.doesNotMatch(
    messageManageModule,
    /from '\.\/base\.module'/,
    'MessageManageModule 已进入 P4b-8，不能继续依赖 BaseModule 文件。',
  )
  assert.doesNotMatch(
    registry,
    /id: 'manage-message', ModuleType: MessageManageModule/,
    'MessageManageModule 必须作为原生 RuntimeModule 注册，不能再通过 BaseModule adapter 注册。',
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

function toModuleClassName(runtimeModule: string): string {
  return runtimeModule
    .replace('helpRuntimeModule', 'HelpModule')
    .replace('diceRuntimeModule', 'DiceModule')
    .replace('banmeRuntimeModule', 'BanmeModule')
    .replace('configRuntimeModule', 'ConfigModule')
    .replace('messageManageRuntimeModule', 'MessageManageModule')
    .replace('antirepeatRuntimeModule', 'AntirepeatModule')
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
