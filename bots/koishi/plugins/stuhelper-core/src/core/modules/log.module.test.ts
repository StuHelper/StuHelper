import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import test from 'node:test'

import { LogModule } from './log.module'

test('LogModule throws when command log JSON is corrupt', () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'stuhelper-log-module-'))
  fs.writeFileSync(path.join(dir, 'command_logs.json'), '{ invalid json', 'utf8')
  const module = createLogModule(dir)

  assert.throws(() => module.readCommandLogs(), SyntaxError)
})

test('LogModule reads legacy object command log data', () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'stuhelper-log-module-'))
  fs.writeFileSync(path.join(dir, 'command_logs.json'), JSON.stringify({
    logs: [
      {
        timestamp: 1_787_803_200_000,
        guildId: '1001',
        userId: 'u1',
        command: 'report',
        target: 'message',
        details: 'created report',
      },
    ],
  }), 'utf8')
  const module = createLogModule(dir)

  const logs = module.readCommandLogs()

  assert.equal(logs.length, 1)
  assert.equal(logs[0].command, 'report')
  assert.equal(logs[0].result, 'created report')
  assert.equal(logs[0].guildId, '1001')
})

test('LogModule redacts sensitive legacy command log data on read', () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'stuhelper-log-module-'))
  fs.writeFileSync(path.join(dir, 'command_logs.json'), JSON.stringify({
    logs: [
      {
        timestamp: 1_787_803_200_000,
        guildId: '1001',
        userId: 'u1',
        command: 'secret',
        args: ['--token', 'tok_legacy_secret'],
        options: {
          headers: {
            authorization: 'Bearer legacy_auth_secret',
          },
          apiKey: 'sk-legacy-secret',
        },
        details: 'Cookie: sid=legacy_cookie_secret',
      },
    ],
  }), 'utf8')
  const module = createLogModule(dir)

  const payload = JSON.stringify(module.readCommandLogs())

  assert.doesNotMatch(payload, /tok_legacy_secret/)
  assert.doesNotMatch(payload, /legacy_auth_secret/)
  assert.doesNotMatch(payload, /sk-legacy-secret/)
  assert.doesNotMatch(payload, /legacy_cookie_secret/)
  assert.match(payload, /\[REDACTED\]/)
})

test('LogModule preserves array command log data', () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'stuhelper-log-module-'))
  fs.writeFileSync(path.join(dir, 'command_logs.json'), JSON.stringify([
    {
      id: 'log-1',
      timestamp: '2026-06-01T00:00:00.000Z',
      guildId: '1001',
      userId: 'u1',
      platform: 'onebot',
      command: 'warn',
      args: ['u2'],
      options: { silent: true },
      success: true,
      executionTime: 12,
      isPrivate: false,
    },
  ]), 'utf8')
  const module = createLogModule(dir)

  const logs = module.readCommandLogs()

  assert.equal(logs.length, 1)
  assert.equal(logs[0].id, 'log-1')
  assert.deepEqual(logs[0].args, ['u2'])
  assert.deepEqual(logs[0].options, { silent: true })
})

test('LogModule throws when command log save fails', (t) => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'stuhelper-log-module-'))
  const module = createLogModule(dir)
  const writeError = new Error('command log write failed')

  t.mock.method(fs, 'writeFileSync', () => {
    throw writeError
  })

  assert.throws(() => module.saveCommandLogs([]), /command log write failed/)
})

function createLogModule(dataPath: string): LogModule {
  return new LogModule(
    { stuhelperGroupCenter: { pluginConfig: {} } } as any,
    { dataPath } as any,
  )
}
