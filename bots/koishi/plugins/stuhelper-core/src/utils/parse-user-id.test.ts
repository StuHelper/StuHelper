import assert from 'node:assert/strict'
import test from 'node:test'

import { parseUserId } from './index'

test('parseUserId accepts Koishi user args and OneBot numeric ids', () => {
  assert.equal(parseUserId('onebot:10001'), '10001')
  assert.equal(parseUserId('10002'), '10002')
  assert.equal(parseUserId('@10003'), '10003')
  assert.equal(parseUserId('<at id="10004"/>'), '10004')
  assert.equal(parseUserId({ id: 'onebot:10005' }), '10005')
  assert.equal(parseUserId({ userId: '@10006' }), '10006')
  assert.equal(parseUserId({ uid: '<at id="10007"/>' }), '10007')
  assert.equal(parseUserId({}), '')
  assert.equal(parseUserId(10008), '')
})
