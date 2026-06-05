import assert from 'node:assert/strict'
import test from 'node:test'

import { buildMessageNodes, buildMessagePlainText } from './message-content-model'

const baseMessage = {
  id: 'm1',
  content: '',
  timestamp: 1,
  userId: 'u1',
  username: 'u1',
  channelId: '1001',
  platform: 'onebot',
}

test('buildMessageNodes drops images with unsafe protocols', () => {
  const nodes = buildMessageNodes({
    ...baseMessage,
    elements: [
      { type: 'img', attrs: { src: 'javascript:alert(1)' } },
      { type: 'img', attrs: { src: 'file:///etc/passwd' } },
      { type: 'img', attrs: { src: 'http://example.com/a.png' } },
    ],
  } as any, [], [])

  assert.deepEqual(nodes, [{
    kind: 'image',
    key: 'm1-2',
    src: 'http://example.com/a.png',
    file: undefined,
    openUrl: 'http://example.com/a.png',
  }])
})

test('buildMessageNodes allows safe image data urls only', () => {
  const safeDataUrl = 'data:image/png;base64,AAAA'
  const nodes = buildMessageNodes({
    ...baseMessage,
    elements: [
      { type: 'img', attrs: { src: 'data:text/html;base64,PHNjcmlwdA==' } },
      { type: 'img', attrs: { src: safeDataUrl } },
    ],
  } as any, [], [])

  assert.equal(nodes.length, 1)
  assert.equal(nodes[0]?.kind, 'image')
  assert.equal((nodes[0] as any).src, safeDataUrl)
  assert.equal((nodes[0] as any).openUrl, null)
})

test('buildMessagePlainText mirrors rendered structured message nodes', () => {
  const text = buildMessagePlainText({
    ...baseMessage,
    elements: [
      { type: 'quote', attrs: { id: 'quoted' } },
      { type: 'at', attrs: { id: 'u2' } },
      { type: 'text', attrs: { content: ' 请看' } },
      { type: 'img', attrs: { src: 'https://example.com/a.png', file: 'a.png' } },
    ],
  } as any, [{
    ...baseMessage,
    id: 'quoted',
    username: 'Alice',
    content: '上一条',
  } as any], [{
    id: 'u2',
    name: 'Bob',
    avatar: '',
    roles: [],
  } as any])

  assert.equal(text, '引用 @Alice: 上一条@Bob 请看[图片:a.png]')
})

test('buildMessagePlainText preserves raw fallback content without DOM parsing', () => {
  const text = buildMessagePlainText({
    ...baseMessage,
    content: '<img src=x onerror=alert(1)> &amp; hello',
    elements: undefined,
  } as any, [], [])

  assert.equal(text, '<img src=x onerror=alert(1)> &amp; hello')
})
