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

test('errorMessage maps network failures to actionable platform guidance', () => {
  const guidance = 'StuHelper 平台服务暂时不可用，请检查后端地址、网络或服务令牌配置。'
  assert.equal(errorMessage(new TypeError('fetch failed'), '默认错误'), guidance)
  assert.equal(errorMessage('Failed to fetch', '默认错误'), guidance)
  assert.equal(
    errorMessage({ error: new Error('connect ECONNREFUSED 127.0.0.1:8080') }, '默认错误'),
    guidance,
  )
})
