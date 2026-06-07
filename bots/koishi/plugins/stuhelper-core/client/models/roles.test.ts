import assert from 'node:assert/strict'
import test from 'node:test'

import type { Role } from '../types'
import {
  canDragRoleForReorder,
  canDropRoleForReorder,
} from './roles'

test('role reorder model only allows ready custom roles to be dragged', () => {
  assert.equal(canDragRoleForReorder(role('custom')), true)
  assert.equal(canDragRoleForReorder(role('authority4+', true)), false)
  assert.equal(canDragRoleForReorder(role('custom'), { roleOperation: 'delete' }), false)
  assert.equal(canDragRoleForReorder(role('custom'), { savingChanges: true }), false)
  assert.equal(canDragRoleForReorder(role('custom'), { loading: true }), false)
  assert.equal(canDragRoleForReorder(role('custom'), { hasLoadError: true }), false)
  assert.equal(canDragRoleForReorder(role('custom'), { hasChanges: true }), false)
})

test('role reorder model rejects builtin and same-role drop targets', () => {
  const source = role('source')

  assert.equal(canDropRoleForReorder(source, role('target')), true)
  assert.equal(canDropRoleForReorder(source, role('source')), false)
  assert.equal(canDropRoleForReorder(role('authority4+', true), role('target')), false)
  assert.equal(canDropRoleForReorder(source, role('authority4+', true)), false)
  assert.equal(canDropRoleForReorder(null, role('target')), false)
  assert.equal(canDropRoleForReorder(source, role('target'), { roleOperation: 'create' }), false)
})

function role(id: string, builtin = false): Role {
  return {
    id,
    name: id,
    priority: 10,
    permissions: [],
    guildIds: [],
    builtin,
  }
}
