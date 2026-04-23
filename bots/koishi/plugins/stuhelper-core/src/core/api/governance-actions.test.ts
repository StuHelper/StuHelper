import assert from 'node:assert/strict'
import test from 'node:test'

import {
  assertCommandPolicyWriteAccess,
  assertGuardBindingWriteAccess,
  assertGuardTemplateWriteAccess,
  parseCommandPolicyInput,
  parseGuardBindingInput,
  parseGuardTemplateInput,
} from './governance-actions'

test('parseCommandPolicyInput trims identifiers and rejects invalid authority values', () => {
  assert.deepEqual(
    parseCommandPolicyInput({
      commandId: '  report  ',
      minAuthority: 4,
      roles: [' admin ', 'moderator'],
    }),
    {
      commandId: 'report',
      minAuthority: 4,
      roles: ['admin', 'moderator'],
    },
  )

  assert.throws(
    () => parseCommandPolicyInput({
      commandId: 'report',
      minAuthority: -1,
      roles: [],
    }),
    /minAuthority/,
  )

  assert.throws(
    () => parseCommandPolicyInput({
      commandId: 'report',
      minAuthority: Number.POSITIVE_INFINITY,
      roles: [],
    }),
    /minAuthority/,
  )
})

test('parseGuardTemplateInput trims text fields and rejects out-of-range numeric values', () => {
  assert.deepEqual(
    parseGuardTemplateInput({
      id: ' tpl-default ',
      name: ' 默认模板 ',
      muteDurationSeconds: 1800,
      kickAfterMinutes: 30,
      reminderTemplate: ' 请先完成认证 ',
      exemptUsers: [' 1001 ', '1002'],
      enabled: true,
    }),
    {
      id: 'tpl-default',
      name: '默认模板',
      muteDurationSeconds: 1800,
      kickAfterMinutes: 30,
      reminderTemplate: '请先完成认证',
      exemptUsers: ['1001', '1002'],
      enabled: true,
    },
  )

  assert.throws(
    () => parseGuardTemplateInput({
      id: 'tpl-default',
      name: '默认模板',
      muteDurationSeconds: -1,
      kickAfterMinutes: 30,
      reminderTemplate: '请先完成认证',
      exemptUsers: [],
      enabled: true,
    }),
    /muteDurationSeconds/,
  )

  assert.throws(
    () => parseGuardTemplateInput({
      id: 'tpl-default',
      name: '默认模板',
      muteDurationSeconds: 1800,
      kickAfterMinutes: Number.POSITIVE_INFINITY,
      reminderTemplate: '请先完成认证',
      exemptUsers: [],
      enabled: true,
    }),
    /kickAfterMinutes/,
  )
})

test('parseGuardBindingInput trims text fields and normalizes blank note to null', () => {
  assert.deepEqual(
    parseGuardBindingInput({
      platform: ' onebot ',
      guildId: ' 1001 ',
      templateId: ' tpl-default ',
      enabled: true,
      note: ' 主群 ',
    }),
    {
      platform: 'onebot',
      guildId: '1001',
      templateId: 'tpl-default',
      enabled: true,
      note: '主群',
    },
  )

  assert.deepEqual(
    parseGuardBindingInput({
      platform: 'onebot',
      guildId: '1001',
      templateId: 'tpl-default',
      enabled: false,
      note: '   ',
    }),
    {
      platform: 'onebot',
      guildId: '1001',
      templateId: 'tpl-default',
      enabled: false,
      note: null,
    },
  )
})

test('governance write access rejects global writes for guild-scoped operators', () => {
  const scoped = {
    kind: 'guilds' as const,
    guildIds: new Set(['1001']),
  }

  assert.throws(
    () => assertCommandPolicyWriteAccess(scoped),
    /requires global console scope/,
  )
  assert.throws(
    () => assertGuardTemplateWriteAccess(scoped),
    /requires global console scope/,
  )
  assert.throws(
    () => assertGuardBindingWriteAccess(scoped, {
      platform: 'onebot',
      guildId: '2002',
      templateId: 'tpl-default',
      enabled: true,
      note: null,
    }),
    /outside of the current console guild scope/,
  )
})
