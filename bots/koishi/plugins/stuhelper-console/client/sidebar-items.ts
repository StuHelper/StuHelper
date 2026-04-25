import type { StuhelperConsoleData } from '../src/console-types'
import { CONSOLE_SECTIONS, type ConsoleSectionId } from './sections'

export interface SidebarItem {
  id: ConsoleSectionId
  label: string
  count: number | null
}

export function buildSidebarItems(data: StuhelperConsoleData | undefined): SidebarItem[] {
  const pendingReviewCount = data?.pendingReviews.length ?? 0
  const pendingMemberCount = data?.pendingMembers.length ?? 0
  const policyCount =
    (data?.keywordRules.length ?? 0) +
    (data?.commandPolicies.length ?? 0) +
    (data?.memberRoles.length ?? 0) +
    (data?.guardTemplates.length ?? 0) +
    (data?.guardBindings.length ?? 0)
  const auditCount =
    (data?.recentEvents.length ?? 0) +
    (data?.recentReports.length ?? 0)

  return CONSOLE_SECTIONS.map((section) => {
    if (section.id === 'enforcement') {
      return { ...section, count: pendingReviewCount }
    }

    if (section.id === 'identity') {
      return { ...section, count: pendingMemberCount }
    }

    if (section.id === 'policy') {
      return { ...section, count: policyCount }
    }

    if (section.id === 'audit') {
      return { ...section, count: auditCount }
    }

    return { ...section, count: null }
  })
}
