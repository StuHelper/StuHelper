import type { Context, Session } from 'koishi'

import { registerRuntimeCommand } from '../../runtime/command'
import type {
  RuntimeModule,
  RuntimeModuleInstance,
  RuntimeModuleMeta,
  RuntimeModuleState,
} from '../../runtime/types'
import { formatDuration } from '../../utils'

const UNIX_MILLISECONDS_THRESHOLD = 1_000_000_000_000
const MILLISECONDS_PER_SECOND = 1_000

interface MemberStatus {
  readonly role: string
  readonly muteLine: string
}

interface AuthResultInput {
  readonly userId: string
  readonly memberStatus: MemberStatus
  readonly authorityLevel: number
  readonly customRoles: string
}

/**
 * 获取权限模块 - 查询用户状态和权限
 */
export class GetAuthModule implements RuntimeModuleInstance {
  readonly meta: RuntimeModuleMeta = {
    name: 'getauth',
    description: '获取权限模块 - 查询用户状态和权限',
  }

  private _state: RuntimeModuleState = 'unloaded'
  private _error: Error | null = null

  constructor(private readonly ctx: Context) {}

  get state(): RuntimeModuleState {
    return this._state
  }

  get error(): Error | null {
    return this._error
  }

  async init(): Promise<void> {
    this._state = 'loading'
    try {
      registerGetAuthCommand(this.ctx, this.meta)
      this._state = 'loaded'
    } catch (error) {
      this._state = 'error'
      this._error = error as Error
      throw error
    }
  }

  async dispose(): Promise<void> {
    this._state = 'unloaded'
  }
}

function registerGetAuthCommand(ctx: Context, meta: RuntimeModuleMeta): void {
  registerRuntimeCommand(ctx, meta, {
    name: 'getauth',
    desc: '获取指定成员状态喵',
    args: '<target:text>',
    permNode: 'getauth',
    permDesc: '查询用户权限状态',
    usage: '查询用户的群角色、禁言状态、权限等级',
    examples: ['getauth @用户', 'getauth 123456789'],
  })
    .alias('ga')
    .example('getauth @可爱猫娘')
    .example('getauth 2038794363')
    .action(async ({ session }, target) => {
      return handleGetAuthCommand(ctx, session, target)
    })
}

async function handleGetAuthCommand(
  ctx: Context,
  session: Session,
  target?: string,
): Promise<string> {
  if (!target) return '请指定要查询的成员喵'

  const userId = parseUserId(target)
  if (!userId) return '无法解析成员喵'

  try {
    const dbAuthority = await readDatabaseAuthority(ctx, session, userId)
    const memberStatus = await readMemberStatus(session, userId)
    const authorityLevel = await resolveAuthorityLevel(session, dbAuthority)
    const customRoles = formatCustomRoles(ctx, userId)
    return formatAuthResult({
      userId,
      memberStatus,
      authorityLevel,
      customRoles,
    })
  } catch (error) {
    return `查询失败：${getErrorMessage(error)}喵`
  }
}

function parseUserId(target: string): string | null {
  if (!target) return null
  if (target.startsWith('<at')) {
    const match = target.match(/id="(\d+)"/)
    if (match) return match[1]
  }
  return target.replace(/^@/, '').trim() || null
}

async function readDatabaseAuthority(
  ctx: Context,
  session: Session,
  userId: string,
): Promise<number> {
  if (!ctx.database) return 0

  try {
    const dbUser = await ctx.database.getUser(session.platform, userId)
    return dbUser && typeof dbUser.authority === 'number' ? dbUser.authority : 0
  } catch {
    return 0
  }
}

async function readMemberStatus(
  session: Session,
  userId: string,
): Promise<MemberStatus> {
  if (!session.guildId || !session.bot.internal?.getGroupMemberInfo) {
    return { role: '未知', muteLine: '未禁言' }
  }

  try {
    const info = await session.bot.internal.getGroupMemberInfo(session.guildId, userId, false)
    return formatMemberStatus(info)
  } catch {
    return { role: '未知', muteLine: '未禁言' }
  }
}

function formatMemberStatus(info: any): MemberStatus {
  if (!info) return { role: '未知', muteLine: '未禁言' }

  const role = typeof info.role !== 'undefined' ? info.role : '未知'
  const timestamp = Number(info.shut_up_timestamp) || 0
  if (timestamp <= 0) return { role, muteLine: '未禁言' }

  const timestampMs = timestamp > UNIX_MILLISECONDS_THRESHOLD
    ? timestamp
    : timestamp * MILLISECONDS_PER_SECOND
  const remaining = timestampMs - Date.now()
  const endTime = new Date(timestampMs).toLocaleString()
  const muteLine = remaining > 0
    ? `禁言至 ${endTime}(剩余 ${formatDuration(remaining)})`
    : `已解除禁言(原到期：${endTime})`

  return { role, muteLine }
}

async function resolveAuthorityLevel(
  session: Session,
  authorityLevel: number,
): Promise<number> {
  if (authorityLevel !== 0 || !session.observeUser) return authorityLevel

  try {
    const observed = await session.observeUser(['authority'])
    return observed && typeof observed.authority === 'number' ? observed.authority : 0
  } catch {
    return 0
  }
}

function formatCustomRoles(ctx: Context, userId: string): string {
  const userRoleIds = ctx.stuhelperGroupCenter.auth.getUserRoleIds(userId)
  const allRoles = ctx.stuhelperGroupCenter.auth.getRoles()
  return userRoleIds
    .map(id => allRoles.find(role => role.id === id)?.name || id)
    .join(', ') || '无'
}

function formatAuthResult(input: AuthResultInput): string {
  return [
    `成员 ${input.userId}`,
    `群角色: ${input.memberStatus.role}`,
    input.memberStatus.muteLine,
    `权限等级: ${input.authorityLevel}`,
    `自定义角色: ${input.customRoles}`,
  ].join('\n')
}

function getErrorMessage(error: unknown): string {
  if (error instanceof Error) return error.message
  return String(error)
}

export const getauthRuntimeModule: RuntimeModule<GetAuthModule> = {
  id: 'getauth',
  create(ctx) {
    return new GetAuthModule(ctx)
  },
}
