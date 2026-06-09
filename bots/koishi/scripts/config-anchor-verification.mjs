import assert from 'node:assert/strict'
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { createRequire } from 'node:module'

const require = createRequire(import.meta.url)
const NodeLoader = require('@koishijs/loader').default

const EXPECTED_BASE_URL = 'https://p5a.platform.example.invalid'
const EXPECTED_SERVICE_TOKEN = 'p5a-service-token'
const UPDATED_BINDING_BASE_URL = 'https://p5a.binding-updated.example.invalid'
const TEMP_PREFIX = 'koishi-p5a-anchor-'

const GROUP_KEY = 'group:stuhelper'
const SHARED_KEY = '_shared'
const BINDING_KEY = 'stuhelper-binding:bind01'
const GROUP_GUARD_KEY = 'stuhelper-group-guard:guard1'
const ADMIN_KEY = 'stuhelper-admin:admin1'
const PLUGIN_KEYS = [BINDING_KEY, GROUP_GUARD_KEY, ADMIN_KEY]

const previousEnv = {
  baseUrl: process.env.STUHELPER_PLATFORM_BASE_URL,
  serviceToken: process.env.STUHELPER_PLATFORM_SERVICE_TOKEN,
}

process.env.STUHELPER_PLATFORM_BASE_URL = EXPECTED_BASE_URL
process.env.STUHELPER_PLATFORM_SERVICE_TOKEN = EXPECTED_SERVICE_TOKEN

const tempDir = await mkdtemp(join(tmpdir(), TEMP_PREFIX))
const configPath = join(tempDir, 'koishi.yml')

try {
  await writeFile(configPath, createAnchorConfig())

  const loader = new NodeLoader()
  await loader.init(configPath)
  const config = await loader.readConfig(false)

  assertPlatformLoaded(config.plugins[GROUP_KEY])
  assertRawAnchorIdentity(loader.config.plugins[GROUP_KEY])

  simulateConsolePluginReload(loader.config.plugins[GROUP_KEY])
  await loader.writeConfig(true)
  await waitForConfigWrite()

  const persisted = await readFile(configPath, 'utf8')
  assertConsolePersistenceBreaksNamedAnchor(persisted)

  console.log('P5a loading verification: PASS')
  console.log('P5a console persistence verification: FAILS named &platform_config preservation')
  console.log('P5 decision: use explicit duplicated env-derived platform config blocks')
} finally {
  restoreEnv()
  await rm(tempDir, { recursive: true, force: true })
}

function createAnchorConfig() {
  return [
    'plugins:',
    `  ${GROUP_KEY}:`,
    `    ${SHARED_KEY}: &platform_config`,
    '      baseUrl: ${{ env.STUHELPER_PLATFORM_BASE_URL }}',
    '      serviceToken: ${{ env.STUHELPER_PLATFORM_SERVICE_TOKEN }}',
    `    ${BINDING_KEY}:`,
    '      platform: *platform_config',
    `    ${GROUP_GUARD_KEY}:`,
    '      platform: *platform_config',
    `    ${ADMIN_KEY}:`,
    '      platform: *platform_config',
    '',
  ].join('\n')
}

function assertPlatformLoaded(groupConfig) {
  for (const key of PLUGIN_KEYS) {
    assert.equal(
      groupConfig[key].platform.baseUrl,
      EXPECTED_BASE_URL,
      `${key} did not receive the interpolated platform.baseUrl`,
    )
    assert.equal(
      groupConfig[key].platform.serviceToken,
      EXPECTED_SERVICE_TOKEN,
      `${key} did not receive the interpolated platform.serviceToken`,
    )
  }
}

function assertRawAnchorIdentity(groupConfig) {
  assert.equal(groupConfig[BINDING_KEY].platform, groupConfig[GROUP_GUARD_KEY].platform)
  assert.equal(groupConfig[BINDING_KEY].platform, groupConfig[ADMIN_KEY].platform)
}

function simulateConsolePluginReload(groupConfig) {
  const configPayload = JSON.parse(JSON.stringify(groupConfig[BINDING_KEY]))
  configPayload.platform.baseUrl = UPDATED_BINDING_BASE_URL
  groupConfig[BINDING_KEY] = configPayload
}

function assertConsolePersistenceBreaksNamedAnchor(persisted) {
  assert.doesNotMatch(
    persisted,
    /&platform_config|\*platform_config/,
    'Koishi config writer preserved the semantic platform_config anchor name unexpectedly.',
  )
  assert.match(
    persisted,
    new RegExp(`${BINDING_KEY}:[\\s\\S]*baseUrl: ${escapeRegExp(UPDATED_BINDING_BASE_URL)}`),
  )
  assert.match(persisted, new RegExp(`${GROUP_GUARD_KEY}:[\\s\\S]*platform: \\*`))
  assert.match(persisted, new RegExp(`${ADMIN_KEY}:[\\s\\S]*platform: \\*`))
}

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function waitForConfigWrite() {
  return new Promise((resolve) => setTimeout(resolve, 10))
}

function restoreEnv() {
  restoreEnvValue('STUHELPER_PLATFORM_BASE_URL', previousEnv.baseUrl)
  restoreEnvValue('STUHELPER_PLATFORM_SERVICE_TOKEN', previousEnv.serviceToken)
}

function restoreEnvValue(key, value) {
  if (value === undefined) {
    delete process.env[key]
    return
  }
  process.env[key] = value
}
