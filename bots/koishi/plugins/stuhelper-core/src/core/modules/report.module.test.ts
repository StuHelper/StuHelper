import assert from 'node:assert/strict'
import test from 'node:test'

import { ReportModule } from './report.module'

test('ReportModule appends command logs when the store currently holds array data', async () => {
  let savedLogs: unknown
  const module = new ReportModule(
    {} as any,
    {
      commandLogs: {
        getAll: () => [
          {
            id: 'existing',
            timestamp: '2026-06-01T00:00:00.000Z',
            guildId: '1001',
            userId: 'u1',
            platform: 'onebot',
            command: 'existing',
            args: [],
            options: {},
            success: true,
            executionTime: 0,
            isPrivate: false,
          },
        ],
        setAll: (value: unknown) => {
          savedLogs = value
        },
      },
    } as any,
    {} as any,
  )

  await module.logCommand({
    session: {
      platform: 'onebot',
      guildId: '1001',
      channelId: '2001',
      userId: 'u2',
      username: 'reporter',
      messageId: 'message-1',
    } as any,
    command: 'report',
    target: 'message-1',
    details: 'queued report',
  })

  assert.ok(savedLogs && typeof savedLogs === 'object')
  const logs = (savedLogs as { logs: Array<Record<string, unknown>> }).logs
  assert.equal(logs.length, 2)
  assert.equal(logs[1].command, 'report')
  assert.equal(logs[1].result, 'queued report')
  assert.deepEqual(logs[1].args, ['message-1'])
})
