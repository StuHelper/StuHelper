import assert from 'node:assert/strict'
import test from 'node:test'

import {
  buildManagedRoleReorder,
  normalizeManagedRoleInput,
  requireAssignableRole,
} from './auth-management'

const GLOBAL_SCOPE = { kind: 'all' as const }
const GUILD_SCOPE = {
  kind: 'guilds' as const,
  guildIds: new Set(['1001']),
}

test('normalizeManagedRoleInput rejects builtin role updates even for global operators', () => {
  assert.throws(
    () => normalizeManagedRoleInput(createCatalog(), GLOBAL_SCOPE, {
      id: 'authority4+',
      name: 'Authority >=4',
      priority: 400,
      permissions: ['*'],
      guildIds: [],
    }),
    /builtin role cannot be modified/,
  )
})

test('normalizeManagedRoleInput rejects global custom roles for guild-scoped operators', () => {
  assert.throws(
    () => normalizeManagedRoleInput(createCatalog(), GUILD_SCOPE, {
      id: 'guild-ops',
      name: 'Guild Ops',
      priority: 90,
      permissions: ['report.*'],
      guildIds: [],
    }),
    /global roles require global console scope/,
  )
})

test('normalizeManagedRoleInput rejects out-of-scope guild ids for guild-scoped operators', () => {
  assert.throws(
    () => normalizeManagedRoleInput(createCatalog(), GUILD_SCOPE, {
      id: 'guild-ops',
      name: 'Guild Ops',
      priority: 90,
      permissions: ['report.*'],
      guildIds: ['1001', '2002'],
    }),
    /outside of the current console guild scope/,
  )
})

test('requireAssignableRole rejects global roles for guild-scoped operators', () => {
  assert.throws(
    () => requireAssignableRole(createCatalog(), GUILD_SCOPE, 'global-ops'),
    /global roles require global console scope/,
  )
})

test('requireAssignableRole allows scoped roles within the current guild scope', () => {
  const role = requireAssignableRole(createCatalog(), GUILD_SCOPE, 'guild-1001-ops')
  assert.equal(role.id, 'guild-1001-ops')
})

test('buildManagedRoleReorder swaps custom role priorities', () => {
  const [source, target] = buildManagedRoleReorder(createCatalog(), GLOBAL_SCOPE, {
    sourceRoleId: 'global-ops',
    targetRoleId: 'guild-1001-ops',
  })

  assert.equal(source.id, 'global-ops')
  assert.equal(source.priority, 120)
  assert.equal(target.id, 'guild-1001-ops')
  assert.equal(target.priority, 200)
})

test('buildManagedRoleReorder rejects builtin source or target roles', () => {
  assert.throws(
    () => buildManagedRoleReorder(createCatalog(), GLOBAL_SCOPE, {
      sourceRoleId: 'authority4+',
      targetRoleId: 'global-ops',
    }),
    /builtin role cannot be modified/,
  )
  assert.throws(
    () => buildManagedRoleReorder(createCatalog(), GLOBAL_SCOPE, {
      sourceRoleId: 'global-ops',
      targetRoleId: 'authority4+',
    }),
    /builtin role cannot be modified/,
  )
})

function createCatalog() {
  return {
    isBuiltinRole(roleId: string) {
      return roleId === 'authority4+'
    },
    getRoles() {
      return [
        {
          id: 'authority4+',
          name: 'Authority >=4',
          priority: 400,
          permissions: ['*'],
          guildIds: [],
          builtin: true,
        },
        {
          id: 'global-ops',
          name: 'Global Ops',
          priority: 200,
          permissions: ['report.*'],
          guildIds: [],
        },
        {
          id: 'guild-1001-ops',
          name: 'Guild 1001 Ops',
          priority: 120,
          permissions: ['report.*'],
          guildIds: ['1001'],
        },
      ]
    },
  }
}
