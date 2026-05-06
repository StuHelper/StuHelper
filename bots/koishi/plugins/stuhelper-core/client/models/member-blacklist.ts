import type { MemberBlacklistEntry, MemberBlacklistScopeType } from '../types'
import { formatTimestamp } from './formatters'
import type { QueueTableColumn, QueueTableRow } from '../components/primitives/QueueTable.vue'

export interface BlacklistDraft {
  readonly platform: string
  readonly userId: string
  readonly scope: MemberBlacklistScopeType
  readonly guildId: string
}

export type MemberBlacklistDisplayStatus = 'active' | 'expired' | 'released'

export const MEMBER_BLACKLIST_COLUMNS: QueueTableColumn[] = [
  { key: 'status', label: '状态', width: '96' },
  { key: 'user', label: '用户 ID' },
  { key: 'scope', label: '范围', width: '160' },
  { key: 'source', label: '来源', width: '120' },
  { key: 'reason', label: '原因' },
  { key: 'createdFrom', label: '创建入口', width: '140' },
  { key: 'createdBy', label: '创建人', width: '200' },
  { key: 'createdAt', label: '创建时间', width: '180' },
  { key: 'expiresAt', label: '过期时间', width: '180' },
  { key: 'releasedAt', label: '解除时间', width: '180' },
]

const SOURCE_LABELS: Record<string, string> = {
  admission_failure: '认证失败',
  kick_blacklist: '踢出拉黑',
  manual_admin: '手动',
  migration_admission_failure: '迁移·认证失败',
  migration_legacy_koishi: '迁移·Koishi 旧库',
  moderation_action: '审核处置',
}

const CREATED_FROM_LABELS: Record<string, string> = {
  admin_console: 'Admin 后台',
  admission_worker: 'Admission Worker',
  koishi_console: 'Koishi 控制台',
  migration_script: '迁移脚本',
  moderation_review: '审核流程',
  qq_command: 'QQ 命令',
}

export function canSubmitBlacklistDraft(draft: BlacklistDraft): boolean {
  if (!draft.platform.trim() || !draft.userId.trim()) return false
  return draft.scope === 'global' || Boolean(draft.guildId.trim())
}

export function entryDisplayStatus(entry: MemberBlacklistEntry): MemberBlacklistDisplayStatus {
  if (entry.releasedAt) return 'released'
  if (entry.expiresAt && Date.parse(entry.expiresAt) <= Date.now()) return 'expired'
  return 'active'
}

export function formatBlacklistStatus(status: MemberBlacklistDisplayStatus): string {
  if (status === 'active') return '生效中'
  if (status === 'expired') return '已过期'
  return '已解除'
}

export function statusTone(status: MemberBlacklistDisplayStatus): 'danger' | 'info' | 'success' {
  if (status === 'active') return 'danger'
  if (status === 'expired') return 'info'
  return 'success'
}

export function toBlacklistRows(entries: readonly MemberBlacklistEntry[]): QueueTableRow[] {
  return entries.map((entry) => {
    const status = entryDisplayStatus(entry)
    return {
      id: entry.id,
      cells: {
        status: { text: formatBlacklistStatus(status), tone: statusTone(status) },
        user: { text: entry.subjectID },
        scope: { text: formatBlacklistScope(entry), tone: entry.scopeType === 'global' ? 'danger' : 'info' },
        source: formatBlacklistSource(entry),
        reason: entry.reasonText || '未提供',
        createdFrom: formatCreatedFrom(entry),
        createdBy: { text: formatCreatedBy(entry), mono: true },
        createdAt: { text: formatTimestamp(Date.parse(entry.createdAt)), mono: true },
        expiresAt: { text: entry.expiresAt ? formatTimestamp(Date.parse(entry.expiresAt)) : '永久', mono: Boolean(entry.expiresAt) },
        releasedAt: { text: entry.releasedAt ? formatTimestamp(Date.parse(entry.releasedAt)) : '—', mono: Boolean(entry.releasedAt) },
      },
      actions: status === 'active' ? [{ key: 'remove', label: '解除', tone: 'danger' }] : [],
    }
  })
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
  return SOURCE_LABELS[entry.source] ?? entry.source
}

export function formatCreatedFrom(entry: MemberBlacklistEntry): string {
  return CREATED_FROM_LABELS[entry.createdFrom] ?? entry.createdFrom
}

export function formatCreatedBy(entry: MemberBlacklistEntry): string {
  return `${entry.createdByType} · ${entry.createdByID}`
}

export function normalizeBlacklistUserID(id: string): string {
  if (!id.startsWith('<at')) return id
  const match = id.match(/id="(\d+)"/)
  return match ? match[1] : id
}
