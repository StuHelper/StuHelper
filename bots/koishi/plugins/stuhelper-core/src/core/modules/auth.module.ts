import type { Context } from 'koishi'

import { registerRuntimeCommand } from '../../runtime/command'
import type {
  RuntimeModule,
  RuntimeModuleInstance,
  RuntimeModuleMeta,
  RuntimeModuleState,
} from '../../runtime/types'
import { BUILTIN_ROLE_IDS } from '../services/auth.service'
import { resolveCommandUserId } from './member-manage-input'

type AuthRole = ReturnType<Context['stuhelperGroupCenter']['auth']['getRoles']>[number]

const BUILTIN_ROLE_ID_SET: ReadonlySet<string> = new Set(BUILTIN_ROLE_IDS)

interface RoleLookup {
  readonly role: AuthRole | null
  readonly warning?: string
}

interface RoleMutationInput {
  readonly ctx: Context
  readonly target: unknown
  readonly roleIdentifier?: string
  readonly operation: 'add' | 'remove'
}

interface RoleFieldLookupInput {
  readonly roles: readonly AuthRole[]
  readonly roleIdentifier: string
  readonly lowerIdentifier: string
  readonly field: 'alias' | 'name'
}

/**
 * 权限管理模块 - 通过命令管理用户角色
 */
