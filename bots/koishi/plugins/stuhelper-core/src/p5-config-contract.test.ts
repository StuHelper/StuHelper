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
  assert.doesNotMatch(coreSchemaBody, /binding|admin|scheduler|moderation|fun|ai/)
  assert.match(coreTypeBody, /platform: StuhelperPlatformConfig/)
  assert.match(coreTypeBody, /guard: StuhelperGuardConfig/)
  assert.match(coreTypeBody, /console: StuhelperConsoleConfig/)
  assert.doesNotMatch(coreTypeBody, /binding|admin|scheduler|moderation|fun|ai/)
})

async function readWorkspaceFile(relativePath: string) {
  return readFile(join(workspaceRoot, relativePath), 'utf8')
}

function extractCoreBlock(content: string, name: string) {
  const match = content.match(new RegExp(`${name}[\\s\\S]*?\\{([\\s\\S]*?)\\n\\}`))
  assert.ok(match, `${name} block not found`)
  return match[1]
}
