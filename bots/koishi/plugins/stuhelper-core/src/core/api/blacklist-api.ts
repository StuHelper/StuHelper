import type { Events as ConsoleEvents } from '@koishijs/console'
import type { DataService } from '@koishijs/console'
import type {
  MemberBlacklistCreateRequest,
  MemberBlacklistEntry,
  MemberBlacklistListQuery,
  MemberBlacklistReleaseRequest,
  PlatformClient,
} from '@stuhelper/koishi-shared'

import {
  assertConsoleGuildAccess,
  assertGlobalConsoleScope,
  hasConsoleGuildAccess,
  type ConsoleGuildScope,
} from './console-guild-scope'

const DEFAULT_BLACKLIST_PLATFORM = 'onebot'
const MEMBER_BLACKLIST_PAGE_SIZE = 100

interface ApiResponse<T = unknown> {
  success: boolean
  data?: T
  error?: string
}

type AuthorityListenerRegistrar = <K extends keyof ConsoleEvents>(
  event: K,
  callback: ConsoleEvents[K],
  options?: DataService.Options,
) => void

interface BlacklistAPIOptions {
  readonly addAuthorityListener: AuthorityListenerRegistrar
  readonly resolveConsoleScope: (client: unknown) => Promise<ConsoleGuildScope>
  readonly platform: PlatformClient
}

interface BlacklistCreateParams {
  readonly platform?: string
  readonly subjectID: string
  readonly scopeType: 'guild' | 'global'
  readonly guildID?: string
  readonly reasonText?: string
}

interface BlacklistReleaseParams {
  readonly id: string
  readonly releaseReason?: string
}

type BlacklistListResponse = ApiResponse<{ items: readonly MemberBlacklistEntry[] }>
type BlacklistCreateResponse = ApiResponse<{ success: boolean; entry: MemberBlacklistEntry }>
type BlacklistReleaseResponse = ApiResponse<{ success: boolean }>

export function registerBlacklistAPI(options: BlacklistAPIOptions): void {
  options.addAuthorityListener('stuhelperGroupCenter/blacklist/list', async function () {
    return handleBlacklistList(options, this)
  })
  options.addAuthorityListener('stuhelperGroupCenter/blacklist/add', async function (params: BlacklistCreateParams) {
    return handleBlacklistCreate(options, this, params)
  })
  options.addAuthorityListener('stuhelperGroupCenter/blacklist/remove', async function (params: BlacklistReleaseParams) {
    return handleBlacklistRelease(options, this, params)
  })
}

export async function listScopedMemberBlacklist(
  platform: PlatformClient,
  scope: ConsoleGuildScope,
): Promise<readonly MemberBlacklistEntry[]> {
  const entries = await listAllMemberBlacklist(platform, { state: 'active' })
  return entries.filter((entry) => hasBlacklistReadAccess(scope, entry))
}

async function handleBlacklistList(
  options: BlacklistAPIOptions,
  client: unknown,
): Promise<BlacklistListResponse> {
  try {
    const scope = await options.resolveConsoleScope(client)
    return success({ items: await listScopedMemberBlacklist(options.platform, scope) })
  } catch (cause) {
    return error(toErrorMessage(cause, '获取黑名单失败'))
  }
}

async function handleBlacklistCreate(
  options: BlacklistAPIOptions,
  client: unknown,
  params: BlacklistCreateParams,
): Promise<BlacklistCreateResponse> {
  try {
    const scope = await options.resolveConsoleScope(client)
    assertBlacklistWriteAccess(scope, params.scopeType, params.guildID)
    const created = await options.platform.createMemberBlacklist(toCreateRequest(params))
    return success({ success: true, entry: created })
  } catch (cause) {
    return error(toErrorMessage(cause, '添加黑名单失败'))
  }
}

async function handleBlacklistRelease(
  options: BlacklistAPIOptions,
  client: unknown,
  params: BlacklistReleaseParams,
): Promise<BlacklistReleaseResponse> {
  try {
    const scope = await options.resolveConsoleScope(client)
    const entry = await findMemberBlacklistByID(options.platform, params.id)
    assertBlacklistEntryWriteAccess(scope, entry)
    await options.platform.releaseMemberBlacklist(entry.id, toReleaseRequest(params))
    return success({ success: true })
  } catch (cause) {
    return error(toErrorMessage(cause, '移除黑名单失败'))
  }
}

async function findMemberBlacklistByID(platform: PlatformClient, id: string) {
  const entry = (await listAllMemberBlacklist(platform, { state: 'active' })).find((item) => item.id === id)
  if (!entry) throw new Error(`member blacklist entry not found: ${id}`)
  return entry
}

export async function listAllMemberBlacklist(platform: PlatformClient, query: MemberBlacklistListQuery) {
  const items: MemberBlacklistEntry[] = []
  let page = 1
  while (true) {
    const result = await platform.listMemberBlacklist({ ...query, page, pageSize: MEMBER_BLACKLIST_PAGE_SIZE })
    items.push(...result.items)
    if (items.length >= result.total || result.items.length === 0) return items
    page += 1
  }
}

function assertBlacklistWriteAccess(scope: ConsoleGuildScope, scopeType: string, guildID?: string) {
  if (scopeType === 'global') {
    assertGlobalConsoleScope(scope, 'global blacklist record')
    return
  }
  if (scopeType !== 'guild') {
    throw new Error(`unsupported blacklist scopeType: ${scopeType}`)
  }
  if (!guildID?.trim()) {
    throw new Error('guildID is required for guild scoped blacklist records')
  }
  assertConsoleGuildAccess(scope, guildID, 'blacklist record')
}

function assertBlacklistEntryWriteAccess(scope: ConsoleGuildScope, entry: MemberBlacklistEntry) {
  assertBlacklistWriteAccess(scope, entry.scopeType, entry.guildID)
}

function hasBlacklistReadAccess(scope: ConsoleGuildScope, entry: MemberBlacklistEntry) {
  if (entry.scopeType === 'global') return true
  return hasConsoleGuildAccess(scope, entry.guildID)
}

function toCreateRequest(params: BlacklistCreateParams): MemberBlacklistCreateRequest {
  const request: MemberBlacklistCreateRequest = {
    platform: params.platform?.trim() || DEFAULT_BLACKLIST_PLATFORM,
    subjectType: 'qq_user',
    subjectID: params.subjectID.trim(),
    scopeType: params.scopeType,
    guildID: params.scopeType === 'guild' ? params.guildID?.trim() : undefined,
    source: 'manual_admin',
    reasonCode: 'manual_blacklist',
    reasonText: params.reasonText?.trim() || 'Koishi console manual blacklist',
    createdFrom: 'koishi_console',
  }
  if (!request.subjectID) throw new Error('subjectID is required')
  return request
}

function toReleaseRequest(params: BlacklistReleaseParams): MemberBlacklistReleaseRequest {
  return {
    releaseReasonCode: 'manual_pardon',
    releaseReason: params.releaseReason?.trim() || 'Koishi console release',
  }
}

function success<T>(data: T): ApiResponse<T> {
  return { success: true, data }
}

function error(message: string): ApiResponse<never> {
  return { success: false, error: message }
}

function toErrorMessage(cause: unknown, fallback: string): string {
  return cause instanceof Error ? cause.message : fallback
}
