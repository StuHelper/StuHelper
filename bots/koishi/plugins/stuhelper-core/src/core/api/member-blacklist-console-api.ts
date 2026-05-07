import type { Context } from 'koishi'
import type {
  MemberBlacklistEntry,
  MemberBlacklistReleaseReasonCode,
  MemberBlacklistScopeType,
  PlatformClient,
} from '@stuhelper/koishi-shared'

import type { StuhelperGroupCenterService } from '../services'
import { createAuthority4ListenerRegistrar } from './authority-listener'
import {
  assertConsoleGuildAccess,
  assertGlobalConsoleScope,
  resolveRequiredConsoleGuildScope,
  type ConsoleGuildScope,
} from './console-guild-scope'
import { DEFAULT_MEMBER_BLACKLIST_PLATFORM } from './member-blacklist-defaults'

const CONSOLE_BLACKLIST_PAGE_SIZE = 200
const CONSOLE_SCOPE_SELECTION_CONTEXT = 'koishi_console_form'

const ALLOWED_CONSOLE_RELEASE_CODES: ReadonlySet<MemberBlacklistReleaseReasonCode> = new Set([
  'manual_pardon',
  'release_only',
  'admission_appeal_passed',
])

interface ConsoleBlacklistCreateParams {
  readonly platform: string
  readonly subjectID: string
  readonly scopeType: MemberBlacklistScopeType
  readonly guildID?: string
  readonly reasonText?: string
}

interface ConsoleBlacklistReleaseParams {
  readonly id: string
  readonly scopeType: MemberBlacklistScopeType
  readonly guildID?: string
  readonly releaseReasonCode: MemberBlacklistReleaseReasonCode
  readonly releaseReason?: string
}

interface ConsoleBlacklistOptions {
  readonly platform?: string
}

type MemberBlacklistBackend = Pick<
  PlatformClient,
  'createMemberBlacklist' | 'listMemberBlacklist' | 'releaseMemberBlacklist'
>

export function registerMemberBlacklistConsoleAPI(
  ctx: Context,
  service: StuhelperGroupCenterService,
  backend: MemberBlacklistBackend,
  options: ConsoleBlacklistOptions = {},
) {
  if (!ctx.console) return

  const addAuthorityListener = createAuthority4ListenerRegistrar(ctx.console)
  const platform = options.platform || DEFAULT_MEMBER_BLACKLIST_PLATFORM

  addAuthorityListener('stuhelperGroupCenter/blacklist/list', async function () {
    const scope = await resolveConsoleScope(ctx, service, this)
    return success(await listVisibleBlacklists(backend, scope, platform))
  })

  addAuthorityListener('stuhelperGroupCenter/blacklist/add', async function (params: ConsoleBlacklistCreateParams) {
    const scope = await resolveConsoleScope(ctx, service, this)
    assertBlacklistScope(scope, params)
    const entry = await backend.createMemberBlacklist({
      platform: params.platform,
      subjectType: 'qq_user',
      subjectID: params.subjectID,
      scopeType: params.scopeType,
      guildID: params.guildID,
      source: 'manual_admin',
      reasonCode: 'manual_blacklist',
      reasonText: params.reasonText?.trim() || 'manual blacklist from Koishi console',
      createdFrom: 'koishi_console',
      metadata: consoleBlacklistMetadata(params, this),
    })
    return success(entry)
  })

  addAuthorityListener('stuhelperGroupCenter/blacklist/remove', async function (params: ConsoleBlacklistReleaseParams) {
    const scope = await resolveConsoleScope(ctx, service, this)
    if (!ALLOWED_CONSOLE_RELEASE_CODES.has(params.releaseReasonCode)) {
      throw new Error(`unsupported releaseReasonCode for koishi console: ${params.releaseReasonCode}`)
    }
    await assertVisibleBlacklistRelease(backend, scope, platform, params.id)
    const entry = await backend.releaseMemberBlacklist(params.id, {
      releaseReasonCode: params.releaseReasonCode,
      releaseReason: params.releaseReason?.trim() || 'manual release from Koishi console',
    })
    return success(entry)
  })
}

async function listVisibleBlacklists(
  backend: MemberBlacklistBackend,
  scope: ConsoleGuildScope,
  platform: string,
): Promise<{ readonly list: readonly MemberBlacklistEntry[]; readonly total: number }> {
  if (scope.kind === 'all') {
    return listAllMemberBlacklistPages(backend, { platform, status: 'active' })
  }

  const pages = await Promise.all([...scope.guildIds].map((guildID) =>
    listAllMemberBlacklistPages(backend, { platform, scopeType: 'guild', guildID, status: 'active' })))
  return {
    list: pages.flatMap((page) => page.list),
    total: pages.reduce((sum, page) => sum + page.total, 0),
  }
}

async function assertVisibleBlacklistRelease(
  backend: MemberBlacklistBackend,
  scope: ConsoleGuildScope,
  platform: string,
  id: string,
) {
  const result = await listVisibleBlacklists(backend, scope, platform)
  const entry = result.list.find((item) => item.id === id)
  if (!entry) {
    throw new Error(`member blacklist entry is outside of the current console scope: ${id}`)
  }
  assertBlacklistScope(scope, entry)
}

async function listAllMemberBlacklistPages(
  backend: MemberBlacklistBackend,
  query: Parameters<MemberBlacklistBackend['listMemberBlacklist']>[0],
) {
  const list: MemberBlacklistEntry[] = []
  let total = 0
  for (let page = 1; ; page++) {
    const result = await backend.listMemberBlacklist({ ...query, page, pageSize: CONSOLE_BLACKLIST_PAGE_SIZE })
    list.push(...result.list)
    total = result.total
    if (list.length >= total) {
      return { list, total }
    }
    if (result.list.length === 0) {
      throw new Error(`member blacklist pagination ended before total was reached: ${list.length}/${total}`)
    }
  }
}

function assertBlacklistScope(
  scope: ConsoleGuildScope,
  input: { readonly scopeType: MemberBlacklistScopeType; readonly guildID?: string },
) {
  if (input.scopeType === 'global') {
    assertGlobalConsoleScope(scope, 'global member blacklist')
    return
  }
  assertConsoleGuildAccess(scope, input.guildID, 'member blacklist')
}

function resolveConsoleScope(ctx: Context, service: StuhelperGroupCenterService, client: unknown) {
  return resolveRequiredConsoleGuildScope(client as never, {
    roles: service.auth.getRoles(),
    getUserRoleIds: (userId: string) => service.auth.getUserRoleIds(userId),
    listBindingsByAuthId: (authId: number) => ctx.database.get('binding', { aid: authId }),
  })
}

function consoleAuthID(client: unknown): string {
  const authID = (client as { auth?: { id?: unknown } }).auth?.id
  return authID === undefined || authID === null ? 'koishi-console' : String(authID)
}

function consoleBlacklistMetadata(params: ConsoleBlacklistCreateParams, client: unknown) {
  return {
    consoleAuthID: consoleAuthID(client),
    operatorInput: params.subjectID,
    scopeSelectionContext: CONSOLE_SCOPE_SELECTION_CONTEXT,
  }
}

function success<T>(data: T) {
  return { success: true, data }
}
