import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { mkdtemp, rm } from 'node:fs/promises'
import { createServer, type IncomingMessage, type ServerResponse } from 'node:http'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import test from 'node:test'

import commands from '@koishijs/plugin-commands'
import sqlite from '@koishijs/plugin-database-sqlite'
import MockBot from '@koishijs/plugin-mock'
import type { Context } from 'koishi'

import {
  AdminRuntimeSettingsStore,
  DEFAULT_ADMIN_RUNTIME_SETTINGS,
  type StuhelperAdminMessageConfig,
} from '@stuhelper/koishi-shared'

import adminPlugin from './index.ts'
import { createKoishiTestRuntime } from '../../test-utils/runtime.ts'

test('新生审核命令不对已验证 session 做非空断言', () => {
  const source = readFileSync(new URL('./admission-review-commands.ts', import.meta.url), 'utf8')

  assert.doesNotMatch(source, /session!/)
})

test('新生审核命令会把操作者 QQ、群号、频道和原始命令发给后端', async () => {
  const requests: CapturedCommandRequest[] = []
  const server = createServer((req, res) => respondCommandRequest(req, res, requests))
  await new Promise<void>((resolve) => server.listen(0, resolve))
  const address = server.address()
  assert.ok(address && typeof address === 'object')

  const runtime = createKoishiTestRuntime()
  const { root } = runtime
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-koishi-admin-'))
  runtime.register(sqlite, { path: join(tempDir, 'koishi.db') })
  runtime.register(commands)
  runtime.register(MockBot, { selfId: '514' })
  runtime.register(adminPlugin, createAdminPluginConfig(address.port))

  try {
    await root.start()
    await root.mock.initUser('20020', 4)
    await root.mock.initChannel('mgmt-1')
    const client = root.mock.client('20020', 'mgmt-1')

    await client.shouldReply('新生审核查看 A123', /申请 A123/)
    await client.shouldReply('新生审核通过 A123', '已通过新生认证申请 A123。')
    await client.shouldReply('新生审核通过 A123 +30d', '已通过新生认证申请 A123，临时身份 30 天后过期。')
    await client.shouldReply('新生审核驳回 A123 材料不清晰', '已驳回新生认证申请 A123。')
    await client.shouldReply('新生黑名单解除 123456 guild-1', '已解除 123456 的群 guild-1 的入群认证黑名单。')

    assert.deepEqual(requests.map((item) => [item.method, item.path]), [
      ['POST', '/api/v1/bot/admission/freshman/applications/A123/view'],
      ['POST', '/api/v1/bot/admission/freshman/applications/A123/review'],
      ['POST', '/api/v1/bot/admission/freshman/applications/A123/review'],
      ['POST', '/api/v1/bot/admission/freshman/applications/A123/review'],
      ['POST', '/api/v1/bot/member-blacklist/release-by-subject'],
    ])
    assert.ok(requests.every((item) => item.authorization === 'Bearer test-token'))
    assert.deepEqual(requests[0].body, commandBody('20020', 'mgmt-1', '新生审核查看 A123'))
    assert.deepEqual(requests[1].body, { ...commandBody('20020', 'mgmt-1', '新生审核通过 A123'), action: 'approve' })
    assert.deepEqual(requests[2].body, {
      ...commandBody('20020', 'mgmt-1', '新生审核通过 A123 +30d'),
      action: 'approve',
      expiresInDays: 30,
    })
    assert.deepEqual(requests[3].body, {
      ...commandBody('20020', 'mgmt-1', '新生审核驳回 A123 材料不清晰'),
      action: 'reject',
      reason: '材料不清晰',
    })
    assert.deepEqual(requests[4].body, {
      platform: 'mock',
      subjectType: 'qq_user',
      subjectID: '123456',
      scopeType: 'guild',
      guildID: 'guild-1',
      releaseReasonCode: 'manual_pardon',
      releaseReason: 'freshman review command release',
      operatorQQID: '20020',
    })
  } finally {
    runtime.dispose()
    await closeServer(server)
    await rm(tempDir, { recursive: true, force: true })
  }
})

test('新生审核命令使用 WebUI runtime settings 里的自定义提示文案', async () => {
  const requests: CapturedCommandRequest[] = []
  const server = createServer((req, res) => respondCommandRequest(req, res, requests))
  await new Promise<void>((resolve) => server.listen(0, resolve))
  const address = server.address()
  assert.ok(address && typeof address === 'object')

  const runtime = createKoishiTestRuntime()
  const { root } = runtime
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-koishi-admin-'))
  runtime.register(sqlite, { path: join(tempDir, 'koishi.db') })
  runtime.register(commands)
  runtime.register(MockBot, { selfId: '514' })
  runtime.register(adminPlugin, createAdminPluginConfig(address.port))

  try {
    await root.start()
    await saveAdminRuntimeMessages(root, {
      freshmanApproveSuccess: '自定义通过：{applicationID}',
      freshmanApplicationSummary: '自定义申请 {applicationID} / {applicantName} / {departmentOrMajor}',
    })
    await root.mock.initUser('20020', 4)
    await root.mock.initChannel('mgmt-1')
    const client = root.mock.client('20020', 'mgmt-1')

    await client.shouldReply('新生审核查看 A123', '自定义申请 A123 / 张* / 计算机科学与技术')
    await client.shouldReply('新生审核通过 A123', '自定义通过：A123')
  } finally {
    runtime.dispose()
    await closeServer(server)
    await rm(tempDir, { recursive: true, force: true })
  }
})

