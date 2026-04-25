import assert from 'node:assert/strict'
import test from 'node:test'

import { buildConfigGovernanceData } from './config-governance.service'

test('buildConfigGovernanceData merges config, template, binding and command policy data', () => {
  const data = buildConfigGovernanceData({
    generatedAt: '2026-04-21T08:00:00.000Z',
    groupConfigs: {
      '1001': {
        welcomeEnabled: true,
      },
    },
    guildNames: {
      '1001': { name: '测试群', avatar: 'avatar-url' },
    },
    templates: [createTemplate()],
    bindings: [createBinding()],
    commandPolicies: [createCommandPolicy()],
    supportedCommandIds: ['report', 'guard-status'],
  })

  assert.equal(data.workspaces[0].id, 'guild-config')
  assert.equal(data.groupConfigs[0].guildName, '测试群')
  assert.equal(data.templates[0].source.label, '模板库')
  assert.equal(data.bindings[0].effectiveTemplateName, '默认模板')
  assert.equal(data.commandPolicies[0].commandId, 'report')
  assert.deepEqual(data.supportedCommandIds, ['report', 'guard-status'])
})

function createTemplate() {
  return {
    id: 'tpl-1',
    name: '默认模板',
    muteDurationSeconds: 1800,
    kickAfterMinutes: 30,
    reminderTemplate: '请先完成认证',
    exemptUsers: ['2001'],
    enabled: true,
    createdAt: new Date('2026-04-21T07:30:00.000Z'),
    updatedAt: new Date('2026-04-21T07:30:00.000Z'),
  }
}

function createBinding() {
  return {
    id: 'onebot:1001',
    platform: 'onebot',
    guildId: '1001',
    templateId: 'tpl-1',
    enabled: true,
    note: '主群',
    createdAt: new Date('2026-04-21T07:35:00.000Z'),
    updatedAt: new Date('2026-04-21T07:35:00.000Z'),
  }
}

function createCommandPolicy() {
  return {
    commandId: 'report',
    roles: ['admin'],
    minAuthority: 3,
    createdAt: new Date('2026-04-21T07:40:00.000Z'),
    updatedAt: new Date('2026-04-21T07:40:00.000Z'),
  }
}
