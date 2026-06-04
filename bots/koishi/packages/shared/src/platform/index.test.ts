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
    botSelfID: '514',
  })
  await client.getAdmissionSessionByMember({ platform: 'qq', guildID: 'guild-1', qqID: '10001' })
  await client.resendAdmissionSessionLink({ platform: 'qq', guildID: 'guild-1', qqID: '10001' })
  await client.regenerateAdmissionSessionLink({
    platform: 'qq',
    guildID: 'guild-1',
    channelID: 'channel-1',
    qqID: '10001',
    botSelfID: '514',
  })
  await client.skipAdmissionSessionForMember({
    platform: 'qq',
    guildID: 'guild-1',
    qqID: '10001',
    operatorQQID: '90001',
  })
  await client.resetAdmissionFailureCount({
    platform: 'qq',
    guildID: 'guild-1',
    qqID: '10001',
    operatorQQID: '90001',
  })
  await client.resolveJoinRequestDecision({
    platform: 'qq',
    guildID: 'guild-1',
    qqID: '10001',
    requestID: 'request-1',
    rawEvent: { comment: '我是新生' },
  })
  await client.recordJoinRequestEvent({
    platform: 'qq',
    guildID: 'guild-1',
    qqID: '10001',
    requestID: 'request-1',
    decision: 'approve',
    success: false,
    error: 'permission denied',
    rawEvent: { comment: '我是新生' },
  })
  await client.listPendingAdmissionActions({ platform: 'qq', botSelfID: '514', limit: 50 })
  await client.getMemberBlacklistAccess({
    platform: 'qq',
    subjectType: 'qq_user',
    subjectID: '10001',
    guildID: 'guild-1',
  })
  await client.listMemberBlacklist({ platform: 'qq', status: 'active', pageSize: 20 })
  await client.createMemberBlacklist({
    platform: 'qq',
    subjectType: 'qq_user',
    subjectID: '10001',
    scopeType: 'guild',
    guildID: 'guild-1',
    source: 'manual_admin',
    reasonCode: 'manual_blacklist',
    reasonText: 'manual command',
    createdFrom: 'qq_command',
    metadata: { operatorQQID: '90001' },
  })
  await client.releaseMemberBlacklist('entry-1', {
    releaseReasonCode: 'release_only',
    operatorQQID: '90001',
  })
  await client.releaseMemberBlacklistBySubject({
    platform: 'qq',
    subjectType: 'qq_user',
    subjectID: '10001',
    scopeType: 'guild',
    guildID: 'guild-1',
    releaseReasonCode: 'manual_pardon',
    operatorQQID: '90001',
  })
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
  assert.deepEqual(calls.map((call) => [call.method, call.path]), [
    ['POST', '/api/v1/bot/admission/sessions'],
    ['GET', '/api/v1/bot/admission/sessions/member?platform=qq&guildID=guild-1&qqID=10001'],
    ['POST', '/api/v1/bot/admission/sessions/member/resend'],
    ['POST', '/api/v1/bot/admission/sessions/member/regenerate'],
    ['POST', '/api/v1/bot/admission/sessions/member/skip'],
    ['POST', '/api/v1/bot/admission/failures/reset'],
    ['POST', '/api/v1/bot/admission/join-requests/decision'],
    ['POST', '/api/v1/bot/admission/join-requests/events'],
    ['GET', '/api/v1/bot/admission/sessions/pending?platform=qq&botSelfID=514&limit=50'],
    ['GET', '/api/v1/bot/member-blacklist/access?platform=qq&subjectType=qq_user&subjectID=10001&guildID=guild-1'],
    ['GET', '/api/v1/bot/member-blacklist?platform=qq&status=active&pageSize=20'],
    ['POST', '/api/v1/bot/member-blacklist'],
    ['POST', '/api/v1/bot/member-blacklist/entry-1/release'],
    ['POST', '/api/v1/bot/member-blacklist/release-by-subject'],
    ['POST', '/api/v1/bot/admission/sessions/session-1/events'],
    ['GET', '/api/v1/bot/admission/freshman/applications/pending-forward'],
    ['POST', '/api/v1/bot/admission/freshman/applications/app-1/forwarded'],
    ['POST', '/api/v1/bot/admission/freshman/applications/app-1/view'],
    ['POST', '/api/v1/bot/admission/freshman/applications/app-1/review'],
  ])
  assert.ok(calls.every((call) => call.authorization === 'Bearer service-token'))
  assert.equal('qqNickname' in calls[0].body, false)
  assert.equal(calls[2].body.qqID, '10001')
  assert.equal(calls[3].body.botSelfID, '514')
  assert.equal(calls[4].body.operatorQQID, '90001')
  assert.equal(calls[5].body.operatorQQID, '90001')
  assert.equal(calls[6].body.rawEvent.comment, '我是新生')
  assert.equal(calls[7].body.decision, 'approve')
  assert.equal(calls[11].body.metadata.operatorQQID, '90001')
  assert.equal(calls[11].body.createdFrom, 'qq_command')
  assert.equal(calls[13].body.releaseReasonCode, 'manual_pardon')
  assert.equal(calls[14].body.messageID, 'message-1')
  assert.equal(calls[17].body.operatorQQID, '90001')
  assert.equal(calls[18].body.expiresInDays, 30)
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

