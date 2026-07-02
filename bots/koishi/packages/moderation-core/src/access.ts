import type { CommandPolicyRecord } from './types'

export interface CommandAccessInput {
  authority: number
  memberRoles: readonly string[]
  policy: CommandPolicyRecord
}

export function canExecuteCommand(input: CommandAccessInput) {
  if (input.authority >= input.policy.minAuthority) {
    return true
  }
  if (!input.policy.roles.length) {
    return false
  }
  return input.policy.roles.some((role) => input.memberRoles.includes(role))
}

export function createFallbackCommandPolicy(commandId: string, minAuthority: number): CommandPolicyRecord {
  return {
    commandId,
    roles: [],
    minAuthority,
    createdAt: new Date(0),
    updatedAt: new Date(0),
  }
}
