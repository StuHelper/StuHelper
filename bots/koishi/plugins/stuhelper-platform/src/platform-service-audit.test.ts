import assert from 'node:assert/strict'
import test from 'node:test'

import * as entry from './index'
import { createModuleRegistry } from './module-registry'
import type { AuditEventRecord } from './platform-models'
import { StuhelperPlatformService } from './platform-service'

const AUDIT_TIME = new Date('2026-04-24T10:00:00.000Z')

class AuditStore {
  readonly auditLimits: Array<number | undefined> = []
  readonly auditEvents: AuditEventRecord[] = [auditRow()]

  async listAuditEvents(limit?: number): Promise<readonly AuditEventRecord[]> {
    this.auditLimits.push(limit)
    return limit === undefined ? this.auditEvents : this.auditEvents.slice(0, limit)
  }
}

test('listAuditEvents returns serialized immutable audit snapshots', async () => {
  const store = new AuditStore()
  const service = new StuhelperPlatformService({
    registry: createModuleRegistry([]),
    store: store as never,
    runtime: {} as never,
  })

  const [event] = await service.listAuditEvents(1)

  assert.deepEqual(store.auditLimits, [1])
  assert.equal(event.createdAt, AUDIT_TIME.toISOString())
  assert.equal(event.updatedAt, AUDIT_TIME.toISOString())
  assert.equal(Object.isFrozen(event), true)
  assert.equal(Object.isFrozen(event.payload), true)
})

test('entry exports platform service', () => {
  assert.equal(entry.StuhelperPlatformService, StuhelperPlatformService)
})

function auditRow(): AuditEventRecord {
  return {
    id: 'audit-1',
    actor: 'admin',
    moduleId: 'warn',
    action: 'module.state.set',
    summary: 'changed',
    payload: { enabled: false },
    createdAt: AUDIT_TIME,
    updatedAt: AUDIT_TIME,
  }
}
