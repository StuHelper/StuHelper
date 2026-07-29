import assert from 'node:assert/strict'
import { spawnSync } from 'node:child_process'
import test from 'node:test'

test('patched file-type parser terminates on a zero-sized ASF sub-header', () => {
  const script = `
    const FileType = require('file-type')
    const payload = Buffer.alloc(55)
    Buffer.from([0x30, 0x26, 0xB2, 0x75, 0x8E, 0x66, 0xCF, 0x11, 0xA6, 0xD9]).copy(payload)
    FileType.fromBuffer(payload)
      .then((result) => {
        if (!result || result.ext !== 'asf') process.exit(2)
      })
      .catch(() => process.exit(3))
  `
  const result = spawnSync(process.execPath, ['-e', script], {
    cwd: process.cwd(),
    encoding: 'utf8',
    timeout: 2_000,
  })

  assert.equal(result.error, undefined)
  assert.equal(result.status, 0, result.stderr)
})