test('新生审核命令会映射后端操作者授权错误并拒绝非法延长期限', async () => {
  const server = createServer((req, res) => respondForbiddenCommand(req, res))
  await new Promise<void>((resolve) => server.listen(0, resolve))
  const address = server.address()
  assert.ok(address && typeof address === 'object')

  const runtime = createKoishiTestRuntime()
  const { root } = runtime
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-koishi-admin-'))
  runtime.register(sqlite, { path: join(tempDir, 'koishi.db') })
  runtime.register(commands)
  runtime.register(MockBot, { selfId: '514' })
  runtime.register(adminPlugin, createAdminPluginConfig(address.port))

  try {
    await root.start()
    await root.mock.initUser('20021', 4)
    await root.mock.initChannel('mgmt-1')
    const client = root.mock.client('20021', 'mgmt-1')

    await client.shouldReply('新生审核查看 UNBOUND', '你的 QQ 未绑定 StuHelper 管理员账号，请先完成管理员 QQ 绑定。')
    await client.shouldReply('新生审核查看 FORBIDDEN', '你的 StuHelper 账号没有新生审核权限。')
    await client.shouldReply('新生审核查看 WRONG_GUILD', '当前群不在新生审核管理群白名单内。')
    await client.shouldReply('新生审核通过 A123 +0d', '审批延长天数格式应为 +30d，且必须是正整数天数。')
  } finally {
    runtime.dispose()
    await closeServer(server)
    await rm(tempDir, { recursive: true, force: true })
  }
})

interface CapturedCommandRequest {
  readonly method: string
  readonly path: string
  readonly authorization: string
  readonly body: Record<string, unknown>
}

function createAdminPluginConfig(port: number) {
  return {
    platform: { baseUrl: `http://127.0.0.1:${port}`, serviceToken: 'test-token' },
  }
}

async function saveAdminRuntimeMessages(
  root: Pick<Context, 'database'>,
  messages: Partial<StuhelperAdminMessageConfig>,
) {
  await new AdminRuntimeSettingsStore(
    root,
    DEFAULT_ADMIN_RUNTIME_SETTINGS,
  ).saveSettings({
    messages: {
      ...messages,
    },
  })
}

function commandBody(operatorQQID: string, guildID: string, rawCommand: string) {
  return { operatorQQID, guildID, channelID: guildID, rawCommand }
}

async function respondCommandRequest(
  req: IncomingMessage,
  res: ServerResponse,
  requests: CapturedCommandRequest[],
) {
  requests.push({
    method: req.method || '',
    path: req.url || '',
    authorization: req.headers.authorization || '',
    body: await readJSONBody(req),
  })
  res.setHeader('content-type', 'application/json')
  res.end(JSON.stringify({ success: true, data: freshmanApplication() }))
}

async function respondForbiddenCommand(req: IncomingMessage, res: ServerResponse) {
  const id = (req.url || '').split('/').at(-2) || ''
  const code = forbiddenCodeByApplicationID(id)
  await readJSONBody(req)
  res.statusCode = 403
  res.setHeader('content-type', 'application/json')
  res.end(JSON.stringify({ success: false, error: { code, message: code } }))
}

function forbiddenCodeByApplicationID(id: string) {
  if (id === 'UNBOUND') return 'admission.operator_qq_unbound'
  if (id === 'FORBIDDEN') return 'admission.operator_forbidden'
  return 'admission.management_guild_forbidden'
}

function readJSONBody(req: IncomingMessage) {
  const chunks: Buffer[] = []
  return new Promise<Record<string, unknown>>((resolve) => {
    req.on('data', (chunk: Buffer) => chunks.push(chunk))
    req.on('end', () => resolve(JSON.parse(Buffer.concat(chunks).toString() || '{}')))
  })
}

function freshmanApplication() {
  return {
    id: 'A123',
    userID: '42',
    schoolID: 4111010006,
    status: 'pending',
    applicantNameMasked: '张*',
    departmentOrMajor: '计算机科学与技术',
    materialType: 'admission_notice',
    provisionalExpiresAt: '2026-10-01T12:00:00+08:00',
    createdAt: '2026-05-03T12:00:00+08:00',
  }
}

function closeServer(server: ReturnType<typeof createServer>) {
  return new Promise<void>((resolve, reject) => {
    server.close((error) => error ? reject(error) : resolve())
  })
}
