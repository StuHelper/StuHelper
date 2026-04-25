import assert from 'node:assert/strict'
import test from 'node:test'

import type {
  StuhelperConsoleEvent,
  StuhelperConsoleReport,
} from '../../src/console-types'
import { buildAuditRows, normalizeAuditFilterKind } from './model'

test('buildAuditRows merges events and reports into a single newest-first list', () => {
  const rows = buildAuditRows(
    [
      createEvent('evt_1', '2026-04-20T09:10:00.000Z'),
      createEvent('evt_2', '2026-04-20T09:40:00.000Z'),
    ],
    [
      createReport('rp_1', '2026-04-20T09:50:00.000Z'),
      createReport('rp_2', '2026-04-20T09:20:00.000Z'),
    ],
  )

  assert.deepEqual(
    rows.map((row) => ({
      id: row.id,
      kind: row.kind,
      memberId: row.memberId,
      target: row.target,
      level: row.level,
      detail: row.detail,
    })),
    [
      {
        id: 'rp_1',
        kind: 'report',
        memberId: 'rp_1-reporter',
        target: 'rp_1-target',
        level: 'high',
        detail: 'completed',
      },
      {
        id: 'evt_2',
        kind: 'event',
        memberId: 'evt_2-member',
        target: 'guild-1',
        level: 'medium',
        detail: 'keyword-hit',
      },
      {
        id: 'rp_2',
        kind: 'report',
        memberId: 'rp_2-reporter',
        target: 'rp_2-target',
        level: 'high',
        detail: 'completed',
      },
      {
        id: 'evt_1',
        kind: 'event',
        memberId: 'evt_1-member',
        target: 'guild-1',
        level: 'medium',
        detail: 'keyword-hit',
      },
    ],
  )
})

test('buildAuditRows applies keyword search across summary, member, target, level and detail', () => {
  const rows = buildAuditRows(
    [createEvent('evt_1', '2026-04-20T09:10:00.000Z')],
    [createReport('rp_1', '2026-04-20T09:50:00.000Z')],
    { query: 'rp_1-target' },
  )

  assert.deepEqual(rows.map((row) => row.id), ['rp_1'])
})

test('buildAuditRows filters by kind', () => {
  const rows = buildAuditRows(
    [createEvent('evt_1', '2026-04-20T09:10:00.000Z')],
    [createReport('rp_1', '2026-04-20T09:50:00.000Z')],
    { kind: 'event' },
  )

  assert.deepEqual(rows.map((row) => row.id), ['evt_1'])
})

test('normalizeAuditFilterKind falls back to all for unknown values', () => {
  assert.equal(normalizeAuditFilterKind(null), 'all')
  assert.equal(normalizeAuditFilterKind('report'), 'report')
  assert.equal(normalizeAuditFilterKind('unexpected'), 'all')
})

function createEvent(id: string, createdAt: string): StuhelperConsoleEvent {
  return {
    id,
    platform: 'mock',
    botSelfId: '514',
    guildId: 'guild-1',
    channelId: 'channel-1',
    memberId: `${id}-member`,
    type: 'keyword-hit',
    level: 'medium',
    summary: `事件摘要 ${id}`,
    payload: null,
    createdAt,
    updatedAt: createdAt,
  }
}

function createReport(id: string, createdAt: string): StuhelperConsoleReport {
  return {
    id,
    platform: 'mock',
    botSelfId: '514',
    guildId: 'guild-1',
    channelId: 'channel-1',
    reporterMemberId: `${id}-reporter`,
    targetMemberId: `${id}-target`,
    reason: `举报原因 ${id}`,
    aiStatus: 'completed',
    aiSeverity: 'high',
    aiSummary: `举报摘要 ${id}`,
    createdAt,
    updatedAt: createdAt,
  }
}
