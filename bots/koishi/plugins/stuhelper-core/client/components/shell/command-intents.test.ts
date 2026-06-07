import test from 'node:test'
import assert from 'node:assert/strict'

import {
  commandRoutePath,
  searchCommandIntents,
} from './command-intents'

test('searchCommandIntents returns non-auth StuHelper command entries', () => {
  const results = searchCommandIntents('群审状态')

  assert.equal(results[0]?.name, '群审状态')
  assert.equal(results[0]?.path, '/commands/群审状态')
  assert.equal(results[0]?.kind, 'command')
})

test('searchCommandIntents matches admission link command keywords', () => {
  const names = searchCommandIntents('认证链接').map((result) => result.name)

  assert.deepEqual(names, ['重发认证链接', '重新生成认证链接'])
})

test('searchCommandIntents matches public command aliases without auth commands', () => {
  const reportNames = searchCommandIntents('report').map((result) => result.name)

  assert.deepEqual(reportNames, ['举报'])
  assert.deepEqual(searchCommandIntents('绑定'), [])
  assert.deepEqual(searchCommandIntents('gauth'), [])
})

test('commandRoutePath follows Koishi command route dot replacement', () => {
  assert.equal(commandRoutePath('cmdlogs.check'), '/commands/cmdlogs/check')
})
