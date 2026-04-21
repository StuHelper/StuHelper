import { computed, reactive } from 'vue'

import type { StuhelperConsoleData } from '../src/console-types'

export type InspectorKind =
  | 'member'
  | 'review'
  | 'event'
  | 'report'
  | 'template'
  | 'binding'
  | 'rule'

export interface InspectorState {
  open: boolean
  kind: InspectorKind | null
  id: string
  reviewCandidateIds: string[]
}

export function createInspectorState() {
  return reactive<InspectorState>({
    open: false,
    kind: null,
    id: '',
    reviewCandidateIds: [],
  })
}

export function openInspector(
  inspector: InspectorState,
  kind: InspectorKind,
  id: string,
  reviewCandidateIds: readonly string[] = [],
) {
  inspector.open = true
  inspector.kind = kind
  inspector.id = id
  inspector.reviewCandidateIds = kind === 'review' ? [...reviewCandidateIds] : []
}

export function closeInspector(inspector: InspectorState) {
  inspector.open = false
  inspector.kind = null
  inspector.id = ''
  inspector.reviewCandidateIds = []
}

export function useInspectorPayload(
  data: Readonly<{ value: StuhelperConsoleData | undefined }>,
  inspector: InspectorState,
) {
  return computed(() => {
    if (!inspector.kind || !inspector.id) return null
    const currentData = data.value
    if (!currentData) return null

    switch (inspector.kind) {
      case 'member':
        return currentData.pendingMembers.find((item) => item.id === inspector.id) ?? null
      case 'review':
        return currentData.pendingReviews.find((item) => item.id === inspector.id) ?? null
      case 'event':
        return currentData.recentEvents.find((item) => item.id === inspector.id) ?? null
      case 'report':
        return currentData.recentReports.find((item) => item.id === inspector.id) ?? null
      case 'template':
        return currentData.guardTemplates.find((item) => item.id === inspector.id) ?? null
      case 'binding':
        return currentData.guardBindings.find((item) => item.id === inspector.id) ?? null
      case 'rule':
        return currentData.keywordRules.find((item) => item.id === inspector.id) ?? null
    }
  })
}
