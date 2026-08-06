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

test('koishi.yml keeps admission production-safe startup config without static guard targets', async () => {
  const config = await readWorkspaceFile('koishi.yml')
  const guardBlock = extractYamlBlock(config, 'stuhelper-group-guard:')

  assert.doesNotMatch(guardBlock, /\n\s+guard:\s*\n/)
  assert.doesNotMatch(guardBlock, /targetGroups|muteDurationSeconds|kickAfterMinutes|reminderTemplate|exemptUsers/)
  assert.match(guardBlock, /scheduler:\s*\n\s+scanIntervalSeconds: 300/)
  assert.match(guardBlock, /actionStream:\s*\n\s+reconnectDelaySeconds: 5/)
  assert.match(
    guardBlock,
    /alerting:\s*\n\s+enabled: \$\{\{ env\.STUHELPER_ALERTMANAGER_WEBHOOK_ENABLED === 'true' \}\}/,
    'Alertmanager enabled env must be converted to a boolean fail-closed instead of passing a string to Schema.boolean().',
  )
  assert.doesNotMatch(guardBlock, /admissionCommands|minAuthority|operatorQQIDs/)
  assert.doesNotMatch(guardBlock, /fallbackScanEnabled:/)
  assert.doesNotMatch(guardBlock, /actionStream:\s*\n\s+enabled:/)
  assert.doesNotMatch(guardBlock, /moderation:\s*\n\s+enabled:/)
  assert.doesNotMatch(guardBlock, /commands:\s*\n\s+enabled:/)
  assert.doesNotMatch(guardBlock, /freshmanForward:\s*\n\s+enabled:/)
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
  assert.doesNotMatch(coreSchemaBody, /console|createConsoleConfigSchema|runtimeModules/)
  assert.doesNotMatch(coreSchemaBody, /guard|createGuardConfigSchema|binding|admin|scheduler|moderation|fun|ai/)
  assert.match(coreTypeBody, /platform: StuhelperPlatformConfig/)
  assert.doesNotMatch(coreTypeBody, /console|StuhelperConsoleConfig|runtimeModules/)
  assert.doesNotMatch(coreTypeBody, /guard|StuhelperGuardConfig|binding|admin|scheduler|moderation|fun|ai/)
  assert.doesNotMatch(schema, /createConsoleConfigSchema/)
  assert.doesNotMatch(types, /StuhelperConsoleConfig/)
})

test('stuhelper-admin schema does not duplicate group-guard business config', async () => {
  const schema = await readWorkspaceFile('packages/shared/src/config/index.ts')
  const types = await readWorkspaceFile('packages/shared/src/types/index.ts')
  const runtimeSettings = await readWorkspaceFile('packages/shared/src/admin/runtime-settings.ts')
  const adminSchemaBody = extractCoreBlock(schema, 'createAdminPluginConfigSchema')
  const adminTypeBody = extractCoreBlock(types, 'StuhelperAdminPluginConfig')

  assert.match(adminSchemaBody, /platform: createPlatformConfigSchema\(\)/)
  assert.doesNotMatch(adminSchemaBody, /messages|createAdminMessageConfigSchema/)
  assert.doesNotMatch(adminSchemaBody, /admin: createAdminConfigSchema|moderation|fun|createModerationConfigSchema|createFunConfigSchema/)
  assert.match(adminTypeBody, /platform: StuhelperPlatformConfig/)
  assert.doesNotMatch(adminTypeBody, /messages\?:|StuhelperAdminMessageConfig/)
  assert.doesNotMatch(adminTypeBody, /admin: StuhelperAdminConfig|moderation|fun/)
  assert.match(runtimeSettings, /ADMIN_RUNTIME_SETTINGS_TABLE/)
  assert.match(runtimeSettings, /DEFAULT_ADMIN_RUNTIME_SETTINGS/)
  assert.match(runtimeSettings, /messages: DEFAULT_ADMIN_MESSAGES/)
})

