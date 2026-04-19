import type { CommandPolicyRecord } from './types'

export interface CommandAccessInput {
  authority: number
  memberRoles: string[]
  policy?: CommandPolicyRecord
}

export function canExecuteCommand(input: CommandAccessInput) {
  if (!input.policy) {
    return true
  }
  if (input.authority >= input.policy.minAuthority) {
    return true
  }
  if (!input.policy.roles.length) {
    return false
  }
  return input.policy.roles.some((role) => input.memberRoles.includes(role))
}
