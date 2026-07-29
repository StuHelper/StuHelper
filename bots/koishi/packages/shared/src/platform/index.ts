import type {
  AdmissionBotEventRequest,
  AdmissionBotActionEventRequest,
  AdmissionFailureResetResult,
  AdmissionJoinRequestDecision,
  AdmissionJoinRequestDecisionRequest,
  AdmissionJoinRequestEvent,
  AdmissionPolicyTarget,
  AdmissionPendingAction,
  AdmissionPendingActionsRequest,
  AdmissionSession,
  AdmissionSessionCreateRequest,
  AdmissionSessionCreateResult,
  AdmissionSessionOperatorRequest,
  AdmissionSessionSubjectRequest,
  ConsumeQQBindingRequest,
  FreshmanApplication,
  FreshmanCommandContext,
  FreshmanForwardItem,
  FreshmanReviewRequest,
  MemberBlacklistAccessDecision,
  MemberBlacklistAccessRequest,
  MemberBlacklistCreateRequest,
  MemberBlacklistEntry,
  MemberBlacklistListRequest,
  MemberBlacklistListResult,
  MemberBlacklistReleaseBySubjectRequest,
  MemberBlacklistReleaseRequest,
  PlatformRequestOptions,
  QQVerificationStatus,
  StuhelperPlatformConfig,
  QQBinding,
} from '../types/index'
import { createFreshmanClient } from './freshman-client'

const HEALTH_PATH = '/health/live'
const QQ_BINDING_CONSUME_PATH = '/api/v1/bot/qq-binding/consume'
const QQ_VERIFICATION_PATH_PREFIX = '/api/v1/bot/qq-users/'
const ADMISSION_SESSIONS_PATH = '/api/v1/bot/admission/sessions'
const ADMISSION_FAILURES_PATH = '/api/v1/bot/admission/failures'
const ADMISSION_JOIN_REQUEST_DECISION_PATH = '/api/v1/bot/admission/join-requests/decision'
const ADMISSION_JOIN_REQUEST_EVENTS_PATH = '/api/v1/bot/admission/join-requests/events'
const ADMISSION_POLICY_TARGETS_PATH = '/api/v1/bot/admission/policies/targets'
const ADMISSION_PENDING_ACTIONS_PATH = '/api/v1/bot/admission/sessions/pending'
const ADMISSION_ACTION_STREAM_PATH = '/api/v1/bot/admission/actions/stream'
const ADMISSION_ACTION_CLAIM_PATH = '/api/v1/bot/admission/actions/claim'
const ADMISSION_ACTIONS_PATH = '/api/v1/bot/admission/actions'
const MEMBER_BLACKLIST_PATH = '/api/v1/bot/member-blacklist'
const AUTH_SCHEME = 'Bearer'
const JSON_CONTENT_TYPE = 'application/json'
const PLATFORM_REQUEST_TIMEOUT_MS = 8_000
const PLATFORM_BASE_URL_ENV = 'STUHELPER_PLATFORM_BASE_URL'
const PLATFORM_SERVICE_TOKEN_ENV = 'STUHELPER_PLATFORM_SERVICE_TOKEN'
const ENV_PLACEHOLDER_RE = /^\s*\$\{\{\s*env\.([A-Za-z_][A-Za-z0-9_]*)\s*\}\}\s*$/

interface APIErrorPayload {
  code?: string
  message?: string
}

interface APIEnvelope<T> {
  data?: T
  error?: APIErrorPayload
  success?: boolean
}

export class PlatformAPIError extends Error {
  constructor(
    message: string,
    readonly status: number,
    readonly code?: string,
  ) {
    super(message)
    this.name = 'PlatformAPIError'
  }
}

export interface ConsumeQQBindingResult {
  binding: QQBinding
  verificationState: QQVerificationStatus
}

export interface AdmissionActionStreamHandlers {
  onAction(action: AdmissionPendingAction): void | Promise<void>
  onError?(error: unknown): void
  onOpen?(): void
}

export interface AdmissionActionStreamHandle {
  close(): void
}

