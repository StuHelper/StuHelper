import assert from 'node:assert/strict'
import test from 'node:test'

import { errorMessage } from './error-message'

test('errorMessage normalizes unknown causes with fallback text', () => {
  assert.equal(errorMessage(undefined, '默认错误'), '默认错误')
  assert.equal(errorMessage(null, '默认错误'), '默认错误')
  assert.equal(errorMessage('  ', '默认错误'), '默认错误')
  assert.equal(errorMessage({ message: ' backend failed ' }, '默认错误'), 'backend failed')
  assert.equal(errorMessage({ error: new Error('remote failed') }, '默认错误'), 'remote failed')
  assert.equal(errorMessage({ reason: 'policy denied' }, '默认错误'), 'policy denied')
  assert.equal(errorMessage({ code: 'E_TEST' }, '默认错误'), '默认错误')
})
