import { send } from '@koishijs/client'

import type {
  StuhelperCommandPolicyInput,
  StuhelperGuardBindingInput,
  StuhelperGuardTemplateInput,
  StuhelperGuardBatchActionInput,
  StuhelperKeywordRuleInput,
  StuhelperMemberRoleInput,
  StuhelperReviewActionInput,
} from '../src/console-types'

export async function refreshConsoleData() {
  return send('stuhelper-console/refresh')
}

export async function runGuardBatchAction(input: StuhelperGuardBatchActionInput) {
  return send('stuhelper-console/guard-action', input)
}

export async function runReviewAction(input: StuhelperReviewActionInput) {
  return send('stuhelper-console/review-action', input)
}

export async function saveKeywordRule(rule: StuhelperKeywordRuleInput) {
  return send('stuhelper-console/save-keyword-rule', rule)
}

export async function saveMemberRoles(input: StuhelperMemberRoleInput) {
  return send('stuhelper-console/save-member-roles', input)
}

export async function saveCommandPolicy(input: StuhelperCommandPolicyInput) {
  return send('stuhelper-console/save-command-policy', input)
}

export async function saveGuardTemplate(input: StuhelperGuardTemplateInput) {
  return send('stuhelper-console/save-guard-template', input)
}

export async function saveGuardBinding(input: StuhelperGuardBindingInput) {
  return send('stuhelper-console/save-guard-binding', input)
}
