import type { PlatformClient } from '@stuhelper/koishi-shared'

import type { ConsoleGuildScope } from './console-guild-scope'
import { DEFAULT_MEMBER_BLACKLIST_PLATFORM } from './member-blacklist-defaults'

const DASHBOARD_BLACKLIST_PAGE_SIZE = 1

export type MemberBlacklistStatsBackend = Pick<PlatformClient, 'listMemberBlacklist'>

export async function loadScopedBlacklistTotal(
  backend: MemberBlacklistStatsBackend | undefined,
  scope: ConsoleGuildScope,
  platform = DEFAULT_MEMBER_BLACKLIST_PLATFORM,
) {
  if (!backend) {
    throw new Error('member blacklist backend client is required for dashboard stats')
  }
  if (scope.kind === 'all') {
    return loadBlacklistTotal(backend, { platform })
  }
  const totals = await Promise.all([
    loadBlacklistTotal(backend, { platform, scopeType: 'global' }),
    ...[...scope.guildIds].map((guildID) =>
      loadBlacklistTotal(backend, { platform, scopeType: 'guild', guildID })),
  ])
  return totals.reduce((sum, n) => sum + n, 0)
}

function loadBlacklistTotal(
  backend: MemberBlacklistStatsBackend,
  input: {
    readonly platform: string
    readonly scopeType?: 'global' | 'guild'
    readonly guildID?: string
  },
) {
  return backend.listMemberBlacklist({
    ...input,
    status: 'active',
    pageSize: DASHBOARD_BLACKLIST_PAGE_SIZE,
  }).then((page) => page.total)
}
