import assert from 'node:assert/strict'
import test from 'node:test'

import { buildMessageNodes } from './message-content-model'

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
