import assert from 'node:assert/strict'
import test from 'node:test'
import type { Session } from 'koishi'

import type { Role } from '../../types'
import { collectSessionPermissions, hasPermissionNode } from './auth-permissions'

const roles: Record<string, Role> = {
  authority3: createRole('authority3', ['authority.three']),
  'guild-admin': createRole('guild-admin', ['guild.manage']),
  scoped: createRole('scoped', ['scoped.manage'], ['guild-1']),
}

test('collectSessionPermissions resolves authority and scoped role permissions', () => {
  const perms = collectSessionPermissions({
    session: createSession({
      guildId: 'guild-1',
      user: { authority: 3 },
    }),
    roles,
    userRoleIds: ['scoped'],
  })

  assert.equal(perms.has('authority.three'), true)
  assert.equal(perms.has('scoped.manage'), true)
  assert.equal(hasPermissionNode(perms, 'authority.three'), true)
})

test('collectSessionPermissions treats universal guild role objects as guild admins', () => {
  const perms = collectSessionPermissions({
    session: createSession({
      guildId: 'guild-1',
      author: { roles: [{ id: 'admin', name: '管理员' }] },
    }),
    roles,
    userRoleIds: [],
  })

  assert.equal(perms.has('guild.manage'), true)
})

test('collectSessionPermissions keeps legacy single role compatibility for guild admins', () => {
  const perms = collectSessionPermissions({
    session: createSession({
      guildId: 'guild-1',
      author: { role: 'owner' },
    }),
    roles,
    userRoleIds: [],
  })

  assert.equal(perms.has('guild.manage'), true)
})

function createRole(id: string, permissions: string[], guildIds?: string[]): Role {
  return {
    id,
    name: id,
    priority: 1,
    permissions,
    guildIds,
  }
}

function createSession(input: {
  readonly guildId?: string
  readonly user?: { readonly authority?: unknown }
  readonly author?: {
    readonly roles?: readonly { readonly id: string, readonly name?: string }[]
    readonly role?: string
  }
}): Session {
  return {
    guildId: input.guildId,
    user: input.user,
    author: input.author,
  } as unknown as Session
}
