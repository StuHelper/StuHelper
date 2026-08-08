import assert from 'node:assert/strict'
import type { IncomingMessage, ServerResponse } from 'node:http'

export function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

export async function waitFor(
  check: () => boolean | Promise<boolean>,
  timeoutMs = 1000,
  intervalMs = 20,
) {
  const deadline = Date.now() + timeoutMs
  while (Date.now() < deadline) {
    if (await check()) return
    await sleep(intervalMs)
  }
  throw new Error('waitFor timed out')
}

export function respondAdmissionSession(input: {
  readonly req: IncomingMessage
  readonly res: ServerResponse
  readonly qqID: string
  readonly guildID: string
}) {
  const { req, res, qqID, guildID } = input
  const url = new URL(req.url || '/', 'http://127.0.0.1')
  if (req.method === 'GET' && url.pathname === '/api/v1/bot/admission/sessions/member') {
    assert.equal(req.headers.authorization, 'Bearer test-token')
    assert.equal(url.searchParams.get('guildID'), guildID)
    assert.equal(url.searchParams.get('qqID'), qqID)
    res.setHeader('content-type', 'application/json')
    res.end(JSON.stringify({ success: true, data: admissionSessionData(qqID, guildID) }))
    return true
  }
  if (req.method !== 'POST' || req.url !== '/api/v1/bot/admission/sessions') return false
  assert.equal(req.headers.authorization, 'Bearer test-token')
  res.setHeader('content-type', 'application/json')
  res.end(JSON.stringify({
    success: true,
      data: {
        token: `token-${qqID}`,
        authURL: `https://join.stuhelper.com/verify/token-${qqID}`,
        session: admissionSessionData(qqID, guildID),
    },
  }))
  return true
}

export function respondPendingActions(
  req: IncomingMessage,
  res: ServerResponse,
  actions: unknown[] | ((url: URL) => unknown[]),
) {
  const url = new URL(req.url || '/', 'http://127.0.0.1')
  const isSessionPendingList = req.method === 'GET' && url.pathname === '/api/v1/bot/admission/sessions/pending'
  const isQueuedActionClaim = req.method === 'POST' && url.pathname === '/api/v1/bot/admission/actions/claim'
  if (!isSessionPendingList && !isQueuedActionClaim) return false
  assert.equal(req.headers.authorization, 'Bearer test-token')
  const data = typeof actions === 'function' ? actions(url) : actions
  res.setHeader('content-type', 'application/json')
  res.end(JSON.stringify({ success: true, data }))
  return true
}

export function respondAdmissionEvent(input: {
  readonly req: IncomingMessage
  readonly res: ServerResponse
  readonly events: unknown[]
  readonly afterEvent?: () => void
}) {
  const { req, res, events, afterEvent } = input
  const match = (req.url || '').match(/^\/api\/v1\/bot\/admission\/sessions\/([^/]+)\/events$/)
  if (req.method !== 'POST' || !match) return false
  assert.equal(req.headers.authorization, 'Bearer test-token')
  const chunks: Buffer[] = []
  req.on('data', (chunk: Buffer) => chunks.push(chunk))
  req.on('end', () => {
    events.push({ sessionID: decodeURIComponent(match[1]), body: JSON.parse(Buffer.concat(chunks).toString()) })
    afterEvent?.()
    res.setHeader('content-type', 'application/json')
    res.end(JSON.stringify({ success: true }))
  })
  return true
}

export function respondAdmissionPolicyTargets(
  req: IncomingMessage,
  res: ServerResponse,
  targets: unknown[] = [],
) {
  const url = new URL(req.url || '/', 'http://127.0.0.1')
  const path = '/api/v1/bot/admission/policies/targets'
  if (req.method !== 'GET' || url.pathname !== path) return false
  assert.equal(req.headers.authorization, 'Bearer test-token')
  res.setHeader('content-type', 'application/json')
  res.end(JSON.stringify({ success: true, data: targets }))
  return true
}

export function admissionAction(qqID: string, guildID: string, action: string) {
  return {
    sessionID: `session-${qqID}`,
    action,
    platform: 'qq',
    guildID,
    channelID: guildID,
    qqID,
    deadlineAt: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
  }
}

function admissionSessionData(qqID: string, guildID: string) {
  const now = Date.now()
  return {
    id: `session-${qqID}`,
    platform: 'qq',
    guildID,
    channelID: guildID,
    qqID,
    status: 'awaiting_account_link',
    tokenExpiresAt: new Date(now + 60 * 60 * 1000).toISOString(),
    linkWaitDeadlineAt: new Date(now + 60 * 60 * 1000).toISOString(),
    submissionWaitDeadlineAt: new Date(now + 24 * 60 * 60 * 1000).toISOString(),
    initialMuteUntil: new Date(now + 30 * 24 * 60 * 60 * 1000).toISOString(),
    projectionPending: false,
    failureCount: 0,
    remainingRetryCount: 2,
    willBlacklistOnTimeout: false,
  }
}
