import { Context, Service, Session } from 'koishi'
import { DataManager } from '../data'
import { Role, PermissionNode, RegisteredCommand } from '../../types'
import { BUILTIN_ROLE_IDS, type BuiltinRoleId, createBuiltinRoles } from './auth-builtins'
import { collectSessionPermissions, hasPermissionNode } from './auth-permissions'

export { BUILTIN_ROLE_IDS, type BuiltinRoleId }

interface RegisterPermissionInput {
  readonly id: string
  readonly name: string
  readonly description: string
  readonly group?: string
}

export class AuthService {
  /** 动态注册的权限节点 */
  private _permissions: Map<string, PermissionNode> = new Map()
  /** 已注册的模块通配符 */
  private _moduleWildcards: Set<string> = new Set()
  /** 已注册的命令（用于动态生成帮助） */
  private _commands: Map<string, RegisteredCommand> = new Map()

  constructor(
    private ctx: Context,
    private data: DataManager
  ) {
    // 注册系统内置权限
    this.registerPermission({
      id: '*',
      name: '超级管理员',
      description: '拥有所有权限',
      group: '系统',
    })
    
    // 初始化内置角色
    this.initBuiltinRoles()
  }

  /**
   * 初始化内置角色（如果不存在则创建）
   */
  private initBuiltinRoles(): void {
    const roles = this.data.authRoles.get('roles') || {}
    let changed = false

    for (const role of createBuiltinRoles()) {
      if (!roles[role.id]) {
        roles[role.id] = role
        changed = true
        this.ctx.logger('stuhelperGroupCenter').info('创建内置角色:', role.id)
      } else {
        // 确保 builtin 标记存在
        if (!roles[role.id].builtin) {
          roles[role.id].builtin = true
          changed = true
        }
      }
    }

    if (changed) {
      this.data.authRoles.set('roles', roles)
      this.data.authRoles.flush()
    }
  }

  /**
   * 检查角色是否为内置角色
   */
  isBuiltinRole(roleId: string): boolean {
    return BUILTIN_ROLE_IDS.includes(roleId as BuiltinRoleId)
  }

  /**
   * 注册权限节点
   * @param id 权限节点ID（如 warn.add）
   * @param name 权限名称
   * @param description 权限描述
   * @param group 分组名称（用于前端显示）
   */
  registerPermission(input: RegisterPermissionInput): void {
    const { id, name, description, group } = input
    this._permissions.set(id, { id, name, description, group })
    
    // 自动注册模块级通配符权限（如 warn.* ）
    const moduleName = id.split('.')[0]
    if (moduleName && moduleName !== '*' && moduleName !== id && !this._moduleWildcards.has(moduleName)) {
      this._moduleWildcards.add(moduleName)
      const wildcardId = `${moduleName}.*`
      if (!this._permissions.has(wildcardId)) {
        this._permissions.set(wildcardId, {
          id: wildcardId,
          name: `${moduleName} 模块全部权限`,
          description: `拥有 ${moduleName} 模块的所有权限`,
          group: group || '通配符'
        })
      }
    }
  }

  /**
   * 获取所有已注册的权限节点
   */
  getPermissions(): PermissionNode[] {
    return Array.from(this._permissions.values())
  }

  /**
   * 注册命令信息（用于动态生成帮助）
   */
  registerCommand(cmd: RegisteredCommand): void {
    this._commands.set(cmd.name, cmd)
  }

  /**
   * 获取所有已注册的命令
   */
  getCommands(): RegisteredCommand[] {
    return Array.from(this._commands.values())
  }

  /**
   * 按模块分组获取命令
   */
  getCommandsByModule(): Map<string, RegisteredCommand[]> {
    const grouped = new Map<string, RegisteredCommand[]>()
    for (const cmd of this._commands.values()) {
      const key = cmd.moduleDesc || cmd.module
      let commands = grouped.get(key)
      if (!commands) {
        commands = []
        grouped.set(key, commands)
      }
      commands.push(cmd)
    }
    return grouped
  }

  /**
   * 创建或更新角色
   */
  async saveRole(role: Role): Promise<void> {
    const roles = this.data.authRoles.get('roles') || {}
    roles[role.id] = role
    this.data.authRoles.set('roles', roles)
  }

