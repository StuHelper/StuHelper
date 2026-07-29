import assert from 'node:assert/strict'
import test from 'node:test'

import {
  formatBotOperationError,
  isOneBotSetGroupBanPermissionError,
} from './bot-errors'

test('detects onebot set_group_ban permission retcode errors', () => {
  const error = new Error(
    'Error with request set_group_ban, args: {"group_id":743762161,"user_id":285466036,"duration":0}, retcode: 1200',
  )

  assert.equal(isOneBotSetGroupBanPermissionError(error), true)
  assert.match(formatBotOperationError(error), /机器人缺少群管理员权限/)
})

test('does not treat unrelated onebot errors as set_group_ban permission errors', () => {
  assert.equal(isOneBotSetGroupBanPermissionError(new Error('Error with request send_msg, retcode: 1200')), false)
  assert.equal(isOneBotSetGroupBanPermissionError(new Error('Error with request set_group_ban, retcode: 100')), false)
  assert.equal(isOneBotSetGroupBanPermissionError(new Error('set_group_ban xretcode: 1200')), false)
  assert.equal(isOneBotSetGroupBanPermissionError(new Error('set_group_ban retcode: not-a-number')), false)
})

test('parses supported onebot retcode separators in linear time', () => {
  for (const separator of [':', '=', ',']) {
    assert.equal(
      isOneBotSetGroupBanPermissionError(new Error(`set_group_ban retcode ${separator} 1200`)),
      true,
    )
  }

  const longPrefix = 'x'.repeat(20_000)
  assert.equal(
    isOneBotSetGroupBanPermissionError(new Error(`${longPrefix} set_group_ban retcode: 1200`)),
    false,
  )
})
