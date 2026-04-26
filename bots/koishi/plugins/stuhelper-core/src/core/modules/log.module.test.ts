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
