import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { dirname, join } from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

const currentDir = dirname(fileURLToPath(import.meta.url))
const workspaceRoot = join(currentDir, '../../..')

const explicitPluginKeys = [
  'stuhelper-binding:',
  'stuhelper-group-guard:',
  'stuhelper-admin:',
] as const

test('P5 loads split StuHelper plugins explicitly from koishi.yml', async () => {
  const config = await readWorkspaceFile('koishi.yml')

  assert.match(config, /\n\s+stuhelper-core:/, 'koishi.yml must keep stuhelper-core enabled.')
  for (const pluginKey of explicitPluginKeys) {
    assert.match(config, new RegExp(`\\n\\s+${pluginKey}`), `koishi.yml must load ${pluginKey}.`)
    assert.match(
      config,
      new RegExp(`${pluginKey}[\\s\\S]*?platform:\\n\\s+baseUrl: \\$\\{\\{ env\\.STUHELPER_PLATFORM_BASE_URL \\}\\}`),
      `${pluginKey} must receive platform.baseUrl from the shared environment variable.`,
    )
    assert.match(
      config,
      new RegExp(`${pluginKey}[\\s\\S]*?serviceToken: \\$\\{\\{ env\\.STUHELPER_PLATFORM_SERVICE_TOKEN \\}\\}`),
      `${pluginKey} must receive platform.serviceToken from the shared environment variable.`,
    )
  }
  assert.doesNotMatch(config, /&platform_config|\*platform_config/, 'P5 must not use YAML anchors.')
})

test('koishi.yml keeps admission MVP production-safe defaults', async () => {
  const config = await readWorkspaceFile('koishi.yml')
  const guardBlock = extractYamlBlock(config, 'stuhelper-group-guard:')

  assert.match(guardBlock, /targetGroups:\s*\n\s+- '178037297'/)
  assert.match(guardBlock, /scheduler:\s*\n\s+fallbackScanEnabled: true\s*\n\s+scanIntervalSeconds: 300/)
  assert.match(guardBlock, /actionStream:\s*\n\s+enabled: true\s*\n\s+reconnectDelaySeconds: 5/)
  assert.match(guardBlock, /commands:\s*\n\s+enabled: false/)
  assert.match(guardBlock, /admissionCommands:\s*\n\s+enabled: true/)
  assert.match(guardBlock, /minAuthority: 4/)
  assert.match(guardBlock, /moderation:\s*\n\s+enabled: false/)
  assert.match(guardBlock, /freshmanForward:\s*\n\s+enabled: false/)
})

test('P6 removes old wrapper setup from stuhelper-core', async () => {
  const entry = await readWorkspaceFile('plugins/stuhelper-core/src/index.ts')
  const setupPath = 'plugins/stuhelper-core/src/setup/register-' + 'legacy-plugins.ts'
  const wrapperPath = 'plugins/stuhelper-core/src/legacy/' + 'legacy-' + 'wrapper.ts'

  assert.doesNotMatch(entry, new RegExp('register' + 'Legacy' + 'Plugins'))
  await assert.rejects(readWorkspaceFile(setupPath), /ENOENT/)
  await assert.rejects(readWorkspaceFile(wrapperPath), /ENOENT/)
})

test('P6 core config schema only keeps fields used by core runtime', async () => {
  const schema = await readWorkspaceFile('packages/shared/src/config/index.ts')
  const types = await readWorkspaceFile('packages/shared/src/types/index.ts')
  const coreSchemaBody = extractCoreBlock(schema, 'createCoreConfigSchema')
  const coreTypeBody = extractCoreBlock(types, 'StuhelperCoreConfig')

  assert.match(coreSchemaBody, /platform: createPlatformConfigSchema\(\)/)
  assert.match(coreSchemaBody, /guard: createGuardConfigSchema\(\)/)
  assert.match(coreSchemaBody, /console: createConsoleConfigSchema\(\)/)
  assert.match(coreSchemaBody, /runtimeModules: Schema\.object/)
  assert.doesNotMatch(coreSchemaBody, /binding|admin|scheduler|moderation|fun|ai/)
  assert.match(coreTypeBody, /platform: StuhelperPlatformConfig/)
  assert.match(coreTypeBody, /guard: StuhelperGuardConfig/)
  assert.match(coreTypeBody, /console: StuhelperConsoleConfig/)
  assert.match(coreTypeBody, /runtimeModules\?:/)
  assert.doesNotMatch(coreTypeBody, /binding|admin|scheduler|moderation|fun|ai/)
})

test('README documents admission scopes and backend policy boundary', async () => {
  const readme = await readWorkspaceFile('README.md')

  for (const scope of [
    'bot.admission.session',
    'bot.admission.event',
    'bot.admission.review',
    'bot.admission.forward',
  ]) {
    assert.match(readme, new RegExp(scope), `README must document ${scope}.`)
  }
  assert.match(readme, /Admission 策略边界/)
  assert.match(readme, /后端 admission policy/)
  assert.match(readme, /`koishi\.yml` 的插件加载保持不变/)
})

async function readWorkspaceFile(relativePath: string) {
  return readFile(join(workspaceRoot, relativePath), 'utf8')
}

function extractCoreBlock(content: string, name: string) {
  const match = content.match(new RegExp(`${name}[\\s\\S]*?\\{([\\s\\S]*?)\\n\\}`))
  assert.ok(match, `${name} block not found`)
  return match[1]
}

function extractYamlBlock(content: string, key: string) {
  const match = content.match(new RegExp(`\\n(\\s+${key}[^\\n]*\\n[\\s\\S]*?)(?=\\n\\s{4}\\S|\\n\\s{2}\\S|$)`))
  assert.ok(match, `${key} block not found`)
  return match[1]
}
