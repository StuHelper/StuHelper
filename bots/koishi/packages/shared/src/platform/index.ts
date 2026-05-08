import type {
  AdmissionBotEventRequest,
  AdmissionJoinRequestEvent,
  AdmissionQQAccessQuery,
  AdmissionPendingAction,
  AdmissionPendingActionsRequest,
  AdmissionQQAccess,
  AdmissionSessionCreateRequest,
  AdmissionSessionCreateResult,
  ConsumeQQBindingRequest,
  FreshmanBlacklistReleaseRequest,
  FreshmanApplication,
  FreshmanCommandContext,
  FreshmanForwardItem,
  FreshmanReviewRequest,
  MemberBlacklistAccessDecision,
  MemberBlacklistAccessQuery,
  MemberBlacklistCreateRequest,
  MemberBlacklistEntry,
  MemberBlacklistListQuery,
  MemberBlacklistListResult,
  MemberBlacklistReleaseBySubjectRequest,
  MemberBlacklistReleaseRequest,
  QQVerificationStatus,
  StuhelperPlatformConfig,
  QQBinding,
} from '../types/index'
import { createFreshmanClient } from './freshman-client'

const HEALTH_PATH = '/health/live'
const QQ_BINDING_CONSUME_PATH = '/api/v1/bot/qq-binding/consume'
const QQ_VERIFICATION_PATH_PREFIX = '/api/v1/bot/qq-users/'
const ADMISSION_SESSIONS_PATH = '/api/v1/bot/admission/sessions'
const ADMISSION_JOIN_REQUEST_EVENTS_PATH = '/api/v1/bot/admission/join-requests/events'
const ADMISSION_PENDING_ACTIONS_PATH = '/api/v1/bot/admission/sessions/pending'
const MEMBER_BLACKLIST_PATH = '/api/v1/bot/member-blacklist'
const AUTH_SCHEME = 'Bearer'
const JSON_CONTENT_TYPE = 'application/json'
const PLATFORM_REQUEST_TIMEOUT_MS = 8_000

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

export interface PlatformClient {
  getHealth(): Promise<void>
  consumeQQBindingCode(input: ConsumeQQBindingRequest): Promise<ConsumeQQBindingResult>
  getQQVerificationStatus(qqID: string): Promise<QQVerificationStatus>
  getAdmissionQQAccess(input: AdmissionQQAccessQuery): Promise<AdmissionQQAccess>
  getMemberBlacklistAccess(input: MemberBlacklistAccessQuery): Promise<MemberBlacklistAccessDecision>
  listMemberBlacklist(input?: MemberBlacklistListQuery): Promise<MemberBlacklistListResult>
  createMemberBlacklist(input: MemberBlacklistCreateRequest): Promise<MemberBlacklistEntry>
  releaseMemberBlacklist(id: string, input: MemberBlacklistReleaseRequest): Promise<void>
  releaseMemberBlacklistBySubject(input: MemberBlacklistReleaseBySubjectRequest): Promise<void>
  createAdmissionSession(input: AdmissionSessionCreateRequest): Promise<AdmissionSessionCreateResult>
  recordJoinRequestEvent(input: AdmissionJoinRequestEvent): Promise<void>
  listPendingAdmissionActions(input?: AdmissionPendingActionsRequest): Promise<readonly AdmissionPendingAction[]>
  recordAdmissionEvent(sessionID: string, input: AdmissionBotEventRequest): Promise<void>
  listPendingFreshmanForwards(): Promise<readonly FreshmanForwardItem[]>
  markFreshmanForwarded(applicationID: string): Promise<void>
  viewFreshmanApplication(applicationID: string, input: FreshmanCommandContext): Promise<FreshmanApplication>
  reviewFreshmanApplication(applicationID: string, input: FreshmanReviewRequest): Promise<FreshmanApplication>
  releaseAdmissionBlacklist(qqID: string, input: FreshmanBlacklistReleaseRequest): Promise<void>
}

export function createPlatformClient(config: StuhelperPlatformConfig): PlatformClient {
  assertPlatformConfig(config)
  const request = createRequest(config)

  return {
    ...createSystemClient(request),
    ...createBindingClient(request),
    ...createAdmissionClient(request),
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
): Pick<PlatformClient,
  'getAdmissionQQAccess'
  | 'createAdmissionSession'
  | 'recordJoinRequestEvent'
  | 'listPendingAdmissionActions'
  | 'recordAdmissionEvent'
  | 'releaseAdmissionBlacklist'
> {
  return {
    async getAdmissionQQAccess(input) {
      return request<AdmissionQQAccess>(
        withQuery(`/api/v1/bot/admission/qq-users/${encodeURIComponent(input.qqID)}/access`, {
          platform: input.platform,
          guildID: input.guildID,
        }),
        { method: 'GET' },
      )
    },

    async createAdmissionSession(input) {
      return request<AdmissionSessionCreateResult>(ADMISSION_SESSIONS_PATH, jsonPost(input))
    },

    async recordJoinRequestEvent(input) {
      await request<void>(ADMISSION_JOIN_REQUEST_EVENTS_PATH, jsonPost(input), true)
    },

    async listPendingAdmissionActions(input = {}) {
      return request<readonly AdmissionPendingAction[]>(withQuery(ADMISSION_PENDING_ACTIONS_PATH, input), {
        method: 'GET',
      })
    },

    async recordAdmissionEvent(sessionID, input) {
      await request<void>(`${ADMISSION_SESSIONS_PATH}/${encodeURIComponent(sessionID)}/events`, jsonPost(input), true)
    },

    async releaseAdmissionBlacklist(qqID, input) {
      await request<void>(
        `/api/v1/bot/admission/blacklist/${encodeURIComponent(qqID)}/release`,
        jsonPost(input),
        true,
      )
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
    async getMemberBlacklistAccess(input) {
      return request<MemberBlacklistAccessDecision>(
        withQuery(`${MEMBER_BLACKLIST_PATH}/access`, input),
        { method: 'GET' },
      )
    },

    async listMemberBlacklist(input = {}) {
      return request<MemberBlacklistListResult>(withQuery(MEMBER_BLACKLIST_PATH, input), { method: 'GET' })
    },

    async createMemberBlacklist(input) {
      return request<MemberBlacklistEntry>(MEMBER_BLACKLIST_PATH, jsonPost(input))
    },

    async releaseMemberBlacklist(id, input) {
      await request<void>(`${MEMBER_BLACKLIST_PATH}/${encodeURIComponent(id)}/release`, jsonPost(input), true)
    },

    async releaseMemberBlacklistBySubject(input) {
      await request<void>(`${MEMBER_BLACKLIST_PATH}/release-by-subject`, jsonPost(input), true)
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

function appendQueryValue(values: URLSearchParams, key: string, value: unknown) {
  if (typeof value !== 'undefined' && value !== '') {
    values.set(key, String(value))
  }
}

function assertPlatformConfig(config: StuhelperPlatformConfig) {
  if (!config.serviceToken?.trim()) {
    throw new Error('platform service token is required')
  }
  if (!config.baseUrl?.trim()) {
    throw new Error('platform baseUrl is required')
  }
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