export class AuthModule implements RuntimeModuleInstance {
  readonly meta: RuntimeModuleMeta = {
    name: 'auth',
    description: '权限管理模块 - 管理用户角色',
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
      registerAuthCommands(this.ctx, this.meta)
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

function registerAuthCommands(ctx: Context, meta: RuntimeModuleMeta): void {
  registerAuthRootCommand(ctx, meta)
  registerAuthListCommand(ctx, meta)
  registerAuthInfoCommand(ctx, meta)
  registerAuthAddCommand(ctx, meta)
  registerAuthRemoveCommand(ctx, meta)
}

function registerAuthRootCommand(ctx: Context, meta: RuntimeModuleMeta): void {
  registerRuntimeCommand(ctx, meta, {
    name: 'gauth',
    desc: '管理用户角色',
    permNode: 'gauth',
    permDesc: '管理用户角色（主命令）',
    usage: '角色管理系统，使用子命令操作',
  })
}

function registerAuthListCommand(ctx: Context, meta: RuntimeModuleMeta): void {
  registerRuntimeCommand(ctx, meta, {
    name: 'gauth.list',
    desc: '列出所有可用角色',
    permNode: 'gauth.list',
    permDesc: '列出所有可用角色',
    usage: '显示系统中所有可分配的角色',
  })
    .action(async () => formatRoleList(ctx))
}

function registerAuthInfoCommand(ctx: Context, meta: RuntimeModuleMeta): void {
  registerRuntimeCommand(ctx, meta, {
    name: 'gauth.info',
    desc: '查看用户的角色',
    args: '<target:user>',
    permNode: 'gauth.info',
    permDesc: '查看用户的角色',
    usage: '查看指定用户所拥有的角色',
    examples: ['gauth.info @用户'],
  })
    .example('gauth.info @可爱猫娘')
    .example('gauth.info 123456')
    .action(async (_, target) => handleRoleInfo(ctx, target))
}

function registerAuthAddCommand(ctx: Context, meta: RuntimeModuleMeta): void {
  registerRuntimeCommand(ctx, meta, {
    name: 'gauth.add',
    desc: '给用户添加角色',
    args: '<target:user> <roleIdentifier:text>',
    permNode: 'gauth.add',
    permDesc: '给用户添加角色',
    usage: '给指定用户分配角色',
    examples: ['gauth.add @用户 admin'],
  })
    .example('gauth.add @可爱猫娘 admin')
    .example('gauth.add @可爱猫娘 管理员')
    .example('gauth.add 123456 moderator')
    .action(async (_, target, roleIdentifier) => {
      return handleRoleMutation({ ctx, target, roleIdentifier, operation: 'add' })
    })
}

function registerAuthRemoveCommand(ctx: Context, meta: RuntimeModuleMeta): void {
  registerRuntimeCommand(ctx, meta, {
    name: 'gauth.remove',
    desc: '从用户移除角色',
    args: '<target:user> <roleIdentifier:text>',
    permNode: 'gauth.remove',
    permDesc: '从用户移除角色',
    usage: '从指定用户撤销角色',
    examples: ['gauth.remove @用户 admin'],
  })
    .alias('gauth.rm')
    .example('gauth.remove @可爱猫娘 admin')
    .example('gauth.remove @可爱猫娘 管理员')
    .example('gauth.rm 123456 moderator')
    .action(async (_, target, roleIdentifier) => {
      return handleRoleMutation({ ctx, target, roleIdentifier, operation: 'remove' })
    })
}

function formatRoleList(ctx: Context): string {
  const roles = ctx.stuhelperGroupCenter.auth.getRoles()
  if (roles.length === 0) return '暂无可用角色'

  const lines = ['可用角色列表:']
  for (const role of roles) {
    const isBuiltin = isBuiltinRoleId(role.id)
    const tag = isBuiltin ? '[内置]' : ''
    const memberCount = isBuiltin
      ? '-'
      : ctx.stuhelperGroupCenter.auth.getRoleMembers(role.id).length
    lines.push(`• ${role.name} (${role.id}) ${tag} - ${memberCount} 成员`)
  }
  return lines.join('\n')
}

function handleRoleInfo(ctx: Context, target: unknown): string {
  if (!target) return '请指定要查询的用户'

  const userId = resolveCommandUserId(target)
  if (!userId) return '无法解析用户 ID'

  const userRoleIds = ctx.stuhelperGroupCenter.auth.getUserRoleIds(userId)
  const allRoles = ctx.stuhelperGroupCenter.auth.getRoles()
  if (userRoleIds.length === 0) return `用户 ${userId} 暂无自定义角色`

  const lines = [`用户 ${userId} 的角色:`]
  for (const roleId of userRoleIds) {
    const role = allRoles.find(item => item.id === roleId)
    lines.push(`• ${role?.name || roleId} (${roleId})`)
  }
  return lines.join('\n')
}

async function handleRoleMutation(input: RoleMutationInput): Promise<string> {
  const missingMessage = input.operation === 'add' ? '请指定要添加的角色 ID 或名称' : '请指定要移除的角色 ID 或名称'
  if (!input.target) return '请指定要操作的用户'
  if (!input.roleIdentifier) return missingMessage

  const userId = resolveCommandUserId(input.target)
  if (!userId) return '无法解析用户 ID'

  const lookup = findRole(input.ctx, input.roleIdentifier)
  if (!lookup.role) return formatMissingRole(input.roleIdentifier, input.operation)
  if (isBuiltinRoleId(lookup.role.id)) {
    return `"${lookup.role.name}" 是内置角色，由系统自动分配，不支持手动${input.operation === 'add' ? '添加' : '移除'}`
  }
  return input.operation === 'add'
    ? assignRole(input.ctx, userId, lookup.role, lookup.warning)
    : revokeRole(input.ctx, userId, lookup.role, lookup.warning)
}

function isBuiltinRoleId(roleId: string): boolean {
  return BUILTIN_ROLE_ID_SET.has(roleId)
}

function findRole(ctx: Context, roleIdentifier: string): RoleLookup {
  const roles = ctx.stuhelperGroupCenter.auth.getRoles()
  const exactRole = roles.find(role => role.id === roleIdentifier)
  if (exactRole) return { role: exactRole }

  const lowerIdentifier = roleIdentifier.toLowerCase()
  const aliasLookup = findRoleByField({
    roles,
    roleIdentifier,
    lowerIdentifier,
    field: 'alias',
  })
  if (aliasLookup.role) return aliasLookup
  return findRoleByField({
    roles,
    roleIdentifier,
    lowerIdentifier,
    field: 'name',
  })
}

function findRoleByField(input: RoleFieldLookupInput): RoleLookup {
  const matches = input.roles.filter(role => role[input.field]?.toLowerCase() === input.lowerIdentifier)
  if (matches.length === 0) return { role: null }
  if (matches.length === 1) return { role: matches[0] }

  const warning = input.field === 'alias'
    ? `存在 ${matches.length} 个角色使用相同别名 "${input.roleIdentifier}"，已匹配第一个：${matches[0].name}`
    : `存在 ${matches.length} 个角色使用相同名称 "${input.roleIdentifier}"，已匹配第一个：${matches[0].name} (${matches[0].id})`
  return { role: matches[0], warning }
}

function formatMissingRole(roleIdentifier: string, operation: 'add' | 'remove'): string {
  const suffix = operation === 'add' ? '，使用 gauth.list 查看可用角色' : ''
  return `角色 "${roleIdentifier}" 不存在${suffix}`
}

async function assignRole(ctx: Context, userId: string, role: AuthRole, warning?: string): Promise<string> {
  try {
    await ctx.stuhelperGroupCenter.auth.assignRole(userId, role.id)
    const message = `已将用户 ${userId} 添加到角色 "${role.name}"`
    return warning ? `${message}\n⚠️ ${warning}` : message
  } catch (error) {
    return `添加失败: ${getErrorMessage(error)}`
  }
}

async function revokeRole(ctx: Context, userId: string, role: AuthRole, warning?: string): Promise<string> {
  const userRoleIds = ctx.stuhelperGroupCenter.auth.getUserRoleIds(userId)
  if (!userRoleIds.includes(role.id)) {
    return `用户 ${userId} 没有角色 "${role.name}"`
  }

  try {
    await ctx.stuhelperGroupCenter.auth.revokeRole(userId, role.id)
    const message = `已从用户 ${userId} 移除角色 "${role.name}"`
    return warning ? `${message}\n⚠️ ${warning}` : message
  } catch (error) {
    return `移除失败: ${getErrorMessage(error)}`
  }
}

function getErrorMessage(error: unknown): string {
  if (error instanceof Error) return error.message
  return String(error)
}

export const authRuntimeModule: RuntimeModule<AuthModule> = {
  id: 'auth',
  create(ctx) {
    return new AuthModule(ctx)
  },
}
