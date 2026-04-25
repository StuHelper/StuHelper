import type { StuhelperConsoleData } from '../../src/console-types'
import { formatTimestamp } from '../formatters'
import type { DashboardChangeRow } from './model'

const RECENT_CHANGE_LIMIT = 6

interface SortableChangeRow extends DashboardChangeRow {
  sortTime: number
}

export function buildRecentChanges(data: StuhelperConsoleData | undefined): DashboardChangeRow[] {
  const rows = [
    ...(data?.keywordRules ?? []).map((rule) => ({
      id: rule.id,
      title: `关键词规则 · ${rule.id}`,
      meta: `${rule.guildId} · ${rule.pattern}`,
      kindLabel: '规则',
      time: formatTimestamp(rule.updatedAt),
      sortTime: toTime(rule.updatedAt),
    })),
    ...(data?.commandPolicies ?? []).map((policy) => ({
      id: policy.commandId,
      title: `命令权限 · ${policy.commandId}`,
      meta: `最低权限 ${policy.minAuthority}`,
      kindLabel: '命令',
      time: formatTimestamp(policy.updatedAt),
      sortTime: toTime(policy.updatedAt),
    })),
    ...(data?.memberRoles ?? []).map((role) => ({
      id: role.id,
      title: `成员角色 · ${role.memberId}`,
      meta: `${role.guildId} · ${role.roles.join(', ') || '无角色'}`,
      kindLabel: '角色',
      time: formatTimestamp(role.updatedAt),
      sortTime: toTime(role.updatedAt),
    })),
    ...(data?.guardTemplates ?? []).map((template) => ({
      id: template.id,
      title: `群模板 · ${template.name}`,
      meta: `${template.id} · ${template.enabled ? '启用' : '停用'}`,
      kindLabel: '模板',
      time: formatTimestamp(template.updatedAt),
      sortTime: toTime(template.updatedAt),
    })),
    ...(data?.guardBindings ?? []).map((binding) => ({
      id: binding.id,
      title: `群绑定 · ${binding.platform}/${binding.guildId}`,
      meta: `模板 ${binding.templateId}`,
      kindLabel: '绑定',
      time: formatTimestamp(binding.updatedAt),
      sortTime: toTime(binding.updatedAt),
    })),
  ] satisfies SortableChangeRow[]

  return rows
    .sort((left, right) => right.sortTime - left.sortTime)
    .slice(0, RECENT_CHANGE_LIMIT)
    .map(({ sortTime: _sortTime, ...row }) => row)
}

function toTime(value?: string | null) {
  if (!value) return 0
  const time = new Date(value).getTime()
  return Number.isNaN(time) ? 0 : time
}
