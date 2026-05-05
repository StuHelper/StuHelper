import assert from 'node:assert/strict'
import test from 'node:test'

import { registerMemberManageCommands } from './member-manage-commands'

test('kick -b --global creates a global member blacklist entry', async () => {
  const host = createMemberManageHost()
  registerMemberManageCommands(host as never)

  const result = await host.runKick('10001 -b --global')

  assert.match(result, /加入黑名单/)
  assert.deepEqual(host.kicks, [{ guildID: 'guild-1', userID: '10001', permanent: true }])
  assert.equal(host.createdBlacklists.length, 1)
  assert.equal(host.createdBlacklists[0].scopeType, 'global')
  assert.equal(host.createdBlacklists[0].guildID, undefined)
  assert.equal(host.createdBlacklists[0].metadata.targetGuildID, 'guild-1')
  assert.equal(host.createdBlacklists[0].metadata.scopeSelectionContext, 'explicit_global_flag')
})

function createMemberManageHost() {
  const host = {
    actions: new Map<string, Function>(),
    kicks: [] as Array<{ guildID: string; userID: string; permanent?: boolean }>,
    createdBlacklists: [] as Array<Record<string, any>>,
    ctx: {
      stuhelperGroupCenter: {
        pushMessage: async () => {},
      },
    },
    config: {
      setTitle: { enabled: false },
    },
    memberBlacklistBackend: {
      async createMemberBlacklist(input: Record<string, any>) {
        host.createdBlacklists.push(input)
        return { id: 'blacklist-1', ...input }
      },
    },
    registerCommand(def: { name: string }) {
      return commandChain((action) => host.actions.set(def.name, action))
    },
    logCommand() {},
    runKick(input: string) {
      const action = host.actions.get('kick')
      assert.ok(action)
      return action({ session: createSession(host) }, input)
    },
  }
  return host
}

function commandChain(onAction: (action: Function) => void) {
  return {
    example() { return this },
    option() { return this },
    action(handler: Function) {
      onAction(handler)
      return this
    },
  }
}

function createSession(host: ReturnType<typeof createMemberManageHost>) {
  return {
    platform: 'qq',
    guildId: 'guild-1',
    userId: '90001',
    content: 'kick 10001 -b --global',
    bot: {
      kickGuildMember: async (guildID: string, userID: string, permanent?: boolean) => {
        host.kicks.push({ guildID, userID, permanent })
      },
    },
  }
}
