import type { Role } from '../types'

interface RoleReorderState {
  readonly roleOperation?: string
  readonly savingChanges?: boolean
  readonly loading?: boolean
  readonly hasLoadError?: boolean
  readonly hasChanges?: boolean
}

export function canDragRoleForReorder(role: Role, state: RoleReorderState = {}): boolean {
  return isCustomRole(role) && isRoleReorderReady(state)
}

export function canDropRoleForReorder(source: Role | null | undefined, target: Role, state: RoleReorderState = {}): boolean {
  return Boolean(
    source &&
    source.id !== target.id &&
    isCustomRole(source) &&
    isCustomRole(target) &&
    isRoleReorderReady(state),
  )
}

function isCustomRole(role: Role): boolean {
  return !role.builtin
}

function isRoleReorderReady(state: RoleReorderState): boolean {
  return (
    !state.roleOperation &&
    !state.savingChanges &&
    !state.loading &&
    !state.hasLoadError &&
    !state.hasChanges
  )
}
