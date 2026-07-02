import { DEFAULT_REPEAT_THRESHOLD } from './constants'
import { testSafeKeywordRegex } from '@stuhelper/koishi-shared'
import type {
  KeywordHit,
  KeywordMatchContext,
  KeywordRuleRecord,
  MessageLedgerRecord,
} from './types'

export {
  type KeywordRuleRecord,
} from './types'

interface RepeatInput {
  normalizedContent: string
  memberId: string
}

export function normalizeModerationContent(content: string) {
  return content.toLowerCase().replace(/\s+/g, ' ').trim()
}

export function detectRepeatTrigger(records: RepeatInput[], normalizedContent: string, threshold = DEFAULT_REPEAT_THRESHOLD) {
  let count = 1
  for (let index = records.length - 1; index >= 0; index -= 1) {
    if (records[index].normalizedContent !== normalizedContent) {
      break
    }
    count += 1
  }
  return { hit: count >= threshold, count }
}

export interface KeywordRuleMatchFailure {
  readonly rule: KeywordRuleRecord
  readonly error: unknown
}

export type KeywordRuleErrorHandler = (failure: KeywordRuleMatchFailure) => void

export function matchKeywordRules(
  rules: KeywordRuleRecord[],
  context: KeywordMatchContext,
  onRuleError?: KeywordRuleErrorHandler,
): KeywordHit[] {
  return rules
    .filter((rule) => rule.enabled)
    .filter((rule) => rule.guildId === context.guildId || rule.guildId === '*')
    .filter((rule) => isRuleMatchedSafely(rule, context, onRuleError))
    .map((rule) => ({
      ruleId: rule.id,
      action: rule.action,
      muteSeconds: rule.muteSeconds,
      note: rule.note,
    }))
}

export function createMessageLedgerPreview(record: Pick<MessageLedgerRecord, 'memberId' | 'content'>) {
  return `${record.memberId}: ${record.content}`
}

function isRuleMatchedSafely(
  rule: KeywordRuleRecord,
  context: KeywordMatchContext,
  onRuleError?: KeywordRuleErrorHandler,
) {
  try {
    return isRuleMatched(rule, context)
  } catch (error) {
    onRuleError?.({ rule, error })
    return false
  }
}

function isRuleMatched(rule: KeywordRuleRecord, context: KeywordMatchContext) {
  if (rule.matchMode === 'includes') {
    return context.normalizedContent.includes(normalizeModerationContent(rule.pattern))
  }
  return testSafeKeywordRegex(rule.pattern, context.content)
}