  /**
   * 删除角色（内置角色不可删除）
   */
  async deleteRole(roleId: string): Promise<void> {
    // 内置角色不可删除
    if (this.isBuiltinRole(roleId)) {
      throw new Error('内置角色不可删除')
    }
    
    const roles = this.data.authRoles.get('roles') || {}
    if (roles[roleId]) {
      delete roles[roleId]
      this.data.authRoles.set('roles', roles)
      
      // 同时清理所有用户关联的该角色
      const users = this.data.authUsers.get('users') || {}
      let changed = false
      for (const userId in users) {
        const userRoles = users[userId]
        if (userRoles.includes(roleId)) {
          users[userId] = userRoles.filter(id => id !== roleId)
          changed = true
        }
      }
      if (changed) {
        this.data.authUsers.set('users', users)
      }
    }
  }

  /**
   * 获取所有角色
   */
  getRoles(): Role[] {
    const roles = this.data.authRoles.get('roles') || {}
    return Object.values(roles).sort((a, b) => b.priority - a.priority)
  }

  /**
   * 获取用户的角色列表
   * 支持两种用户 ID 格式匹配：
   * - 完整格式: "onebot:123456"
   * - 简短格式: "123456" (纯 QQ 号)
   */
  getUserRoleIds(userId: string): string[] {
    const users = this.data.authUsers.get('users') || {}
    
    // 优先精确匹配
    if (users[userId]) {
      return users[userId]
    }
    
    // 如果是 "platform:id" 格式，尝试匹配纯 id
    if (userId.includes(':')) {
      const pureId = userId.split(':').pop()
      if (pureId && users[pureId]) {
        return users[pureId]
      }
    }
    
    // 如果是纯 id 格式，尝试匹配各平台的完整格式
    for (const storedId in users) {
      if (storedId.endsWith(':' + userId)) {
        return users[storedId]
      }
    }
    
    return []
  }

  /**
   * 获取拥有某角色的所有用户ID
   */
  getRoleMembers(roleId: string): string[] {
    const users = this.data.authUsers.get('users') || {}
    const memberIds: string[] = []
    for (const userId in users) {
      if (users[userId].includes(roleId)) {
        memberIds.push(userId)
      }
    }
    return memberIds
  }

  /**
   * 给用户分配角色（内置角色不可手动分配）
   */
  async assignRole(userId: string, roleId: string): Promise<void> {
    // 内置角色不可手动分配
    if (this.isBuiltinRole(roleId)) {
      throw new Error('内置角色由系统自动分配，不支持手动添加成员')
    }
    
    const users = this.data.authUsers.get('users') || {}
    const userRoles = users[userId] || []
    if (!userRoles.includes(roleId)) {
      userRoles.push(roleId)
      users[userId] = userRoles
      this.data.authUsers.set('users', users)
    }
  }

  /**
   * 移除用户的角色（内置角色不可手动移除）
   */
  async revokeRole(userId: string, roleId: string): Promise<void> {
    // 内置角色不可手动移除
    if (this.isBuiltinRole(roleId)) {
      throw new Error('内置角色由系统自动分配，不支持手动移除成员')
    }
    
    const users = this.data.authUsers.get('users') || {}
    const userRoles = users[userId] || []
    if (userRoles.includes(roleId)) {
      users[userId] = userRoles.filter(id => id !== roleId)
      this.data.authUsers.set('users', users)
    }
  }

  /**
   * 获取用户的所有权限节点
   * @param session 会话对象
   *
   * 权限来源：
   * 1. 内置角色（authority1-4+）根据 Koishi 原生权限等级自动分配
   * 2. 内置角色（guild-admin）根据群管理员身份自动分配（仅当前群生效）
   * 3. 用户被手动分配的自定义角色
   */
  getUserPermissions(session: Session): Set<string> {
    const user = session.userId
    if (!user) return new Set()
    const roles = this.data.authRoles.get('roles') || {}
    return collectSessionPermissions({ session, roles, userRoleIds: this.getUserRoleIds(user) })
  }

  /**
   * 检查用户是否有指定权限
   *
   * 权限来源（按优先级）：
   * 1. 用户被分配的角色权限
   * 2. Koishi 原生 authority 等级对应的默认权限（需在 WebUI 中配置）
   *
   * 注意：不再对高 authority 用户自动放行，必须在 WebUI 配置相应权限
   */
  check(session: Session, node: string): boolean {
    return hasPermissionNode(this.getUserPermissions(session), node)
  }
}
