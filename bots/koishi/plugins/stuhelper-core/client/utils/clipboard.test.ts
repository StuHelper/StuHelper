import assert from 'node:assert/strict'
import test from 'node:test'

import { copyTextToClipboard, copyTextWithTextarea } from './clipboard'

test('copyTextToClipboard uses the async clipboard writer first', async () => {
  const writes: string[] = []

  const copied = await copyTextToClipboard('role-id', {
    clipboard: {
      writeText: (text) => writes.push(text),
    },
    execCopy: () => {
      throw new Error('fallback should not run')
    },
  })

  assert.equal(copied, true)
  assert.deepEqual(writes, ['role-id'])
})

test('copyTextToClipboard falls back to textarea copy when clipboard fails', async () => {
  const target = createTarget()

  const copied = await copyTextToClipboard('guild-id', {
    clipboard: {
      writeText: () => Promise.reject(new Error('clipboard denied')),
    },
    createTextarea: () => target,
    execCopy: () => true,
  })

  assert.equal(copied, true)
  assert.equal(target.value, 'guild-id')
  assert.equal(target.selected, true)
  assert.equal(target.removed, true)
})

test('copyTextWithTextarea returns false and still removes fallback target on failure', () => {
  const target = createTarget()

  const copied = copyTextWithTextarea('message', {
    createTextarea: () => target,
    execCopy: () => {
      throw new Error('copy failed')
    },
  })

  assert.equal(copied, false)
  assert.equal(target.value, 'message')
  assert.equal(target.selected, true)
  assert.equal(target.removed, true)
})

function createTarget() {
  return {
    value: '',
    selected: false,
    removed: false,
    select() {
      this.selected = true
    },
    remove() {
      this.removed = true
    },
  }
}
