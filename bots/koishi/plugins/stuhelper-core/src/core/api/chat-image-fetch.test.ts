import assert from 'node:assert/strict'
import test from 'node:test'

import { createChatImageAccessRegistry } from './chat-image-fetch'
import type { ConsoleGuildScope } from './console-guild-scope'

const SCOPE_ALL: ConsoleGuildScope = { kind: 'all' }

function imageElement(file: string, url: string) {
  return { type: 'img', attrs: { file, src: url } }
}

test('chat image access registry rejects entries after the TTL expires', () => {
  let now = 1_000_000
  const registry = createChatImageAccessRegistry({ ttlMs: 1_000, now: () => now })
  registry.remember([imageElement('f1.png', 'https://gchat.qpic.cn/img1')], 'guild-1')

  assert.deepEqual(
    registry.assertAllowed({ file: 'f1.png', url: 'https://gchat.qpic.cn/img1' }, SCOPE_ALL),
    { file: 'f1.png', url: 'https://gchat.qpic.cn/img1' },
  )

  now += 1_001
  assert.throws(
    () => registry.assertAllowed({ file: 'f1.png', url: 'https://gchat.qpic.cn/img1' }, SCOPE_ALL),
    /not attached to a delivered chat message/,
  )
})

test('chat image access registry evicts oldest entries beyond capacity', () => {
  const registry = createChatImageAccessRegistry({ ttlMs: 60_000, capacity: 2, now: () => 1_000_000 })
  registry.remember([imageElement('f1.png', 'https://gchat.qpic.cn/img1')], 'guild-1')
  registry.remember([imageElement('f2.png', 'https://gchat.qpic.cn/img2')], 'guild-1')
  registry.remember([imageElement('f3.png', 'https://gchat.qpic.cn/img3')], 'guild-1')

  assert.throws(
    () => registry.assertAllowed({ file: 'f1.png', url: 'https://gchat.qpic.cn/img1' }, SCOPE_ALL),
    /not attached to a delivered chat message/,
  )
  assert.deepEqual(
    registry.assertAllowed({ file: 'f2.png', url: 'https://gchat.qpic.cn/img2' }, SCOPE_ALL),
    { file: 'f2.png', url: 'https://gchat.qpic.cn/img2' },
  )
  assert.deepEqual(
    registry.assertAllowed({ file: 'f3.png', url: 'https://gchat.qpic.cn/img3' }, SCOPE_ALL),
    { file: 'f3.png', url: 'https://gchat.qpic.cn/img3' },
  )
})

test('chat image access registry prefers dropping expired entries when over capacity', () => {
  let now = 1_000_000
  const registry = createChatImageAccessRegistry({ ttlMs: 1_000, capacity: 2, now: () => now })
  registry.remember([imageElement('f1.png', 'https://gchat.qpic.cn/img1')], 'guild-1')

  now += 2_000
  registry.remember([imageElement('f2.png', 'https://gchat.qpic.cn/img2')], 'guild-1')
  registry.remember([imageElement('f3.png', 'https://gchat.qpic.cn/img3')], 'guild-1')

  // f1 已过期被清理，容量内的活跃条目 f2/f3 都保留
  assert.deepEqual(
    registry.assertAllowed({ file: 'f2.png', url: 'https://gchat.qpic.cn/img2' }, SCOPE_ALL),
    { file: 'f2.png', url: 'https://gchat.qpic.cn/img2' },
  )
  assert.deepEqual(
    registry.assertAllowed({ file: 'f3.png', url: 'https://gchat.qpic.cn/img3' }, SCOPE_ALL),
    { file: 'f3.png', url: 'https://gchat.qpic.cn/img3' },
  )
})
