import assert from 'node:assert/strict'
import test from 'node:test'

import { isGuildAdminMember, isGuildAdminRole } from './auth-guild-admin'

test('isGuildAdminRole accepts universal role objects and legacy role strings', () => {
  assert.equal(isGuildAdminRole('admin'), true)
  assert.equal(isGuildAdminRole('owner'), true)
  assert.equal(isGuildAdminRole({ id: 'admin', name: '管理员' }), true)
  assert.equal(isGuildAdminRole({ id: 'member', name: '普通成员' }), false)
})

test('isGuildAdminMember accepts universal and legacy member role fields', () => {
  assert.equal(isGuildAdminMember({ roles: [{ id: 'owner' }] }), true)
  assert.equal(isGuildAdminMember({ role: 'admin' }), true)
  assert.equal(isGuildAdminMember({ roles: [{ id: 'member' }] }), false)
  assert.equal(isGuildAdminMember(null), false)
})