export interface PlatformClient {
  getHealth(): Promise<void>
  consumeQQBindingCode(input: ConsumeQQBindingRequest): Promise<ConsumeQQBindingResult>
  getQQVerificationStatus(qqID: string): Promise<QQVerificationStatus>
  getMemberBlacklistAccess(
    input: MemberBlacklistAccessRequest,
    options?: PlatformRequestOptions,
  ): Promise<MemberBlacklistAccessDecision>
  listMemberBlacklist(input?: MemberBlacklistListRequest): Promise<MemberBlacklistListResult>
  createMemberBlacklist(input: MemberBlacklistCreateRequest): Promise<MemberBlacklistEntry>
  releaseMemberBlacklist(id: string, input: MemberBlacklistReleaseRequest): Promise<MemberBlacklistEntry>
  releaseMemberBlacklistBySubject(input: MemberBlacklistReleaseBySubjectRequest): Promise<MemberBlacklistEntry>
  createAdmissionSession(input: AdmissionSessionCreateRequest): Promise<AdmissionSessionCreateResult>
  getAdmissionSessionByMember(input: AdmissionSessionSubjectRequest): Promise<AdmissionSession>
  resendAdmissionSessionLink(input: AdmissionSessionSubjectRequest): Promise<AdmissionSession>
  regenerateAdmissionSessionLink(input: AdmissionSessionCreateRequest): Promise<AdmissionSessionCreateResult>
  skipAdmissionSessionForMember(input: AdmissionSessionOperatorRequest): Promise<AdmissionSession>
  resetAdmissionFailureCount(input: AdmissionSessionOperatorRequest): Promise<AdmissionFailureResetResult>
  resolveJoinRequestDecision(input: AdmissionJoinRequestDecisionRequest): Promise<AdmissionJoinRequestDecision>
  recordJoinRequestEvent(input: AdmissionJoinRequestEvent): Promise<void>
  listAdmissionPolicyTargets(): Promise<readonly AdmissionPolicyTarget[]>
  listPendingAdmissionActions(input: AdmissionPendingActionsRequest): Promise<readonly AdmissionPendingAction[]>
  claimQueuedAdmissionActions(input: AdmissionPendingActionsRequest): Promise<readonly AdmissionPendingAction[]>
  recordAdmissionEvent(sessionID: string, input: AdmissionBotEventRequest): Promise<void>
  streamAdmissionActions(input: AdmissionPendingActionsRequest, handlers: AdmissionActionStreamHandlers): AdmissionActionStreamHandle
  recordAdmissionActionEvent(actionID: string, input: AdmissionBotActionEventRequest): Promise<void>
  listPendingFreshmanForwards(): Promise<readonly FreshmanForwardItem[]>
  markFreshmanForwarded(applicationID: string): Promise<void>
  viewFreshmanApplication(applicationID: string, input: FreshmanCommandContext): Promise<FreshmanApplication>
  reviewFreshmanApplication(applicationID: string, input: FreshmanReviewRequest): Promise<FreshmanApplication>
}

export function createPlatformClient(config: StuhelperPlatformConfig): PlatformClient {
  const resolvedConfig = resolvePlatformConfig(config)
  assertPlatformConfig(resolvedConfig)
  const request = createRequest(resolvedConfig)

  return {
    ...createSystemClient(request),
    ...createBindingClient(request),
    ...createAdmissionClient(request, resolvedConfig),
    ...createMemberBlacklistClient(request),
    ...createFreshmanClient(request),
  }
}

type PlatformRequest = ReturnType<typeof createRequest>

function createSystemClient(request: PlatformRequest): Pick<PlatformClient, 'getHealth'> {
  return {
    async getHealth() {
      await request<void>(HEALTH_PATH, { method: 'GET' }, true)
    },
  }
}

