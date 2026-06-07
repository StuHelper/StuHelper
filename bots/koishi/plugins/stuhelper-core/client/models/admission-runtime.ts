export interface AdmissionRuntimePageData {
  generatedAt: string
  platform: {
    baseUrl: string
    serviceTokenConfigured: boolean
  }
  guard: {
    targetGroups: string[]
    muteDurationSeconds: number
    kickAfterMinutes: number
    reminderTemplate: string
    exemptUserCount: number
  }
  scheduler: {
    fallbackScanEnabled: boolean
    scanIntervalSeconds: number
  }
  actionStream: {
    enabled: boolean
    reconnectDelaySeconds?: number
  }
  commands: {
    publicCommandsRegistered: boolean
    publicCommandsEnabled: boolean
    admissionCommandsRegistered: boolean
    admissionCommandsEnabled: boolean
    admissionCommandMinAuthority?: number
    admissionCommandOperatorQQIDCount: number
  }
  moderation: {
    enabled: boolean
    keywordRuleCount: number
    repeatThreshold: number
    repeatWindowSize: number
    antiRecallNotify: boolean
  }
  freshmanForward: {
    enabled: boolean
  }
  bots: AdmissionRuntimeBot[]
  stats: {
    targetGroupCount: number
    templateCount: number
    bindingCount: number
    enabledBindingCount: number
    activeMemberCount: number
    backendSyncPendingCount: number
    membersWithAdmissionSessionCount: number
    membersWithLastErrorCount: number
  }
  templates: AdmissionRuntimeTemplate[]
  bindings: AdmissionRuntimeBinding[]
  activeMembers: AdmissionRuntimeMember[]
}

export interface AdmissionRuntimeBot {
  platform: string
  selfId: string
  status: string
}

export interface AdmissionRuntimeTemplate {
  id: string
  name: string
  enabled: boolean
  muteDurationSeconds: number
  kickAfterMinutes: number
  exemptUserCount: number
  updatedAt: string
}

export interface AdmissionRuntimeBinding {
  id: string
  platform: string
  guildId: string
  templateId: string
  enabled: boolean
  note: string | null
  updatedAt: string
}

export interface AdmissionRuntimeMember {
  id: string
  platform: string
  botSelfId: string
  guildId: string
  channelId: string
  memberId: string
  memberName: string
  verificationState: string
  admissionSessionID: string | null
  backendSyncPending: boolean
  joinedAt: string
  deadlineAt: string
  nextReminderAt: string | null
  manualReviewDeadlineAt: string | null
  mutedAt: string | null
  reminderSentAt: string | null
  lastError: string | null
  availableActions: AdmissionRuntimeAction[]
}

export type AdmissionRuntimeAction =
  | 'query'
  | 'resend'
  | 'regenerate'
  | 'skip'
  | 'reset-failures'
  | 'release-blacklist'

export interface AdmissionMetric {
  label: string
  value: number | string
  note: string
  tone?: 'primary' | 'warning' | 'danger' | 'success'
}

export interface AdmissionSwitchRow {
  id: string
  label: string
  value: boolean | string | number
  tone: 'success' | 'warning' | 'danger' | 'primary'
  note: string
  editable?: boolean
  settingKey?: AdmissionRuntimeSettingsKey
}

export type AdmissionRuntimeSettingsKey =
  | 'actionStreamEnabled'
  | 'publicCommandsEnabled'
  | 'admissionCommandsEnabled'
  | 'moderationEnabled'
  | 'freshmanForwardEnabled'
  | 'fallbackScanEnabled'

export type AdmissionRuntimeSettingsPatch = Partial<Record<AdmissionRuntimeSettingsKey, boolean>>

export function buildAdmissionRuntimeModel(data: AdmissionRuntimePageData) {
  return {
    metrics: buildMetrics(data),
    switchRows: buildSwitchRows(data),
    enabledBindings: data.bindings.filter((binding) => binding.enabled),
    disabledBindings: data.bindings.filter((binding) => !binding.enabled),
    activeMembers: [...data.activeMembers].sort((left, right) => (
      Date.parse(left.deadlineAt) - Date.parse(right.deadlineAt)
    )),
  }
}

