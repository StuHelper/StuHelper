import { DEFAULT_REPEAT_THRESHOLD } from './constants'
import { evaluateThresholdExpression as evaluateExpression } from './expression'
import type {
  KeywordHit,
  KeywordMatchContext,
  KeywordRuleRecord,
  MessageLedgerRecord,
  ThresholdMetrics,
} from './types'

export {
  type KeywordRuleRecord,
} from './types'

interface RepeatInput {
  normalizedContent: string
  memberId: string
}

export function evaluateThresholdExpression(input: string, metrics: ThresholdMetrics) {
  return evaluateExpression(input, metrics)
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

export function matchKeywordRules(rules: KeywordRuleRecord[], context: KeywordMatchContext): KeywordHit[] {
  return rules
    .filter((rule) => rule.enabled)
    .filter((rule) => rule.guildId === context.guildId || rule.guildId === '*')
    .filter((rule) => isRuleMatched(rule, context))
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

function isRuleMatched(rule: KeywordRuleRecord, context: KeywordMatchContext) {
  if (rule.matchMode === 'includes') {
    return context.normalizedContent.includes(normalizeModerationContent(rule.pattern))
  }
  return new RegExp(rule.pattern, 'i').test(context.content)
}
