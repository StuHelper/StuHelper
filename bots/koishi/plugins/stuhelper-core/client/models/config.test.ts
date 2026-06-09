import assert from 'node:assert/strict'
import test from 'node:test'

import type { ConfigGovernancePageData } from '../page-types'
import { buildConfigGovernanceModel } from './config'

test('buildConfigGovernanceModel exposes current workspace and governance source labels', () => {
  const model = buildConfigGovernanceModel(createConfigFixture(), {
    workspace: 'templates',
  })

  assert.equal(model.currentWorkspace, 'templates')
  assert.equal(model.templateRows[0].sourceLabel, '模板库')
  assert.equal(model.workspaceTabs[3].id, 'command-policies')
})

function createConfigFixture(): ConfigGovernancePageData {
  return {
    generatedAt: '2026-04-21T08:00:00.000Z',
    workspaces: [
      { id: 'guild-config', label: '群配置' },
      { id: 'templates', label: '模板库' },
      { id: 'bindings', label: '同步绑定' },
      { id: 'command-policies', label: '命令策略' },
    ],
    groupConfigs: [],
    templates: [
      {
        id: 'default',
        name: '默认模板',
        muteDurationSeconds: 1800,
        kickAfterMinutes: 30,
        reminderTemplate: '请先认证',
        exemptUsers: [],
        enabled: true,
        createdAt: '2026-04-21T07:00:00.000Z',
        updatedAt: '2026-04-21T07:00:00.000Z',
        source: { kind: 'template-library', label: '模板库' },
      },
    ],
    bindings: [],
    commandPolicies: [],
    supportedCommandIds: ['report'],
  }
}
