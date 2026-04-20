import { CONSOLE_SECTIONS, type ConsoleSectionId } from './sections'

export interface SidebarCounts {
  pendingReviews: number
  pendingMembers: number
  policyCount: number
  auditCount: number
}

export interface SidebarItem {
  id: ConsoleSectionId
  label: string
  count: number | null
}

export function buildSidebarItems(counts: SidebarCounts): SidebarItem[] {
  return CONSOLE_SECTIONS.map((section) => {
    if (section.id === 'enforcement') {
      return { ...section, count: counts.pendingReviews }
    }

    if (section.id === 'identity') {
      return { ...section, count: counts.pendingMembers }
    }

    if (section.id === 'policy') {
      return { ...section, count: counts.policyCount }
    }

    if (section.id === 'audit') {
      return { ...section, count: counts.auditCount }
    }

    return { ...section, count: null }
  })
}
