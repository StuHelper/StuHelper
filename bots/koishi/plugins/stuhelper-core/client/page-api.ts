import { send } from '@koishijs/client'

import type { EntityProfileQuery, ReviewWorkItem } from './page-types'
import type {
  AdmissionRuntimeAction,
  AdmissionRuntimePageData,
  AdmissionRuntimeSettingsPatch,
} from './models/admission-runtime'

export const consolePageApi = {
  dashboard() {
    return send('stuhelperGroupCenter/page/dashboard')
  },
  identity() {
    return send('stuhelperGroupCenter/page/identity')
  },
  review() {
    return send('stuhelperGroupCenter/page/review')
  },
  configGovernance() {
    return send('stuhelperGroupCenter/page/config-governance')
  },
  admissionRuntime(): Promise<AdmissionRuntimePageData> {
    return send('stuhelperGroupGuard/page/admission-runtime')
  },
  admissionAction(input: { recordId: string; action: AdmissionRuntimeAction }) {
    return send('stuhelperGroupGuard/action/admission-member', input)
  },
  saveAdmissionRuntimeSettings(input: AdmissionRuntimeSettingsPatch) {
    return send('stuhelperGroupGuard/action/save-admission-runtime-settings', input)
  },
  entityProfile(query: EntityProfileQuery) {
    return send('stuhelperGroupCenter/page/entity-profile', query)
  },
  reviewAction(input: { reviewId: string; action: 'execute' | 'reject'; note?: string }) {
    return send('stuhelperGroupCenter/action/review', input)
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
    return send('stuhelperGroupCenter/action/work-item', input)
  },
  saveCommandPolicy(input: { commandId: string; roles: string[]; minAuthority: number }) {
    return send('stuhelperGroupCenter/action/save-command-policy', input)
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
    return send('stuhelperGroupCenter/action/save-guard-template', input)
  },
  saveGuardBinding(input: {
    platform: string
    guildId: string
    templateId: string
    enabled: boolean
    note?: string | null
  }) {
    return send('stuhelperGroupCenter/action/save-guard-binding', input)
  },
}
