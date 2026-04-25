import test from 'node:test'
import assert from 'node:assert/strict'

import { prependQuoteElement, serializeChatElements } from './chat-element-serializer'

test('serializeChatElements escapes text payloads instead of emitting raw HTML', () => {
  const serialized = serializeChatElements([
    {
      type: 'text',
      attrs: {
        content: '<img src=x onerror=alert(1)>',
      },
    },
  ])

  assert.equal(serialized, '&lt;img src=x onerror=alert(1)&gt;')
})

test('serializeChatElements escapes special characters inside generated attributes', () => {
  const serialized = serializeChatElements([
    {
      type: 'at',
      attrs: {
        id: '42',
        name: 'foo" onclick="alert(1)',
      },
    },
    {
      type: 'img',
      attrs: {
        src: `https://example.com/x'"><svg onload=alert(1)>`,
      },
    },
  ])

  assert.match(serialized, /name="foo&quot; onclick=&quot;alert\(1\)"/)
  assert.match(serialized, /src="https:\/\/example\.com\/x&#39;&quot;&gt;&lt;svg onload=alert\(1\)&gt;"/)
})

test('prependQuoteElement prepends a structured quote element to the existing element list', () => {
  const elements = prependQuoteElement(
    [
      {
        type: 'text',
        attrs: {
          content: 'hello',
        },
      },
    ],
    {
      messageId: 'msg-1',
      user: 'alice',
      content: 'quoted',
    },
  )

  assert.equal(elements[0]?.type, 'quote')
  assert.deepEqual(elements[0]?.attrs, {
    id: 'msg-1',
    user: 'alice',
    content: 'quoted',
  })
  assert.equal(elements[1]?.type, 'text')
})
