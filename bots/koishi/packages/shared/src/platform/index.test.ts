import assert from 'node:assert/strict'
import test from 'node:test'

import { createPlatformClient } from './index.ts'

test('platform admission client sends expected paths and payloads', async (t) => {
  const originalFetch = globalThis.fetch
  const calls: CapturedRequest[] = []
  globalThis.fetch = async (input, init) => {
    const url = new URL(String(input))
    calls.push({
      path: url.pathname + url.search,
      method: init?.method || 'GET',
      authorization: new Headers(init?.headers).get('authorization') || '',
      body: typeof init?.body === 'string' ? JSON.parse(init.body) : null,
    })
    return jsonResponse(responseDataForPath(url.pathname))
  }
  t.after(() => {
    globalThis.fetch = originalFetch
  })

  const client = createPlatformClient({
    baseUrl: 'https://api.example.test',
    serviceToken: 'service-token',
  })

  await client.createAdmissionSession({
    platform: 'qq',
    guildID: 'guild-1',
    channelID: 'channel-1',
    qqID: '10001',
    qqNickname: 'Alice',
    botSelfID: '514',
  })
  await client.recordJoinRequestEvent({
    platform: 'qq',
    guildID: 'guild-1',
    qqID: '10001',
    requestID: 'request-1',
    success: false,
    error: 'permission denied',
    rawEvent: { comment: '我是新生' },
  })
  await client.listPendingAdmissionActions({ platform: 'qq', botSelfID: '514', limit: 50 })
  await client.recordAdmissionEvent('session-1', {
    action: 'release',
    success: true,
    messageID: 'message-1',
  })
  await client.listPendingFreshmanForwards()
  await client.markFreshmanForwarded('app-1')
  await client.viewFreshmanApplication('app-1', {
    operatorQQID: '90001',
    guildID: 'mgmt-1',
    channelID: 'channel-1',
    rawCommand: '新生审核查看 app-1',
  })
  await client.reviewFreshmanApplication('app-1', {
    action: 'approve',
    operatorQQID: '90001',
    guildID: 'mgmt-1',
    channelID: 'channel-1',
    rawCommand: '新生审核通过 app-1 +30d',
    expiresInDays: 30,
  })
  await client.releaseAdmissionBlacklist('10001', {
    platform: 'qq',
    targetGuildID: 'guild-1',
    operatorQQID: '90001',
    guildID: 'mgmt-1',
    channelID: 'channel-1',
    rawCommand: '新生黑名单解除 10001',
  })
  await client.getMemberBlacklistAccess({
    platform: 'qq',
    subjectType: 'qq_user',
    subjectID: '10001',
    guildID: 'guild-1',
  })
  await client.listMemberBlacklist({ pageSize: 50 })
  await client.createMemberBlacklist({
    platform: 'qq',
    subjectType: 'qq_user',
    subjectID: '10001',
    scopeType: 'guild',
    guildID: 'guild-1',
    source: 'kick_blacklist',
    reasonCode: 'manual_kick_blacklist',
    reasonText: 'kick command',
    createdFrom: 'qq_command',
    operatorID: '90001',
    metadata: { rawCommand: 'kick 10001 -b' },
  })
  await client.releaseMemberBlacklist('blk-1', {
    releaseReasonCode: 'manual_pardon',
    releaseReason: 'test',
    operatorID: '90001',
  })
  await client.releaseMemberBlacklistBySubject({
    platform: 'qq',
    subjectType: 'qq_user',
    subjectID: '10001',
    scopeType: 'guild',
    guildID: 'guild-1',
    releaseReasonCode: 'manual_pardon',
    releaseReason: 'test',
    operatorID: '90001',
  })

  assert.deepEqual(calls.map((call) => [call.method, call.path]), [
    ['POST', '/api/v1/bot/admission/sessions'],
    ['POST', '/api/v1/bot/admission/join-requests/events'],
    ['GET', '/api/v1/bot/admission/sessions/pending?platform=qq&botSelfID=514&limit=50'],
    ['POST', '/api/v1/bot/admission/sessions/session-1/events'],
    ['GET', '/api/v1/bot/admission/freshman/applications/pending-forward'],
    ['POST', '/api/v1/bot/admission/freshman/applications/app-1/forwarded'],
    ['POST', '/api/v1/bot/admission/freshman/applications/app-1/view'],
    ['POST', '/api/v1/bot/admission/freshman/applications/app-1/review'],
    ['POST', '/api/v1/bot/admission/blacklist/10001/release'],
    ['GET', '/api/v1/bot/member-blacklist/access?platform=qq&subjectType=qq_user&subjectID=10001&guildID=guild-1'],
    ['GET', '/api/v1/bot/member-blacklist?pageSize=50'],
    ['POST', '/api/v1/bot/member-blacklist'],
    ['POST', '/api/v1/bot/member-blacklist/blk-1/release'],
    ['POST', '/api/v1/bot/member-blacklist/release-by-subject'],
  ])
  assert.ok(calls.every((call) => call.authorization === 'Bearer service-token'))
  assert.equal(calls[0].body.qqNickname, 'Alice')
  assert.equal(calls[1].body.rawEvent.comment, '我是新生')
  assert.equal(calls[3].body.messageID, 'message-1')
  assert.equal(calls[6].body.operatorQQID, '90001')
  assert.equal(calls[7].body.expiresInDays, 30)
  assert.equal(calls[8].body.rawCommand, '新生黑名单解除 10001')
  assert.equal(calls[11].body.source, 'kick_blacklist')
  assert.equal(calls[12].body.releaseReasonCode, 'manual_pardon')
})

