import type { Context } from 'koishi'

import {
  renderMessageTemplate,
  resolveAdminMessages,
  resolveGroupGuardMessages,
} from './message-template'
import type {
  StuhelperAdminMessageConfig,
  StuhelperGroupGuardMessageConfig,
} from './types/index'

type I18nContext = Pick<Context, 'i18n'>
type CommandDescriptionBinding<T extends string> = {
  readonly commandName: string
  readonly messageKey: T
}

export const ADMIN_COMMAND_DESCRIPTION_BINDINGS = Object.freeze([
  { commandName: '群审状态', messageKey: 'guardStatusCommandDescription' },
  { commandName: '群审警告', messageKey: 'guardWarningCommandDescription' },
  { commandName: '群审复核', messageKey: 'guardReviewListCommandDescription' },
  { commandName: '群审禁言', messageKey: 'guardBatchMuteCommandDescription' },
  { commandName: '群审踢人申请', messageKey: 'guardKickReviewCommandDescription' },
  { commandName: '群审拉黑申请', messageKey: 'guardBlockReviewCommandDescription' },
] as const satisfies readonly CommandDescriptionBinding<keyof StuhelperAdminMessageConfig>[])

export const GROUP_GUARD_COMMAND_DESCRIPTION_BINDINGS = Object.freeze([
  { commandName: '举报', messageKey: 'publicReportCommandDescription' },
  { commandName: '骰子', messageKey: 'diceCommandDescription' },
  { commandName: '抽禁言', messageKey: 'muteLotteryCommandDescription' },
  { commandName: '查询入群认证', messageKey: 'admissionQueryCommandDescription' },
  { commandName: '重发认证链接', messageKey: 'admissionResendCommandDescription' },
  { commandName: '重新生成认证链接', messageKey: 'admissionRegenerateCommandDescription' },
  { commandName: '跳过入群认证', messageKey: 'admissionSkipCommandDescription' },
  { commandName: '清空入群未认证次数', messageKey: 'admissionResetFailureCountCommandDescription' },
  { commandName: '解除入群拉黑', messageKey: 'admissionReleaseBlacklistCommandDescription' },
] as const satisfies readonly CommandDescriptionBinding<keyof StuhelperGroupGuardMessageConfig>[])

export function syncAdminCommandDescriptions(
  ctx: I18nContext,
  messages?: Partial<StuhelperAdminMessageConfig>,
) {
  syncCommandDescriptions(ctx, ADMIN_COMMAND_DESCRIPTION_BINDINGS, resolveAdminMessages(messages))
}

export function syncGroupGuardCommandDescriptions(
  ctx: I18nContext,
  messages?: Partial<StuhelperGroupGuardMessageConfig>,
) {
  syncCommandDescriptions(ctx, GROUP_GUARD_COMMAND_DESCRIPTION_BINDINGS, resolveGroupGuardMessages(messages))
}

function syncCommandDescriptions<T extends string>(
  ctx: I18nContext,
  bindings: readonly CommandDescriptionBinding<T>[],
  messages: Record<T, string>,
) {
  const dictionary: Record<string, string> = {}
  for (const binding of bindings) {
    dictionary[`commands.${binding.commandName}.description`] = renderMessageTemplate(messages[binding.messageKey])
  }
  ctx.i18n.define('', dictionary)
}
