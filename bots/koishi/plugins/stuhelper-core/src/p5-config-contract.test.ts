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

test('P5 legacy setup no longer loads split StuHelper plugins from core', async () => {
  const setup = await readWorkspaceFile('plugins/stuhelper-core/src/setup/register-legacy-plugins.ts')
  const wrapper = await readWorkspaceFile('plugins/stuhelper-core/src/legacy/legacy-wrapper.ts')

  assert.doesNotMatch(setup, /applyLegacyFeatures|ctx\.plugin|new Logger/)
  assert.doesNotMatch(wrapper, /ctx\.plugin|koishi-plugin-stuhelper-(binding|group-guard|admin)/)
})

async function readWorkspaceFile(relativePath: string) {
  return readFile(join(workspaceRoot, relativePath), 'utf8')
}
