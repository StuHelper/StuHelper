import assert from 'node:assert/strict'
import test from 'node:test'
import type { Bot } from 'koishi'

import {
  getOneBotInternalMethod,
  requireOneBotInternalMethod,
} from './onebot-internal'

test('getOneBotInternalMethod returns null outside onebot platform', async () => {
  const bot = {
    platform: 'qq',
    internal: {
      async getImage() {
        return { base64: 'AAAA' }
      },
    },
  } as unknown as Bot

  assert.equal(getOneBotInternalMethod(bot, 'getImage'), null)
})

test('getOneBotInternalMethod binds OneBot internal methods to their owner', async () => {
  const bot = {
    platform: 'onebot',
    internal: {
      marker: 'bound',
      async getImage(this: { marker: string }, file: string) {
        return `${this.marker}:${file}`
      },
    },
  } as unknown as Bot

  const getImage = getOneBotInternalMethod(bot, 'getImage')

  assert.equal(await getImage?.('file-id'), 'bound:file-id')
})

test('requireOneBotInternalMethod reports unsupported actions explicitly', () => {
  const bot = {
    platform: 'onebot',
    internal: {},
  } as unknown as Bot

  assert.throws(
    () => requireOneBotInternalMethod(bot, 'setGroupAdmin', 'set_group_admin'),
    /当前适配器不支持 OneBot set_group_admin/,
  )
})