test('stuhelper-binding schema does not duplicate runtime business config', async () => {
  const schema = await readWorkspaceFile('packages/shared/src/config/index.ts')
  const types = await readWorkspaceFile('packages/shared/src/types/index.ts')
  const runtimeSettings = await readWorkspaceFile('packages/shared/src/binding/runtime-settings.ts')
  const bindingSchemaBody = extractCoreBlock(schema, 'createBindingPluginConfigSchema')
  const bindingTypeBody = extractCoreBlock(types, 'StuhelperBindingPluginConfig')

  assert.match(bindingSchemaBody, /platform: createPlatformConfigSchema\(\)/)
  assert.doesNotMatch(bindingSchemaBody, /binding|createBindingConfigSchema|createBindingMessageConfigSchema|codeTtlMinutes|command:/)
  assert.match(bindingTypeBody, /platform: StuhelperPlatformConfig/)
  assert.doesNotMatch(bindingTypeBody, /binding|StuhelperBindingConfig|codeTtlMinutes/)
  assert.match(runtimeSettings, /BINDING_RUNTIME_SETTINGS_TABLE/)
  assert.match(runtimeSettings, /DEFAULT_BINDING_RUNTIME_SETTINGS/)
  assert.match(runtimeSettings, /command: DEFAULT_BINDING_COMMAND/)
  assert.match(runtimeSettings, /messages: DEFAULT_BINDING_MESSAGES/)
})

