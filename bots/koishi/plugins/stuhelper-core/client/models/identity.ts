import type { IdentityMemberSnapshot, IdentityPageData } from '../page-types'

import { formatTimestamp } from './formatters'

export interface IdentityModelOptions {
  guildId?: string | null
  itemId?: string | null
  keyword?: string
}

export interface IdentityDetailCard {
  label: string
  value: string
}

export function buildIdentityModel(
  data: IdentityPageData,
  options: IdentityModelOptions,
) {
  const selectedGuildId = options.guildId || ''
  const keyword = (options.keyword || '').trim().toLowerCase()
  const filteredMembers = data.members.filter((member) => {
    if (selectedGuildId && member.guildId !== selectedGuildId) {
      return false
    }
    if (!keyword) {
      return true
    }
    return [member.memberId, member.memberName]
      .filter(Boolean)
      .some((value) => String(value).toLowerCase().includes(keyword))
  })

  const selectedMember = filteredMembers.find((item) => item.id === options.itemId) ?? filteredMembers[0] ?? null

  return {
    selectedGuildId,
    filteredMembers,
    selectedMember,
    detailCards: buildIdentityDetailCards(selectedMember),
  }
}

function buildIdentityDetailCards(member: IdentityMemberSnapshot | null): IdentityDetailCard[] {
  if (!member) {
    return []
  }

  return [
    { label: '认证状态', value: member.profile?.verificationState || member.verificationState },
    { label: '绑定记录', value: member.profile?.boundAt ? formatTimestamp(member.profile.boundAt) : '暂无绑定时间' },
    { label: '限制截止', value: formatTimestamp(member.deadlineAt) },
    { label: '最近错误', value: member.lastError || '无' },
  ]
}
