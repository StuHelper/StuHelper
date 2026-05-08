export interface PermissionNode {
  id: string
  name: string
  description: string
  group?: string
}

export interface Role {
  id: string
  name: string
  alias?: string
  color?: string
  priority: number
  permissions: string[]
  guildIds?: string[]
  builtin?: boolean
}

export interface AuthRolesData extends Record<string, unknown> {
  roles: Record<string, Role>
  defaultLevels: Record<number, string[]>
}

export interface AuthUsersData extends Record<string, unknown> {
  users: Record<string, string[]>
}