function createBindingClient(
  request: PlatformRequest,
): Pick<PlatformClient, 'consumeQQBindingCode' | 'getQQVerificationStatus'> {
  return {
    async consumeQQBindingCode(input) {
      return request<ConsumeQQBindingResult>(QQ_BINDING_CONSUME_PATH, {
        method: 'POST',
        headers: { 'Content-Type': JSON_CONTENT_TYPE },
        body: JSON.stringify(input),
      })
    },

    async getQQVerificationStatus(qqID) {
      return request<QQVerificationStatus>(`${QQ_VERIFICATION_PATH_PREFIX}${encodeURIComponent(qqID)}/verification`, {
        method: 'GET',
      })
    },
  }
}

function createAdmissionClient(
  request: PlatformRequest,
  config: StuhelperPlatformConfig,
): Pick<PlatformClient,
  'createAdmissionSession'
  | 'getAdmissionSessionByMember'
  | 'resendAdmissionSessionLink'
  | 'regenerateAdmissionSessionLink'
  | 'skipAdmissionSessionForMember'
  | 'resetAdmissionFailureCount'
  | 'resolveJoinRequestDecision'
  | 'recordJoinRequestEvent'
  | 'listAdmissionPolicyTargets'
  | 'listPendingAdmissionActions'
  | 'claimQueuedAdmissionActions'
  | 'recordAdmissionEvent'
  | 'streamAdmissionActions'
  | 'recordAdmissionActionEvent'
> {
  return {
    async createAdmissionSession(input) {
      return request<AdmissionSessionCreateResult>(ADMISSION_SESSIONS_PATH, jsonPost(input))
    },

    async getAdmissionSessionByMember(input) {
      assertAdmissionSessionSubjectRequest(input)
      return request<AdmissionSession>(withQuery(`${ADMISSION_SESSIONS_PATH}/member`, input), {
        method: 'GET',
      })
    },

    async resendAdmissionSessionLink(input) {
      assertAdmissionSessionSubjectRequest(input)
      return request<AdmissionSession>(`${ADMISSION_SESSIONS_PATH}/member/resend`, jsonPost(input))
    },

    async regenerateAdmissionSessionLink(input) {
      return request<AdmissionSessionCreateResult>(`${ADMISSION_SESSIONS_PATH}/member/regenerate`, jsonPost(input))
    },

    async skipAdmissionSessionForMember(input) {
      assertAdmissionSessionOperatorRequest(input)
      return request<AdmissionSession>(`${ADMISSION_SESSIONS_PATH}/member/skip`, jsonPost(input))
    },

    async resetAdmissionFailureCount(input) {
      assertAdmissionSessionOperatorRequest(input)
      return request<AdmissionFailureResetResult>(`${ADMISSION_FAILURES_PATH}/reset`, jsonPost(input))
    },

    async resolveJoinRequestDecision(input) {
      return request<AdmissionJoinRequestDecision>(ADMISSION_JOIN_REQUEST_DECISION_PATH, jsonPost(input))
    },

    async recordJoinRequestEvent(input) {
      await request<void>(ADMISSION_JOIN_REQUEST_EVENTS_PATH, jsonPost(input), true)
    },

    async listAdmissionPolicyTargets() {
      return request<readonly AdmissionPolicyTarget[]>(ADMISSION_POLICY_TARGETS_PATH, { method: 'GET' })
    },

    async listPendingAdmissionActions(input) {
      assertPendingAdmissionActionsRequest(input)
      return request<readonly AdmissionPendingAction[]>(withQuery(ADMISSION_PENDING_ACTIONS_PATH, input), {
        method: 'GET',
      })
    },

    async claimQueuedAdmissionActions(input) {
      assertPendingAdmissionActionsRequest(input)
      return request<readonly AdmissionPendingAction[]>(withQuery(ADMISSION_ACTION_CLAIM_PATH, input), {
        method: 'POST',
      })
    },

    async recordAdmissionEvent(sessionID, input) {
      await request<void>(`${ADMISSION_SESSIONS_PATH}/${encodeURIComponent(sessionID)}/events`, jsonPost(input), true)
    },

    streamAdmissionActions(input, handlers) {
      assertPendingAdmissionActionsRequest(input)
      return createAdmissionActionStream(config, input, handlers)
    },

    async recordAdmissionActionEvent(actionID, input) {
      await request<void>(`${ADMISSION_ACTIONS_PATH}/${encodeURIComponent(actionID)}/events`, jsonPost(input), true)
    },
  }
}

