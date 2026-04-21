import { reactive, ref } from 'vue'

import type {
  StuhelperConsoleCommandPolicy,
  StuhelperConsoleGuardBinding,
  StuhelperConsoleGuardTemplate,
  StuhelperConsoleKeywordRule,
  StuhelperConsoleMemberRole,
  StuhelperKeywordRuleInput,
} from '../src/console-types'

const DEFAULT_COMMAND_POLICY_ID = 'report'

export function useConsoleForms() {
  const selectedGuardIds = ref<string[]>([])
  const guardForm = reactive({
    action: 'mute' as 'mute' | 'unmute' | 'kick' | 'set-role' | 'unset-role',
    seconds: 600,
    reason: '控制台批量操作',
    roleId: '',
    permanent: false,
  })
  const reviewForm = reactive({ note: '' })
  const ruleForm = reactive<StuhelperKeywordRuleInput>({
    id: '',
    guildId: '*',
    pattern: '',
    matchMode: 'includes',
    action: 'warn',
    enabled: true,
    muteSeconds: 0,
    note: '',
  })
  const templateForm = reactive({
    id: '',
    name: '',
    muteDurationSeconds: 600,
    kickAfterMinutes: 30,
    reminderTemplate: '请先完成 StuHelper 注册、QQ 绑定与学生认证。',
    exemptUsersText: '',
    enabled: true,
  })
  const bindingForm = reactive({
    platform: '',
    guildId: '',
    templateId: '',
    enabled: true,
    note: '',
  })
  const roleForm = reactive({ guildId: '', memberId: '', rolesText: '' })
  const policyForm = reactive({
    commandId: DEFAULT_COMMAND_POLICY_ID,
    minAuthority: 0,
    rolesText: '',
  })

  function loadRule(rule: StuhelperConsoleKeywordRule) {
    ruleForm.id = rule.id
    ruleForm.guildId = rule.guildId
    ruleForm.pattern = rule.pattern
    ruleForm.matchMode = rule.matchMode
    ruleForm.action = rule.action
    ruleForm.enabled = rule.enabled
    ruleForm.muteSeconds = rule.muteSeconds
    ruleForm.note = rule.note || ''
  }

  function loadMemberRoles(entry: StuhelperConsoleMemberRole) {
    roleForm.guildId = entry.guildId
    roleForm.memberId = entry.memberId
    roleForm.rolesText = entry.roles.join(', ')
  }

  function loadPolicy(policy: StuhelperConsoleCommandPolicy) {
    policyForm.commandId = policy.commandId
    policyForm.minAuthority = policy.minAuthority
    policyForm.rolesText = policy.roles.join(', ')
  }

  function loadTemplate(template: StuhelperConsoleGuardTemplate) {
    templateForm.id = template.id
    templateForm.name = template.name
    templateForm.muteDurationSeconds = template.muteDurationSeconds
    templateForm.kickAfterMinutes = template.kickAfterMinutes
    templateForm.reminderTemplate = template.reminderTemplate
    templateForm.exemptUsersText = template.exemptUsers.join(', ')
    templateForm.enabled = template.enabled
  }

  function loadBinding(binding: StuhelperConsoleGuardBinding) {
    bindingForm.platform = binding.platform
    bindingForm.guildId = binding.guildId
    bindingForm.templateId = binding.templateId
    bindingForm.enabled = binding.enabled
    bindingForm.note = binding.note || ''
  }

  return {
    selectedGuardIds,
    guardForm,
    reviewForm,
    ruleForm,
    templateForm,
    bindingForm,
    roleForm,
    policyForm,
    loadRule,
    loadMemberRoles,
    loadPolicy,
    loadTemplate,
    loadBinding,
  }
}

export function splitTokens(input: string) {
  return input
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
}
