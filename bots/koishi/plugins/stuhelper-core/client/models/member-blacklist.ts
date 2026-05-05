import type { MemberBlacklistEntry, MemberBlacklistScopeType } from '../types'
import { formatTimestamp } from './formatters'
import type { QueueTableColumn, QueueTableRow } from '../components/primitives/QueueTable.vue'

export interface BlacklistDraft {
  readonly platform: string
  readonly userId: string
  readonly scope: MemberBlacklistScopeType
  readonly guildId: string
}

export const MEMBER_BLACKLIST_COLUMNS: QueueTableColumn[] = [
  { key: 'user', label: '用户 ID' },
  { key: 'scope', label: '范围', width: '160' },
  { key: 'source', label: '来源', width: '180' },
  { key: 'reason', label: '原因' },
  { key: 'time', label: '加入时间', width: '220' },
]

export function canSubmitBlacklistDraft(draft: BlacklistDraft): boolean {
  if (!draft.platform.trim() || !draft.userId.trim()) return false
  return draft.scope === 'global' || Boolean(draft.guildId.trim())
}

export function toBlacklistRows(entries: readonly MemberBlacklistEntry[]): QueueTableRow[] {
  return entries.map((entry) => ({
    id: entry.id,
    cells: {
      user: { text: entry.subjectID },
      scope: { text: formatBlacklistScope(entry), tone: entry.scopeType === 'global' ? 'danger' : 'info' },
      source: formatBlacklistSource(entry),
      reason: entry.reasonText || '未提供',
      time: { text: formatTimestamp(Date.parse(entry.createdAt)), mono: true },
    },
    actions: [{ key: 'remove', label: '移除', tone: 'danger' }],
  }))
}

export function filterBlacklistEntries(
  entries: readonly MemberBlacklistEntry[],
  query: string,
): readonly MemberBlacklistEntry[] {
  if (!query) return entries
  return entries.filter((entry) => {
    return [entry.subjectID, entry.guildID || '', entry.reasonText, entry.source]
      .some((value) => value.toLowerCase().includes(query))
  })
}

export function formatBlacklistScope(entry: Pick<MemberBlacklistEntry, 'scopeType' | 'guildID'>): string {
  return entry.scopeType === 'global' ? '全局' : `群 ${entry.guildID || '未知'}`
}

export function formatBlacklistSource(entry: MemberBlacklistEntry): string {
  if (entry.source === 'manual_admin') return '手动'
  if (entry.source === 'kick_blacklist') return '踢出拉黑'
  if (entry.source === 'moderation_action') return '审核处置'
  if (entry.source === 'admission_failure') return '入群认证失败'
  return entry.source
}

export function normalizeBlacklistUserID(id: string): string {
  if (!id.startsWith('<at')) return id
  const match = id.match(/id="(\d+)"/)
  return match ? match[1] : id
}