test('platform admission client requires bot identity for pending actions', async () => {
  const client = createPlatformClient({
    baseUrl: 'https://api.example.test',
    serviceToken: 'service-token',
  })

  await assert.rejects(
    client.listPendingAdmissionActions({ platform: 'qq', botSelfID: '' }),
    /platform and botSelfID are required/,
  )
})

test('platform admission client requires member query identity', async () => {
  const client = createPlatformClient({
    baseUrl: 'https://api.example.test',
    serviceToken: 'service-token',
  })

  await assert.rejects(
    client.getAdmissionSessionByMember({ platform: 'qq', guildID: '', qqID: '10001' }),
    /platform, guildID and qqID are required/,
  )

  await assert.rejects(
    client.resetAdmissionFailureCount({ platform: 'qq', guildID: 'guild-1', qqID: '10001', operatorQQID: '' }),
    /operatorQQID is required/,
  )
})

test('platform client resolves Koishi env placeholders from process env', async (t) => {
  const originalFetch = globalThis.fetch
  const previousBaseURL = process.env.STUHELPER_PLATFORM_BASE_URL
  const previousServiceToken = process.env.STUHELPER_PLATFORM_SERVICE_TOKEN
  let capturedURL = ''
  let capturedAuthorization = ''

  process.env.STUHELPER_PLATFORM_BASE_URL = 'https://env-api.example.test'
  process.env.STUHELPER_PLATFORM_SERVICE_TOKEN = 'env-service-token'
  globalThis.fetch = async (input, init) => {
    capturedURL = String(input)
    capturedAuthorization = new Headers(init?.headers).get('authorization') || ''
    return new Response(null, { status: 204 })
  }
  t.after(() => {
    globalThis.fetch = originalFetch
    restoreEnv('STUHELPER_PLATFORM_BASE_URL', previousBaseURL)
    restoreEnv('STUHELPER_PLATFORM_SERVICE_TOKEN', previousServiceToken)
  })

  const client = createPlatformClient({
    baseUrl: '${{ env.STUHELPER_PLATFORM_BASE_URL }}',
    serviceToken: '${{ env.STUHELPER_PLATFORM_SERVICE_TOKEN }}',
  })

  await client.getHealth()

  assert.equal(capturedURL, 'https://env-api.example.test/health/live')
  assert.equal(capturedAuthorization, 'Bearer env-service-token')
})

