import { Context } from 'koishi'

import {
  MODERATION_COMMAND_POLICY_TABLE,
  MODERATION_EVENT_TABLE,
  MODERATION_FUN_PROFILE_TABLE,
  MODERATION_KEYWORD_RULE_TABLE,
  MODERATION_MEMBER_ROLE_TABLE,
  MODERATION_MESSAGE_LEDGER_TABLE,
  MODERATION_REPORT_TABLE,
  MODERATION_REVIEW_TABLE,
  MODERATION_WARNING_TABLE,
} from './constants'
import type {
  CommandPolicyRecord,
  FunProfileRecord,
  KeywordRuleRecord,
  MemberRoleRecord,
  MessageLedgerRecord,
  ModerationEventRecord,
  ModerationReportRecord,
  ReviewQueueRecord,
  WarningCounterRecord,
} from './types'

declare module 'koishi' {
  interface Tables {
    [MODERATION_EVENT_TABLE]: ModerationEventRecord
    [MODERATION_REVIEW_TABLE]: ReviewQueueRecord
    [MODERATION_MESSAGE_LEDGER_TABLE]: MessageLedgerRecord
    [MODERATION_WARNING_TABLE]: WarningCounterRecord
    [MODERATION_KEYWORD_RULE_TABLE]: KeywordRuleRecord
    [MODERATION_MEMBER_ROLE_TABLE]: MemberRoleRecord
    [MODERATION_COMMAND_POLICY_TABLE]: CommandPolicyRecord
    [MODERATION_FUN_PROFILE_TABLE]: FunProfileRecord
    [MODERATION_REPORT_TABLE]: ModerationReportRecord
  }
}

export function registerModerationModels(ctx: Context) {
  registerEventModel(ctx)
  registerReviewModel(ctx)
  registerMessageLedgerModel(ctx)
  registerWarningModel(ctx)
  registerKeywordRuleModel(ctx)
  registerMemberRoleModel(ctx)
  registerCommandPolicyModel(ctx)
  registerFunProfileModel(ctx)
  registerReportModel(ctx)
}

function registerEventModel(ctx: Context) {
  ctx.model.extend(MODERATION_EVENT_TABLE, {
    id: 'string',
    platform: 'string',
    botSelfId: 'string',
    guildId: 'string',
    channelId: 'string',
    memberId: 'string',
    type: 'string',
    level: 'string',
    summary: 'text',
    payload: { type: 'json', initial: null },
    createdAt: 'timestamp',
    updatedAt: 'timestamp',
  }, { primary: 'id' })
}

function registerReviewModel(ctx: Context) {
  ctx.model.extend(MODERATION_REVIEW_TABLE, {
    id: 'string',
    platform: 'string',
    botSelfId: 'string',
    guildId: 'string',
    channelId: 'string',
    memberId: 'string',
    actionType: 'string',
    status: 'string',
    reason: 'text',
    operatorMemberId: 'string',
    resolutionNote: 'text',
    payload: { type: 'json', initial: null },
    createdAt: 'timestamp',
    updatedAt: 'timestamp',
  }, { primary: 'id' })
}

function registerMessageLedgerModel(ctx: Context) {
  ctx.model.extend(MODERATION_MESSAGE_LEDGER_TABLE, {
    messageId: 'string',
    platform: 'string',
    botSelfId: 'string',
    guildId: 'string',
    channelId: 'string',
    memberId: 'string',
    content: 'text',
    normalizedContent: 'text',
    quoteMessageId: 'string',
    createdAt: 'timestamp',
    deletedAt: 'timestamp',
  }, {
    primary: 'messageId',
    indexes: [{ keys: { guildId: 'asc', createdAt: 'desc' } }],
  })
}

function registerWarningModel(ctx: Context) {
  ctx.model.extend(MODERATION_WARNING_TABLE, {
    id: 'string',
    guildId: 'string',
    memberId: 'string',
    total: 'unsigned',
    lastReason: 'text',
    lastAt: 'timestamp',
    createdAt: 'timestamp',
    updatedAt: 'timestamp',
  }, { primary: 'id' })
}

function registerKeywordRuleModel(ctx: Context) {
  ctx.model.extend(MODERATION_KEYWORD_RULE_TABLE, {
    id: 'string',
    guildId: 'string',
    pattern: 'text',
    matchMode: 'string',
    action: 'string',
    enabled: 'boolean',
    muteSeconds: 'unsigned',
    note: 'text',
    createdAt: 'timestamp',
    updatedAt: 'timestamp',
  }, { primary: 'id' })
}

function registerMemberRoleModel(ctx: Context) {
  ctx.model.extend(MODERATION_MEMBER_ROLE_TABLE, {
    id: 'string',
    guildId: 'string',
    memberId: 'string',
    roles: { type: 'json', initial: [] },
    createdAt: 'timestamp',
    updatedAt: 'timestamp',
  }, { primary: 'id' })
}

function registerCommandPolicyModel(ctx: Context) {
  ctx.model.extend(MODERATION_COMMAND_POLICY_TABLE, {
    commandId: 'string',
    roles: { type: 'json', initial: [] },
    minAuthority: 'unsigned',
    createdAt: 'timestamp',
    updatedAt: 'timestamp',
  }, { primary: 'commandId' })
}

function registerFunProfileModel(ctx: Context) {
  ctx.model.extend(MODERATION_FUN_PROFILE_TABLE, {
    memberId: 'string',
    muteDrawCount: 'unsigned',
    pityBonus: 'unsigned',
    lastDrawAt: 'timestamp',
    createdAt: 'timestamp',
    updatedAt: 'timestamp',
  }, { primary: 'memberId' })
}

function registerReportModel(ctx: Context) {
  ctx.model.extend(MODERATION_REPORT_TABLE, {
    id: 'string',
    platform: 'string',
    botSelfId: 'string',
    guildId: 'string',
    channelId: 'string',
    reporterMemberId: 'string',
    targetMemberId: 'string',
    reason: 'text',
    aiStatus: 'string',
    aiSeverity: 'string',
    aiSummary: 'text',
    createdAt: 'timestamp',
    updatedAt: 'timestamp',
  }, { primary: 'id' })
}
