import assert from 'node:assert/strict'
import { mkdtemp, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import test from 'node:test'
import { createServer } from 'node:http'

import commands from '@koishijs/plugin-commands'
import sqlite from '@koishijs/plugin-database-sqlite'
import MockBot from '@koishijs/plugin-mock'

import {
  BindingRuntimeSettingsStore,
  DEFAULT_BINDING_RUNTIME_SETTINGS,
  type BindingRuntimeSettingsInput,
} from '@stuhelper/koishi-shared'

import bindingPlugin from './index.ts'
import { createKoishiTestRuntime } from '../../test-utils/runtime.ts'

test('绑定命令在私聊中消费绑定码并返回成功提示', async () => {
  const server = createServer((req, res) => {
    assert.equal(req.headers.authorization, 'Bearer test-token')
    assert.equal(req.method, 'POST')
    assert.equal(req.url, '/api/v1/bot/qq-binding/consume')

    let body = ''
    req.on('data', (chunk) => {
      body += chunk
    })
    req.on('end', () => {
      assert.match(body, /"code":"ABCD1234"/)
      assert.match(body, /"qqID":"10001"/)
      res.setHeader('content-type', 'application/json')
      res.end(JSON.stringify({
        success: true,
        data: {
          binding: {
            userID: 42,
            qqID: '10001',
            boundAt: '2026-04-19T00:00:00Z',
            createdAt: '2026-04-19T00:00:00Z',
            updatedAt: '2026-04-19T00:00:00Z',
          },
          verificationState: {
            qqID: '10001',
            userID: 42,
            boundAt: '2026-04-19T00:00:00Z',
            verificationState: 'verified',
            studentVerified: true,
          },
        },
      }))
    })
  })

  await new Promise<void>((resolve) => server.listen(0, resolve))
  const address = server.address()
  assert.ok(address && typeof address === 'object')

  const runtime = createKoishiTestRuntime()
  const { root } = runtime
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-koishi-binding-'))

  runtime.register(sqlite, { path: join(tempDir, 'koishi.db') })
  runtime.register(commands)
  runtime.register(MockBot, { selfId: '514' })
  runtime.register(bindingPlugin, {
    platform: {
      baseUrl: `http://127.0.0.1:${address.port}`,
      serviceToken: 'test-token',
    },
  })

  try {
    await root.start()
    const client = root.mock.client('10001')
    await client.shouldReply('绑定 ABCD1234', '绑定成功，当前账号已完成学生认证，加入受控群时会自动放行。')
  } finally {
    runtime.dispose()
    await new Promise<void>((resolve, reject) => server.close((error) => error ? reject(error) : resolve()))
    await rm(tempDir, { recursive: true, force: true })
  }
})

test('绑定命令在群聊中提示用户改用私聊', async () => {
  const runtime = createKoishiTestRuntime()
  const { root } = runtime
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-koishi-binding-'))

  runtime.register(sqlite, { path: join(tempDir, 'koishi.db') })
  runtime.register(commands)
  runtime.register(MockBot, { selfId: '514' })
  runtime.register(bindingPlugin, {
    platform: {
      baseUrl: 'http://127.0.0.1:8080',
      serviceToken: 'test-token',
    },
  })

  try {
    await root.start()
    const client = root.mock.client('10001', 'group-1')
    await client.shouldReply('绑定 ABCD1234', '请在私聊中发送绑定命令。')
  } finally {
    runtime.dispose()
    await rm(tempDir, { recursive: true, force: true })
  }
})

test('绑定命令使用 WebUI runtime settings 里的命令字和自定义提示文案', async () => {
  const runtime = createKoishiTestRuntime()
  const { root } = runtime
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-koishi-binding-'))

  runtime.register(sqlite, { path: join(tempDir, 'koishi.db') })
  runtime.register(commands)
  runtime.register(MockBot, { selfId: '514' })
  runtime.register(bindingPlugin, {
    platform: {
      baseUrl: 'http://127.0.0.1:8080',
      serviceToken: 'test-token',
    },
  })

  try {
    await root.start()
    await saveBindingRuntimeSettings(root, {
      command: '绑定账号',
      messages: {
        directOnly: '自定义：请私聊机器人绑定。',
        missingCode: '自定义：请输入 {command} 后面的绑定码。',
      },
    })
    const groupClient = root.mock.client('10001', 'group-1')
    await groupClient.shouldReply('绑定账号 ABCD1234', '自定义：请私聊机器人绑定。')

    const directClient = root.mock.client('10001')
    await directClient.shouldReply('绑定账号', '自定义：请输入 绑定账号 后面的绑定码。')
  } finally {
    runtime.dispose()
    await rm(tempDir, { recursive: true, force: true })
  }
})

async function saveBindingRuntimeSettings(
  root: ConstructorParameters<typeof BindingRuntimeSettingsStore>[0],
  overrides: BindingRuntimeSettingsInput,
) {
  await new BindingRuntimeSettingsStore(root, DEFAULT_BINDING_RUNTIME_SETTINGS).saveSettings(overrides)
}