function buildMetrics(data: AdmissionRuntimePageData): AdmissionMetric[] {
  const activeTargetGuildCount = countActiveTargetGuilds(data)
  return [
    {
      label: '目标群',
      value: activeTargetGuildCount,
      note: `${data.stats.targetGroupCount} 个静态目标群，${data.stats.enabledBindingCount} 个启用绑定`,
      tone: activeTargetGuildCount > 0 ? 'success' : 'warning',
    },
    {
      label: '受限成员',
      value: data.stats.activeMemberCount,
      note: `${data.stats.backendSyncPendingCount} 个等待后端同步`,
      tone: data.stats.activeMemberCount > 0 ? 'warning' : 'success',
    },
    {
      label: '认证会话',
      value: data.stats.membersWithAdmissionSessionCount,
      note: '本地队列中已有后端 session 的成员',
      tone: data.stats.membersWithAdmissionSessionCount > 0 ? 'primary' : undefined,
    },
    {
      label: '错误',
      value: data.stats.membersWithLastErrorCount,
      note: '本地队列中最近一次 admission 错误',
      tone: data.stats.membersWithLastErrorCount > 0 ? 'danger' : 'success',
    },
  ]
}

function countActiveTargetGuilds(data: AdmissionRuntimePageData): number {
  const guildIds = new Set(data.guard.targetGroups.map((guildId) => guildId.trim()).filter(Boolean))
  for (const binding of data.bindings) {
    if (binding.enabled && binding.guildId.trim()) {
      guildIds.add(binding.guildId.trim())
    }
  }
  return guildIds.size
}

function buildSwitchRows(data: AdmissionRuntimePageData): AdmissionSwitchRow[] {
  return [
    {
      id: 'service-token',
      label: '服务凭据',
      value: data.platform.serviceTokenConfigured,
      tone: data.platform.serviceTokenConfigured ? 'success' : 'danger',
      note: data.platform.baseUrl,
    },
    {
      id: 'action-stream',
      label: 'Action Stream',
      value: data.actionStream.enabled,
      tone: data.actionStream.enabled ? 'success' : 'warning',
      note: `重连 ${data.actionStream.reconnectDelaySeconds ?? 0} 秒`,
      editable: true,
      settingKey: 'actionStreamEnabled',
    },
    {
      id: 'fallback-scan',
      label: '兜底扫描',
      value: data.scheduler.fallbackScanEnabled,
      tone: data.scheduler.fallbackScanEnabled ? 'warning' : 'primary',
      note: `${data.scheduler.scanIntervalSeconds} 秒`,
      editable: true,
      settingKey: 'fallbackScanEnabled',
    },
    {
      id: 'public-commands',
      label: '公开命令',
      value: data.commands.publicCommandsEnabled,
      tone: data.commands.publicCommandsEnabled && data.commands.publicCommandsRegistered ? 'warning' : 'success',
      note: data.commands.publicCommandsRegistered ? '举报 / 骰子 / 抽禁言' : '启动未注册，需重启后启用',
      editable: data.commands.publicCommandsRegistered,
      settingKey: 'publicCommandsEnabled',
    },
    {
      id: 'admission-commands',
      label: '准入命令',
      value: data.commands.admissionCommandsEnabled,
      tone: data.commands.admissionCommandsEnabled && data.commands.admissionCommandsRegistered ? 'success' : 'warning',
      note: data.commands.admissionCommandsRegistered ? `authority ${data.commands.admissionCommandMinAuthority ?? 0}` : '启动未注册，需重启后启用',
      editable: data.commands.admissionCommandsRegistered,
      settingKey: 'admissionCommandsEnabled',
    },
    {
      id: 'moderation',
      label: '消息风控',
      value: data.moderation.enabled,
      tone: data.moderation.enabled ? 'warning' : 'success',
      note: `${data.moderation.keywordRuleCount} 条启动规则`,
      editable: true,
      settingKey: 'moderationEnabled',
    },
    {
      id: 'freshman-forward',
      label: '材料转发',
      value: data.freshmanForward.enabled,
      tone: data.freshmanForward.enabled ? 'warning' : 'success',
      note: '新生原始材料转发扫描',
      editable: true,
      settingKey: 'freshmanForwardEnabled',
    },
  ]
}