test('platform client rejects missing service token at construction', () => {
  assert.throws(() => createPlatformClient({
    baseUrl: 'https://api.example.test',
    serviceToken: '',
  }), /platform service token is required/)
})

test('platform client accepts empty success responses for void requests', async (t) => {
  const originalFetch = globalThis.fetch
  globalThis.fetch = async () => new Response(null, { status: 204 })
  t.after(() => {
    globalThis.fetch = originalFetch
  })

  const client = createPlatformClient({
    baseUrl: 'https://api.example.test',
    serviceToken: 'service-token',
  })

  await client.getHealth()
  await client.recordAdmissionEvent('session-1', {
    action: 'release',
    success: true,
  })
})

interface CapturedRequest {
  readonly path: string
  readonly method: string
  readonly authorization: string
  readonly body: Record<string, unknown> | null
}

function jsonResponse(data: unknown) {
  return new Response(JSON.stringify({ success: true, data }), {
    status: 200,
    headers: { 'content-type': 'application/json' },
  })
}

function responseDataForPath(path: string) {
  if (path.endsWith('/sessions')) {
    return {
      session: admissionSession('session-1'),
      token: 'token-1',
      authURL: 'https://auth.stuhelper.com/admission/a/token-1?qq=10001',
    }
  }
  if (path.endsWith('/sessions/pending')) {
    return [{ sessionID: 'session-1', action: 'release' }]
  }
  if (path.endsWith('/pending-forward')) {
    return [{
      application: freshmanApplication('app-1'),
      materialURL: 'https://cdn.example.test/material.png',
      managementGuildIDs: ['mgmt-1'],
    }]
  }
  if (path.endsWith('/view') || path.endsWith('/review')) {
    return freshmanApplication('app-1')
  }
  return { message: 'ok' }
}

function admissionSession(id: string) {
  return {
    id,
    platform: 'qq',
    guildID: 'guild-1',
    channelID: 'channel-1',
    qqID: '10001',
    status: 'joined_muted',
    tokenExpiresAt: '2026-05-03T13:00:00Z',
    linkWaitDeadlineAt: '2026-05-03T13:00:00Z',
    submissionWaitDeadlineAt: '2026-05-04T12:00:00Z',
    initialMuteUntil: '2026-06-02T12:00:00Z',
    projectionPending: false,
  }
}

function freshmanApplication(id: string) {
  return {
    id,
    userID: 42,
    schoolID: 1,
    status: 'pending',
    applicantNameMasked: 'A***',
    materialType: 'admission_notice',
    createdAt: '2026-05-03T12:00:00Z',
  }
}