function createMemberBlacklistClient(
  request: PlatformRequest,
): Pick<PlatformClient,
  'getMemberBlacklistAccess'
  | 'listMemberBlacklist'
  | 'createMemberBlacklist'
  | 'releaseMemberBlacklist'
  | 'releaseMemberBlacklistBySubject'
> {
  return {
    async getMemberBlacklistAccess(input, options) {
      return request<MemberBlacklistAccessDecision>(
        withQuery(`${MEMBER_BLACKLIST_PATH}/access`, input),
        withRequestOptions({ method: 'GET' }, options),
      )
    },

    async listMemberBlacklist(input = {}) {
      return request<MemberBlacklistListResult>(withQuery(MEMBER_BLACKLIST_PATH, input), { method: 'GET' })
    },

    async createMemberBlacklist(input) {
      return request<MemberBlacklistEntry>(MEMBER_BLACKLIST_PATH, jsonPost(input))
    },

    async releaseMemberBlacklist(id, input) {
      return request<MemberBlacklistEntry>(`${MEMBER_BLACKLIST_PATH}/${encodeURIComponent(id)}/release`, jsonPost(input))
    },

    async releaseMemberBlacklistBySubject(input) {
      return request<MemberBlacklistEntry>(`${MEMBER_BLACKLIST_PATH}/release-by-subject`, jsonPost(input))
    },
  }
}

function jsonPost(input: unknown): RequestInit {
  return {
    method: 'POST',
    headers: { 'Content-Type': JSON_CONTENT_TYPE },
    body: JSON.stringify(input),
  }
}

function withQuery(path: string, input: object) {
  const values = new URLSearchParams()
  for (const [key, value] of Object.entries(input)) {
    appendQueryValue(values, key, value)
  }
  const query = values.toString()
  return query ? `${path}?${query}` : path
}

function createAdmissionActionStream(
  config: StuhelperPlatformConfig,
  input: AdmissionPendingActionsRequest,
  handlers: AdmissionActionStreamHandlers,
): AdmissionActionStreamHandle {
  const controller = new AbortController()
  void runAdmissionActionStream(config, input, handlers, controller.signal)
    .catch((error) => {
      if (!controller.signal.aborted) {
        handlers.onError?.(error)
      }
    })
  return {
    close() {
      controller.abort()
    },
  }
}

async function runAdmissionActionStream(
  config: StuhelperPlatformConfig,
  input: AdmissionPendingActionsRequest,
  handlers: AdmissionActionStreamHandlers,
  signal: AbortSignal,
) {
  const endpoint = new URL(withQuery(ADMISSION_ACTION_STREAM_PATH, input), config.baseUrl)
  const response = await fetch(endpoint, withAuthHeaders(config, { method: 'GET', signal }))
  if (!response.ok) {
    throw await buildPlatformError(response)
  }
  if (!response.body) {
    throw new Error('admission action stream response missing body')
  }
  handlers.onOpen?.()
  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  while (!signal.aborted) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    const parts = buffer.split(/\r?\n\r?\n/)
    buffer = parts.pop() ?? ''
    for (const part of parts) {
      await dispatchAdmissionActionStreamEvent(part, handlers)
    }
  }
  if (!signal.aborted) {
    throw new Error('admission action stream closed')
  }
}

async function dispatchAdmissionActionStreamEvent(
  raw: string,
  handlers: AdmissionActionStreamHandlers,
) {
  let event = 'message'
  const data: string[] = []
  for (const line of raw.split(/\r?\n/)) {
    if (line.startsWith('event:')) {
      event = line.slice('event:'.length).trim()
      continue
    }
    if (line.startsWith('data:')) {
      data.push(line.slice('data:'.length).trimStart())
    }
  }
  if (event !== 'action' || data.length === 0) {
    return
  }
  await handlers.onAction(JSON.parse(data.join('\n')) as AdmissionPendingAction)
}

function appendQueryValue(values: URLSearchParams, key: string, value: unknown) {
  if (typeof value !== 'undefined' && value !== null && value !== '') {
    values.set(key, String(value))
  }
}

