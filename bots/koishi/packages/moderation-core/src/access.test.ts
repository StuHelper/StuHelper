import assert from 'node:assert/strict'
import test from 'node:test'

import { canExecuteCommand, createFallbackCommandPolicy } from './access'

test('canExecuteCommand 按 minAuthority 与角色判定', () => {
  const policy = {
    commandId: 'guard.status',
    roles: ['moderator'],
    minAuthority: 4,
    createdAt: new Date(0),
    updatedAt: new Date(0),
  }

  assert.equal(canExecuteCommand({ authority: 4, memberRoles: [], policy }), true)
  assert.equal(canExecuteCommand({ authority: 3, memberRoles: ['moderator'], policy }), true)
  assert.equal(canExecuteCommand({ authority: 3, memberRoles: ['member'], policy }), false)
  assert.equal(canExecuteCommand({ authority: 0, memberRoles: [], policy }), false)
})

test('无角色策略在权限不足时直接拒绝（fail-closed）', () => {
  const policy = createFallbackCommandPolicy('guard.mute', 4)

  assert.equal(policy.commandId, 'guard.mute')
  assert.deepEqual(policy.roles, [])
  assert.equal(canExecuteCommand({ authority: 3, memberRoles: ['moderator'], policy }), false)
  assert.equal(canExecuteCommand({ authority: 4, memberRoles: [], policy }), true)
})

test('公开命令兜底策略 minAuthority 0 对所有人开放', () => {
  const policy = createFallbackCommandPolicy('report', 0)

  assert.equal(canExecuteCommand({ authority: 0, memberRoles: [], policy }), true)
})
