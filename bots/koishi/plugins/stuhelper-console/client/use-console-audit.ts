import { computed, ref, watch } from 'vue'

import { buildAuditRows, normalizeAuditFilterKind, type AuditFilterKind, type AuditRow } from './audit/model'
import type { ConsoleSearchState } from './navigation'
import type { InspectorKind } from './use-console-inspector'
import type {
  StuhelperConsoleEvent,
  StuhelperConsoleReport,
} from '../src/console-types'

interface RefLike<T> {
  value: T
}

interface UseConsoleAuditOptions {
  routeState: RefLike<ConsoleSearchState>
  setRouteState: (
    next: Partial<ConsoleSearchState>,
    options?: { history?: 'push' | 'replace' },
  ) => void
  recentEvents: RefLike<readonly StuhelperConsoleEvent[]>
  recentReports: RefLike<readonly StuhelperConsoleReport[]>
  inspector: { kind: InspectorKind | null }
  closeInspector: () => void
  openInspector: (kind: InspectorKind, id: string, reviewCandidateIds?: readonly string[]) => void
  pushNotice: (kind: 'success' | 'error', message: string) => void
}

export function useConsoleAudit(options: UseConsoleAuditOptions) {
  const rawAuditQuery = ref('')
  const selectedAuditId = computed(() =>
    options.routeState.value.section === 'audit' ? options.routeState.value.id : '',
  )
  const auditKind = computed<AuditFilterKind>({
    get: () =>
      options.routeState.value.section === 'audit'
        ? normalizeAuditFilterKind(options.routeState.value.queue)
        : 'all',
    set: (kind) => {
      updateAuditRoute(normalizeAuditFilterKind(kind))
    },
  })
  const auditQuery = computed({
    get: () => rawAuditQuery.value,
    set: (query: string) => {
      rawAuditQuery.value = query
      syncAuditSelection()
    },
  })
  const auditRows = computed(() =>
    buildAuditRows(options.recentEvents.value, options.recentReports.value, {
      kind: auditKind.value,
      query: rawAuditQuery.value,
    }),
  )

  watch(auditRows, (rows) => {
    if (options.routeState.value.section !== 'audit') return
    if (!options.routeState.value.id && rows[0]) {
      options.setRouteState({
        section: 'audit',
        queue: getAuditQueue(auditKind.value),
        id: rows[0].id,
        source: 'direct',
      }, { history: 'replace' })
      return
    }
    if (!options.routeState.value.id || rows.some((row) => row.id === options.routeState.value.id)) return
    options.closeInspector()
    options.setRouteState({
      section: 'audit',
      queue: getAuditQueue(auditKind.value),
      id: rows[0]?.id ?? '',
      source: 'direct',
    }, { history: 'replace' })
  })

  function inspectAuditRow(row: AuditRow) {
    options.setRouteState({
      section: 'audit',
      queue: getAuditQueue(auditKind.value),
      id: row.id,
      source: 'direct',
    })

    if (row.kind === 'event') {
      const event = options.recentEvents.value.find((item) => item.id === row.id)
      if (event) {
        options.openInspector('event', event.id)
        return
      }
    }

    if (row.kind === 'report') {
      const report = options.recentReports.value.find((item) => item.id === row.id)
      if (report) {
        options.openInspector('report', report.id)
        return
      }
    }

    options.closeInspector()
    options.setRouteState({ id: '' }, { history: 'replace' })
    options.pushNotice('error', '记录已不存在，请刷新后重试。')
  }

  function getAuditQueue(kind: AuditFilterKind) {
    return kind === 'all' ? null : kind
  }

  function resolveAuditSelection(kind: AuditFilterKind, currentId = options.routeState.value.id) {
    const rows = buildAuditRows(options.recentEvents.value, options.recentReports.value, {
      kind,
      query: rawAuditQuery.value,
    })
    if (currentId && rows.some((row) => row.id === currentId)) return currentId
    return rows[0]?.id ?? ''
  }

  function updateAuditRoute(kind: AuditFilterKind) {
    const nextId = resolveAuditSelection(kind)
    options.setRouteState({
      section: 'audit',
      queue: getAuditQueue(kind),
      id: nextId,
      source: 'direct',
    })
    if (options.inspector.kind === 'event' || options.inspector.kind === 'report') {
      options.closeInspector()
    }
  }

  function syncAuditSelection() {
    if (options.routeState.value.section !== 'audit') return
    const nextId = resolveAuditSelection(auditKind.value)
    if (nextId === options.routeState.value.id) return
    options.closeInspector()
    options.setRouteState({
      section: 'audit',
      queue: getAuditQueue(auditKind.value),
      id: nextId,
      source: 'direct',
    }, { history: 'replace' })
  }

  return {
    selectedAuditId,
    auditKind,
    auditQuery,
    auditRows,
    inspectAuditRow,
  }
}