test('platform client rejects missing platform config at construction', () => {
  const previousBaseURL = process.env.STUHELPER_PLATFORM_BASE_URL
  const previousServiceToken = process.env.STUHELPER_PLATFORM_SERVICE_TOKEN
  delete process.env.STUHELPER_PLATFORM_BASE_URL
  delete process.env.STUHELPER_PLATFORM_SERVICE_TOKEN
  try {
    assert.throws(() => createPlatformClient({
      baseUrl: 'https://api.example.test',
      serviceToken: '',
    }), /platform service token is required/)

    assert.throws(() => createPlatformClient({
      baseUrl: '',
      serviceToken: 'service-token',
    }), /platform baseUrl is required/)

    assert.throws(() => createPlatformClient({
      baseUrl: '${{ env.STUHELPER_PLATFORM_BASE_URL }}',
      serviceToken: 'service-token',
    }), /platform baseUrl is required/)
  } finally {
    restoreEnv('STUHELPER_PLATFORM_BASE_URL', previousBaseURL)
    restoreEnv('STUHELPER_PLATFORM_SERVICE_TOKEN', previousServiceToken)
  }
})

test('platform client rejects invalid platform baseUrl at construction', () => {
  assert.throws(() => createPlatformClient({
    baseUrl: 'not-a-url',
    serviceToken: 'service-token',
  }), /platform baseUrl must be an absolute URL/)
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
  if (path.endsWith('/sessions/member')) {
    return admissionSession('session-1')
  }
  if (path.endsWith('/sessions/member/resend')) {
    return {
      ...admissionSession('session-1'),
      authURL: 'https://join.stuhelper.com/verify/token-1',
    }
  }
  if (path.endsWith('/sessions/member/regenerate')) {
    return {
      session: admissionSession('session-2'),
      token: 'token-2',
      authURL: 'https://join.stuhelper.com/verify/token-2',
    }
  }
  if (path.endsWith('/sessions/member/skip')) {
    return { ...admissionSession('session-3'), status: 'cancelled' }
  }
  if (path.endsWith('/failures/reset')) {
    return { platform: 'qq', guildID: 'guild-1', qqID: '10001', previousFailureCount: 2 }
  }
  if (path.endsWith('/join-requests/decision')) {
    return {
      decision: 'approve',
      reason: 'unverified_auto_approve',
      verificationState: 'unverified',
      autoApproveVerifiedJoin: true,
      autoApproveUnverifiedJoin: true,
      policyID: 'policy-1',
    }
  }
  if (path.endsWith('/sessions')) {
    return {
      session: admissionSession('session-1'),
      token: 'token-1',
      authURL: 'https://join.stuhelper.com/verify/token-1',
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
  if (path.endsWith('/member-blacklist/access')) {
    return { canJoin: true, decision: 'allowed' }
  }
  if (path.endsWith('/member-blacklist')) {
    return { list: [memberBlacklistEntry('entry-1')], total: 1 }
  }
  if (path.includes('/member-blacklist/')) {
    return memberBlacklistEntry('entry-1')
  }
  if (path.endsWith('/view') || path.endsWith('/review')) {
    return freshmanApplication('app-1')
  }
  return { message: 'ok' }
}

function restoreEnv(name: string, value: string | undefined) {
  if (typeof value === 'undefined') {
    delete process.env[name]
    return
  }
  process.env[name] = value
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
    schoolID: 4111010006,
    status: 'pending',
    applicantNameMasked: 'A***',
    materialType: 'admission_notice',
    createdAt: '2026-05-03T12:00:00Z',
  }
}

function memberBlacklistEntry(id: string) {
  return {
    id,
    platform: 'qq',
    subjectType: 'qq_user',
    subjectID: '10001',
    scopeType: 'guild',
    guildID: 'guild-1',
    source: 'manual_admin',
    reasonCode: 'manual_blacklist',
    reasonText: 'manual command',
    metadata: {},
    createdByType: 'qq_operator',
    createdByID: '90001',
    createdFrom: 'qq_command',
    createdAt: '2026-05-03T12:00:00Z',
    updatedAt: '2026-05-03T12:00:00Z',
  }
}
