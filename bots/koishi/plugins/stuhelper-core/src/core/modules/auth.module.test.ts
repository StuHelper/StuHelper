import assert from 'node:assert/strict'
import test from 'node:test'
import type { Context } from 'koishi'

import { AuthModule } from './auth.module'

type CommandAction = (...args: unknown[]) => unknown

function createAuthHarness() {
  const commandActions = new Map<string, CommandAction>()
  const assignments: unknown[] = []
  const roles = [
    {
      id: 'moderator',
      name: '值日管理员',
      alias: 'mod',
      color: '#409eff',
      priority: 10,
      permissions: ['warn.add'],
    },
  ]

  const ctx = {
    stuhelperGroupCenter: {
      auth: {
        registerPermission: () => undefined,
        registerCommand: () => undefined,
        check: () => true,
        getRoles: () => roles,
        getRoleMembers: () => [],
        getUserRoleIds: (userId: string) => (userId === '10004' ? ['moderator'] : []),
        assignRole: async (userId: string, roleId: string) => {
          assignments.push([userId, roleId])
        },
        revokeRole: async () => undefined,
      },
    },
    command(def: string) {
      const commandName = def.split(/\s+/)[0]
      const chain = {
        before: () => chain,
        example: () => chain,
        alias: () => chain,
        action(fn: CommandAction) {
          commandActions.set(commandName, fn)
          return chain
        },
      }
      return chain
    },
  }

  return {
    commandActions,
    assignments,
    module: new AuthModule(ctx as unknown as Context),
  }
}

test('auth role commands accept structured Koishi user arguments', async () => {
  const harness = createAuthHarness()
  await harness.module.init()

  const infoResult = await harness.commandActions.get('gauth.info')?.(
    {},
    { uid: 'onebot:10004' },
  )
  assert.equal(infoResult, '用户 10004 的角色:\n• 值日管理员 (moderator)')

  const addResult = await harness.commandActions.get('gauth.add')?.(
    {},
    { userId: 'onebot:10005' },
    'moderator',
  )
  assert.equal(addResult, '已将用户 10005 添加到角色 "值日管理员"')
  assert.deepEqual(harness.assignments, [['10005', 'moderator']])
})
