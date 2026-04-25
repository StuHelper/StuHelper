import assert from 'node:assert/strict'
import test from 'node:test'

import {
  buildPlatformView,
  parseConfigText,
  type PlatformData,
} from './model'

const GENERATED_AT = '2026-04-24T10:00:00.000Z'

test('buildPlatformView maps modules into concise navigation and selected config', () => {
  const view = buildPlatformView(platformData(), 'group-guard')

  assert.deepEqual(view.modules, [{
    id: 'group-guard',
    name: '群守卫',
    description: '管理群成员准入。',
    category: 'moderation',
    enabled: true,
    statusText: '已加载',
  }])
  assert.deepEqual(view.selectedModule, {
    id: 'group-guard',
    name: '群守卫',
    enabled: true,
    configText: [
      '{',
      '  "targetGroups": [',
      '    "10001"',
      '  ],',
      '  "diceSides": 100',
      '}',
    ].join('\n'),
  })
})

test('buildPlatformView derives policy permissions and audit rows', () => {
  const view = buildPlatformView(platformData(), 'group-guard')

  assert.deepEqual(view.policyRows, [{
    id: 'group-guard.manage',
    moduleName: '群守卫',
    label: '管理群守卫',
    description: '调整群守卫配置。',
  }])
  assert.deepEqual(view.groupPolicyRows, [{
    id: 'group-guard.policy',
    moduleName: '群守卫',
    label: '群守卫策略',
  }])
  assert.deepEqual(view.auditRows, [{
    id: 'audit-1',
    moduleName: '群守卫',
    actor: 'console',
    action: 'module.config.save',
    summary: '保存模块配置：group-guard',
    createdAt: '2026-04-24 10:00:00',
  }])
})

test('buildPlatformView selects first module when no selected id is provided', () => {
  const view = buildPlatformView(platformData())

  assert.equal(view.selectedModule?.id, 'group-guard')
})

test('parseConfigText accepts JSON objects and rejects invalid config text', () => {
  assert.deepEqual(parseConfigText('{"enabled":true}'), { enabled: true })
  assert.throws(() => parseConfigText('[]'), /配置必须是 JSON 对象/)
  assert.throws(() => parseConfigText('{bad'), /配置不是有效 JSON/)
})

function platformData(): PlatformData {
  return {
    generatedAt: GENERATED_AT,
    modules: [{
      manifest: {
        id: 'group-guard',
        name: '群守卫',
        description: '管理群成员准入。',
        version: '0.1.0',
        category: 'moderation',
        defaultEnabled: true,
        order: 10,
      },
      enabled: true,
      status: 'loaded',
      lastError: null,
      config: {
        targetGroups: ['10001'],
        diceSides: 100,
      },
      permissions: [{
        id: 'group-guard.manage',
        label: '管理群守卫',
        description: '调整群守卫配置。',
      }],
      commands: [],
      events: [],
      webui: [{
        id: 'group-guard.policy',
        label: '群守卫策略',
        section: 'policy',
      }],
    }],
    auditEvents: [{
      id: 'audit-1',
      actor: 'console',
      moduleId: 'group-guard',
      action: 'module.config.save',
      summary: '保存模块配置：group-guard',
      payload: { config: { diceSides: 100 } },
      createdAt: GENERATED_AT,
      updatedAt: GENERATED_AT,
    }],
  }
}
