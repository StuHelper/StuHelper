import assert from 'node:assert/strict'
import test from 'node:test'
import type { Context } from 'koishi'

import { GetAuthModule } from './getauth.module'

type CommandAction = (...args: unknown[]) => unknown

function createGetAuthHarness() {
  const commandActions = new Map<string, CommandAction>()
  const databaseLookups: unknown[] = []
  const memberLookups: unknown[] = []
  const muteLookups: unknown[] = []
  const roles = [
    {
      id: 'moderator',
      name: '值日管理员',
      color: '#409eff',
      priority: 10,
      permissions: ['warn.add'],
    },
  ]

  const ctx = {
    database: {
      async getUser(platform: string, userId: string) {
        databaseLookups.push([platform, userId])
        return { authority: 3 }
      },
    },
    stuhelperGroupCenter: {
      auth: {
        registerPermission: () => undefined,
        registerCommand: () => undefined,
        check: () => true,
        getUserRoleIds: (userId: string) => (userId === '10004' ? ['moderator'] : []),
        getRoles: () => roles,
      },
    },
    command(def: string) {
      const commandName = def.split(/\s+/)[0]
      const chain = {
        before: () => chain,
        alias: () => chain,
        example: () => chain,
        action(fn: CommandAction) {
          commandActions.set(commandName, fn)
          return chain
        },
      }
      return chain
    },
  }

  const session = {
    platform: 'qq',
    guildId: 'guild-1',
    bot: {
      platform: 'onebot',
      getGuildMember: async (guildId: string, userId: string) => {
        memberLookups.push([guildId, userId])
        return { roles: [{ id: 'admin', name: '管理员' }] }
      },
      internal: {
        getGroupMemberInfo: async (guildId: string, userId: string, noCache: boolean) => {
          muteLookups.push([guildId, userId, noCache])
          return { shut_up_timestamp: 0 }
        },
      },
    },
  }

  return {
    commandActions,
    databaseLookups,
    memberLookups,
    muteLookups,
    module: new GetAuthModule(ctx as unknown as Context),
    session,
  }
}

test('getauth normalizes platform-qualified user ids before querying state', async () => {
  const harness = createGetAuthHarness()
  await harness.module.init()

  const result = await harness.commandActions.get('getauth')?.(
    { session: harness.session },
    'onebot:10004',
  )

  assert.equal(result, [
    '成员 10004',
    '群角色: 管理员',
    '未禁言',
    '权限等级: 3',
    '自定义角色: 值日管理员',
  ].join('\n'))
  assert.deepEqual(harness.databaseLookups, [['qq', '10004']])
  assert.deepEqual(harness.memberLookups, [['guild-1', '10004']])
  assert.deepEqual(harness.muteLookups, [['guild-1', '10004', false]])
})
