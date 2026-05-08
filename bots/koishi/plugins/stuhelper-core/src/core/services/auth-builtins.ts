import type { Role } from '../../types'

export const BUILTIN_ROLE_IDS = [
  'authority1',
  'authority2',
  'authority3',
  'authority4+',
  'guild-admin',
] as const

export type BuiltinRoleId = typeof BUILTIN_ROLE_IDS[number]

export function createBuiltinRoles(): Role[] {
  return [
    createBuiltinRole({ id: 'authority1', name: 'Authority 1', color: '#67c23a', priority: 100 }),
    createBuiltinRole({ id: 'authority2', name: 'Authority 2', color: '#e6a23c', priority: 200 }),
    createBuiltinRole({ id: 'authority3', name: 'Authority 3', color: '#f56c6c', priority: 300 }),
    createBuiltinRole({ id: 'authority4+', name: 'Authority ≥4', color: '#9c27b0', priority: 400 }),
    createBuiltinRole({ id: 'guild-admin', name: '群管理员', color: '#409eff', priority: 50 }),
  ]
}

function createBuiltinRole(input: {
  readonly id: BuiltinRoleId
  readonly name: string
  readonly color: string
  readonly priority: number
}): Role {
  const { id, name, color, priority } = input
  return {
    id,
    name,
    color,
    priority,
    permissions: [],
    guildIds: [],
    builtin: true,
  }
}
