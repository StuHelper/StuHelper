import { send } from '@koishijs/client'

import type {
  ConfigGovernancePageData,
  DashboardPageData,
  EntityProfile,
  EntityProfileQuery,
  IdentityPageData,
  ReviewPageData,
  ReviewWorkItem,
} from './page-types'

async function callPage<T>(event: string, params?: Record<string, unknown>) {
  return (send as unknown as (name: string, payload?: Record<string, unknown>) => Promise<T>)(event, params)
}

export const consolePageApi = {
  dashboard() {
    return callPage<DashboardPageData>('stuhelperGroupCenter/page/dashboard')
  },
  identity() {
    return callPage<IdentityPageData>('stuhelperGroupCenter/page/identity')
  },
  review() {
    return callPage<ReviewPageData>('stuhelperGroupCenter/page/review')
  },
  configGovernance() {
    return callPage<ConfigGovernancePageData>('stuhelperGroupCenter/page/config-governance')
  },
  entityProfile(query: EntityProfileQuery) {
    return callPage<EntityProfile>('stuhelperGroupCenter/page/entity-profile', query as unknown as Record<string, unknown>)
  },
  reviewAction(input: { reviewId: string; action: 'execute' | 'reject'; note?: string }) {
    return callPage<string>('stuhelperGroupCenter/action/review', input)
  },
  workItemAction(input: {
    kind: ReviewWorkItem['kind']
    itemId: string
    action:
      | 'execute'
      | 'reject'
      | 'approve'
      | 'deny'
      | 'defer'
      | 'dismiss'
      | 'escalate'
      | 'create-review'
    note?: string
  }) {
    return callPage<string>('stuhelperGroupCenter/action/work-item', input)
  },
  saveCommandPolicy(input: { commandId: string; roles: string[]; minAuthority: number }) {
    return callPage<string>('stuhelperGroupCenter/action/save-command-policy', input)
  },
  saveGuardTemplate(input: {
    id: string
    name: string
    muteDurationSeconds: number
    kickAfterMinutes: number
    reminderTemplate: string
    exemptUsers: string[]
    enabled: boolean
  }) {
    return callPage<string>('stuhelperGroupCenter/action/save-guard-template', input)
  },
  saveGuardBinding(input: {
    platform: string
    guildId: string
    templateId: string
    enabled: boolean
    note?: string | null
  }) {
    return callPage<string>('stuhelperGroupCenter/action/save-guard-binding', input)
  },
}