test('group-guard runtime switches are WebUI settings, not native plugin config', async () => {
  const schema = await readWorkspaceFile('packages/shared/src/config/index.ts')
  const types = await readWorkspaceFile('packages/shared/src/types/index.ts')
  const runtimeSettings = await readWorkspaceFile('packages/shared/src/guard/runtime-settings.ts')
  const aiSettings = await readWorkspaceFile('packages/shared/src/guard/ai-settings.ts')
  const behaviorSettings = await readWorkspaceFile('packages/shared/src/guard/behavior-settings.ts')
  const messageSettings = await readWorkspaceFile('packages/shared/src/guard/message-settings.ts')
  const keywordRuleAPI = await readWorkspaceFile('plugins/stuhelper-core/src/core/api/keyword-rule-api.ts')
  const groupGuardSchemaBody = extractCoreBlock(schema, 'createGroupGuardPluginConfigSchema')
  const groupGuardTypeBody = extractCoreBlock(types, 'StuhelperGroupGuardPluginConfig')

  assert.match(runtimeSettings, /DEFAULT_ADMISSION_RUNTIME_SETTINGS/)
  assert.match(runtimeSettings, /publicCommandsEnabled: false/)
  assert.match(runtimeSettings, /adminCommandsEnabled: true/)
  assert.match(runtimeSettings, /admissionCommandsEnabled: true/)
  assert.match(runtimeSettings, /actionStreamEnabled: true/)
  assert.match(runtimeSettings, /moderationEnabled: false/)
  assert.doesNotMatch(runtimeSettings, /freshmanForwardEnabled/)
  assert.match(runtimeSettings, /fallbackScanEnabled: true/)
  assert.match(runtimeSettings, /reminderGroupEnabled: true/)
  assert.match(runtimeSettings, /reminderDirectEnabled: false/)
  assert.match(aiSettings, /GROUP_GUARD_AI_SETTINGS_TABLE/)
  assert.match(aiSettings, /DEFAULT_GROUP_GUARD_AI_SETTINGS/)
  assert.match(aiSettings, /enabled: false/)
  assert.match(aiSettings, /apiKey: ''/)
  assert.match(behaviorSettings, /GROUP_GUARD_BEHAVIOR_SETTINGS_TABLE/)
  assert.match(behaviorSettings, /DEFAULT_GROUP_GUARD_FUN_SETTINGS/)
  assert.match(behaviorSettings, /DEFAULT_GROUP_GUARD_MODERATION_SETTINGS/)
  assert.match(behaviorSettings, /repeatThreshold: 3/)
  assert.match(behaviorSettings, /warningThresholdExpression: 'warnings >= 3'/)
  assert.match(behaviorSettings, /defaultMuteSeconds: 600/)
  assert.match(messageSettings, /GROUP_GUARD_MESSAGE_SETTINGS_TABLE/)
  assert.match(messageSettings, /DEFAULT_GROUP_GUARD_MESSAGE_SETTINGS/)
  assert.match(messageSettings, /messages: DEFAULT_GROUP_GUARD_MESSAGES/)
  assert.doesNotMatch(groupGuardSchemaBody, /guard: createGuardConfigSchema|commands: createCommandConfigSchema|fun: createFunConfigSchema|freshmanForward: createFreshmanForwardConfigSchema|admissionCommands|createAdmissionCommandConfigSchema|admissionReminderDelivery: createAdmissionReminderDeliveryConfigSchema/)
  assert.doesNotMatch(groupGuardSchemaBody, /ai: createAIConfigSchema|createAIConfigSchema/)
  assert.doesNotMatch(groupGuardSchemaBody, /messages|createGroupGuardMessageConfigSchema/)
  assert.doesNotMatch(groupGuardSchemaBody, /moderation|createModerationConfigSchema|keywordRules|createKeywordRuleConfigSchema/)
  assert.doesNotMatch(groupGuardSchemaBody, /targetGroups|muteDurationSeconds|kickAfterMinutes|reminderTemplate|exemptUsers/)
  assert.doesNotMatch(groupGuardSchemaBody, /fallbackScanEnabled|enabled: Schema\.boolean\(\)\.default\(true\)\.description\('是否启用后端 admission action SSE 下行流|enabled: Schema\.boolean\(\)\.default\(true\)\.description\('是否启用消息风控监听/)
  assert.doesNotMatch(groupGuardSchemaBody, /enabled: Schema\.boolean\(\)\.default\(true\)\.description\('是否注册/)
  assert.doesNotMatch(groupGuardTypeBody, /guard: StuhelperGuardConfig|commands\?:|fun: StuhelperFunConfig|freshmanForward\?:|admissionCommands\?:|StuhelperAdmissionCommandConfig|admissionReminderDelivery\?:|fallbackScanEnabled\?:|enabled\?: boolean/)
  assert.doesNotMatch(groupGuardTypeBody, /ai: StuhelperAIConfig|StuhelperAIConfig/)
  assert.doesNotMatch(groupGuardTypeBody, /messages\?: StuhelperGroupGuardMessageConfig/)
  assert.doesNotMatch(groupGuardTypeBody, /moderation|StuhelperModerationConfig|keywordRules|StuhelperKeywordRuleConfig/)
  assert.doesNotMatch(types, /interface StuhelperGuardConfig/)
  assert.doesNotMatch(types, /StuhelperAdmissionCommandConfig/)
  assert.doesNotMatch(types, /interface StuhelperAIConfig/)
  assert.doesNotMatch(schema, /createModerationConfigSchema|createKeywordRuleConfigSchema/)
  assert.doesNotMatch(types, /StuhelperModerationConfig|StuhelperKeywordRuleConfig/)
  assert.match(keywordRuleAPI, /new ModerationStore\(api\.ctx\)/)
  assert.match(keywordRuleAPI, /stuhelperGroupCenter\/keyword-rules\/list/)
  assert.match(keywordRuleAPI, /stuhelperGroupCenter\/keyword-rules\/upsert/)
  assert.match(keywordRuleAPI, /stuhelperGroupCenter\/keyword-rules\/delete/)
})

test('README documents admission scopes and backend policy boundary', async () => {
  const readme = await readWorkspaceFile('README.md')

  for (const scope of [
    'bot.admission.session',
    'bot.admission.event',
  ]) {
    assert.match(readme, new RegExp(scope), `README must document ${scope}.`)
  }
  assert.doesNotMatch(readme, /bot\.admission\.(review|forward)/)
  assert.match(readme, /Admission 策略边界/)
  assert.match(readme, /后端 admission policy/)
  assert.match(readme, /同步绑定/)
  assert.match(readme, /只读展示/)
  assert.match(readme, /CommandPolicy/)
  assert.match(readme, /admission-admin/)
  assert.doesNotMatch(readme, /静态 fallback|guard\.targetGroups/)
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