function assertPendingAdmissionActionsRequest(input: AdmissionPendingActionsRequest) {
  if (!input.platform?.trim() || !input.botSelfID?.trim()) {
    throw new Error('platform and botSelfID are required for pending admission actions')
  }
}

function assertAdmissionSessionSubjectRequest(input: AdmissionSessionSubjectRequest) {
  if (!input.platform?.trim() || !input.guildID?.trim() || !input.qqID?.trim()) {
    throw new Error('platform, guildID and qqID are required for admission session member queries')
  }
}

function assertAdmissionSessionOperatorRequest(input: AdmissionSessionOperatorRequest) {
  assertAdmissionSessionSubjectRequest(input)
  if (!input.operatorQQID?.trim()) {
    throw new Error('operatorQQID is required for admission member operations')
  }
}

function withRequestOptions(init: RequestInit, options?: PlatformRequestOptions): RequestInit {
  if (!options?.timeoutMs) return init
  return { ...init, signal: AbortSignal.timeout(options.timeoutMs) }
}

function assertPlatformConfig(config: StuhelperPlatformConfig) {
  if (!config.serviceToken?.trim()) {
    throw new Error('platform service token is required')
  }
  if (!config.baseUrl?.trim()) {
    throw new Error('platform baseUrl is required')
  }
  try {
    new URL(config.baseUrl)
  } catch {
    throw new Error('platform baseUrl must be an absolute URL')
  }
}

function resolvePlatformConfig(config: StuhelperPlatformConfig): StuhelperPlatformConfig {
  return {
    baseUrl: resolveConfigValue(config.baseUrl, PLATFORM_BASE_URL_ENV),
    serviceToken: resolveConfigValue(config.serviceToken, PLATFORM_SERVICE_TOKEN_ENV),
  }
}

function resolveConfigValue(value: string | undefined, fallbackEnvName: string) {
  const raw = value?.trim() ?? ''
  const placeholder = raw.match(ENV_PLACEHOLDER_RE)
  if (placeholder) {
    return process.env[placeholder[1]]?.trim() ?? ''
  }
  if (raw) {
    return raw
  }
  return process.env[fallbackEnvName]?.trim() ?? ''
}

function createRequest(config: StuhelperPlatformConfig) {
  return async function request<T>(path: string, init: RequestInit, allowEmptyData = false): Promise<T> {
    const endpoint = new URL(path, config.baseUrl)
    const response = await fetch(endpoint, withAuthHeaders(config, withDefaultTimeout(init)))
    if (!response.ok) {
      throw await buildPlatformError(response)
    }

    const body = await parseJSONBody<T>(response)
    if (!body) {
      if (allowEmptyData) {
        return undefined as T
      }
      throw new Error(`platform response missing data for ${path}`)
    }
    if (!allowEmptyData && typeof body.data === 'undefined') {
      throw new Error(`platform response missing data for ${path}`)
    }
    return body.data as T
  }
}

function withAuthHeaders(config: StuhelperPlatformConfig, init: RequestInit): RequestInit {
  const headers = new Headers(init.headers)
  headers.set('Authorization', `${AUTH_SCHEME} ${config.serviceToken}`)
  return { ...init, headers }
}

function withDefaultTimeout(init: RequestInit): RequestInit {
  return {
    ...init,
    signal: init.signal ?? AbortSignal.timeout(PLATFORM_REQUEST_TIMEOUT_MS),
  }
}

async function buildPlatformError(response: Response): Promise<Error> {
  const body = await parseJSONBody<unknown>(response)
  const code = body?.error?.code
  const message = body?.error?.message || `platform request failed: ${response.status}`
  return new PlatformAPIError(message, response.status, code)
}

async function parseJSONBody<T>(response: Response): Promise<APIEnvelope<T> | null> {
  const contentType = response.headers.get('content-type') || ''
  if (!contentType.includes(JSON_CONTENT_TYPE)) {
    return null
  }
  return response.json() as Promise<APIEnvelope<T>>
}
