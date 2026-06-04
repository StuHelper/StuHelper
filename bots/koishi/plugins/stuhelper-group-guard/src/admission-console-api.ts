import type {} from '@koishijs/plugin-console'
import type { Context } from 'koishi'

import type {
  GuardPolicyStore,
  StuhelperGroupGuardPluginConfig,
} from '@stuhelper/koishi-shared'

import type { GuardMemberRecord } from './model'
import type { GuardMemberStore } from './store'

const ADMISSION_RUNTIME_PAGE_EVENT = 'stuhelperGroupGuard/page/admission-runtime'
const CONSOLE_AUTHORITY = 4
const ACTIVE_MEMBER_LIMIT = 100

interface AdmissionConsoleAPIDeps {
  readonly config: StuhelperGroupGuardPluginConfig
  readonly guardStore: GuardMemberStore
  readonly policyStore: GuardPolicyStore
}

declare module '@koishijs/console' {
  interface Events {
    [ADMISSION_RUNTIME_PAGE_EVENT](): ReturnType<typeof buildAdmissionRuntimePageData>
  }
}

export function registerAdmissionConsoleAPI(ctx: Context, deps: AdmissionConsoleAPIDeps) {
  if (!ctx.console) {
    return
  }

  ctx.console.addListener(ADMISSION_RUNTIME_PAGE_EVENT, async () => {
    return buildAdmissionRuntimePageData(ctx, deps)
  }, { authority: CONSOLE_AUTHORITY })
}

export async function buildAdmissionRuntimePageData(ctx: Context, deps: AdmissionConsoleAPIDeps) {
  const [activeMembers, templates, bindings] = await Promise.all([
    deps.guardStore.listActive(),
    deps.policyStore.listTemplates(),
    deps.policyStore.listBindings(),
  ])
  const sortedMembers = [...activeMembers]
    .sort((left, right) => left.deadlineAt.getTime() - right.deadlineAt.getTime())
    .slice(0, ACTIVE_MEMBER_LIMIT)

  return {
    generatedAt: new Date().toISOString(),
    platform: {
      baseUrl: redactURL(deps.config.platform.baseUrl),
      serviceTokenConfigured: Boolean(deps.config.platform.serviceToken),
    },
    guard: {
      targetGroups: [...deps.config.guard.targetGroups],
      muteDurationSeconds: deps.config.guard.muteDurationSeconds,
      kickAfterMinutes: deps.config.guard.kickAfterMinutes,
      reminderTemplate: deps.config.guard.reminderTemplate,
      exemptUserCount: deps.config.guard.exemptUsers.length,
    },
    scheduler: {
      fallbackScanEnabled: deps.config.scheduler.fallbackScanEnabled !== false,
      scanIntervalSeconds: deps.config.scheduler.scanIntervalSeconds,
    },
    actionStream: {
      enabled: deps.config.actionStream?.enabled !== false,
      reconnectDelaySeconds: deps.config.actionStream?.reconnectDelaySeconds,
    },
    commands: {
      publicCommandsEnabled: deps.config.commands?.enabled !== false,
      admissionCommandsEnabled: deps.config.admissionCommands?.enabled !== false,
      admissionCommandMinAuthority: deps.config.admissionCommands?.minAuthority,
      admissionCommandOperatorQQIDCount: deps.config.admissionCommands?.operatorQQIDs?.length ?? 0,
    },
    moderation: {
      enabled: deps.config.moderation.enabled !== false,
      keywordRuleCount: deps.config.moderation.keywordRules.length,
      repeatThreshold: deps.config.moderation.repeatThreshold,
      repeatWindowSize: deps.config.moderation.repeatWindowSize,
      antiRecallNotify: deps.config.moderation.antiRecallNotify,
    },
    freshmanForward: {
      enabled: deps.config.freshmanForward?.enabled !== false,
    },
    bots: ctx.bots.map((bot) => ({
      platform: bot.platform,
      selfId: bot.selfId,
      status: String((bot as { status?: unknown }).status ?? 'unknown'),
    })),
    stats: {
      targetGroupCount: deps.config.guard.targetGroups.length,
      templateCount: templates.length,
      bindingCount: bindings.length,
      enabledBindingCount: bindings.filter((binding) => binding.enabled).length,
      activeMemberCount: activeMembers.length,
      backendSyncPendingCount: activeMembers.filter((record) => record.backendSyncPending).length,
      membersWithAdmissionSessionCount: activeMembers.filter((record) => Boolean(record.admissionSessionID)).length,
      membersWithLastErrorCount: activeMembers.filter((record) => Boolean(record.lastError)).length,
    },
    templates: templates.map((template) => ({
      id: template.id,
      name: template.name,
      enabled: template.enabled,
      muteDurationSeconds: template.muteDurationSeconds,
      kickAfterMinutes: template.kickAfterMinutes,
      exemptUserCount: template.exemptUsers.length,
      updatedAt: template.updatedAt.toISOString(),
    })),
    bindings: bindings.map((binding) => ({
      id: binding.id,
      platform: binding.platform,
      guildId: binding.guildId,
      templateId: binding.templateId,
      enabled: binding.enabled,
      note: binding.note,
      updatedAt: binding.updatedAt.toISOString(),
    })),
    activeMembers: sortedMembers.map(serializeGuardMember),
  }
}

function serializeGuardMember(record: GuardMemberRecord) {
  return {
    id: record.id,
    platform: record.platform,
    botSelfId: record.botSelfId,
    guildId: record.guildId,
    channelId: record.channelId,
    memberId: record.memberId,
    memberName: record.memberName,
    verificationState: record.verificationState,
    admissionSessionID: record.admissionSessionID,
    backendSyncPending: record.backendSyncPending,
    joinedAt: record.joinedAt.toISOString(),
    deadlineAt: record.deadlineAt.toISOString(),
    nextReminderAt: record.nextReminderAt?.toISOString() ?? null,
    manualReviewDeadlineAt: record.manualReviewDeadlineAt?.toISOString() ?? null,
    mutedAt: record.mutedAt?.toISOString() ?? null,
    reminderSentAt: record.reminderSentAt?.toISOString() ?? null,
    lastError: record.lastError,
  }
}

function redactURL(value: string) {
  try {
    const url = new URL(value)
    url.username = ''
    url.password = ''
    return url.toString().replace(/\/$/, '')
  } catch {
    return value
  }
}
