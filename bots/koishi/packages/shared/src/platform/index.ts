import type {
  AdmissionBotEventRequest,
  AdmissionJoinRequestEvent,
  AdmissionPendingAction,
  AdmissionPendingActionsRequest,
  AdmissionSessionCreateRequest,
  AdmissionSessionCreateResult,
  ConsumeQQBindingRequest,
  FreshmanApplication,
  FreshmanForwardItem,
  FreshmanReviewRequest,
  QQVerificationStatus,
  StuhelperPlatformConfig,
  QQBinding,
} from '../types/index'

const HEALTH_PATH = '/health/live'
const QQ_BINDING_CONSUME_PATH = '/api/v1/bot/qq-binding/consume'
const QQ_VERIFICATION_PATH_PREFIX = '/api/v1/bot/qq-users/'
const ADMISSION_SESSIONS_PATH = '/api/v1/bot/admission/sessions'
const ADMISSION_JOIN_REQUEST_EVENTS_PATH = '/api/v1/bot/admission/join-requests/events'
const ADMISSION_PENDING_ACTIONS_PATH = '/api/v1/bot/admission/sessions/pending'
const ADMISSION_FRESHMAN_FORWARD_PATH = '/api/v1/bot/admission/freshman/applications/pending-forward'
const ADMISSION_FRESHMAN_APPLICATIONS_PATH = '/api/v1/bot/admission/freshman/applications'
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
  createAdmissionSession(input: AdmissionSessionCreateRequest): Promise<AdmissionSessionCreateResult>
  recordJoinRequestEvent(input: AdmissionJoinRequestEvent): Promise<void>
  listPendingAdmissionActions(input?: AdmissionPendingActionsRequest): Promise<readonly AdmissionPendingAction[]>
  recordAdmissionEvent(sessionID: string, input: AdmissionBotEventRequest): Promise<void>
  listPendingFreshmanForwards(): Promise<readonly FreshmanForwardItem[]>
  markFreshmanForwarded(applicationID: string): Promise<void>
  reviewFreshmanApplication(applicationID: string, input: FreshmanReviewRequest): Promise<FreshmanApplication>
}

export function createPlatformClient(config: StuhelperPlatformConfig): PlatformClient {
  const request = createRequest(config)

  return {
    async getHealth() {
      await request<void>(HEALTH_PATH, { method: 'GET' }, true)
    },

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

    async listPendingFreshmanForwards() {
      return request<readonly FreshmanForwardItem[]>(ADMISSION_FRESHMAN_FORWARD_PATH, { method: 'GET' })
    },

    async markFreshmanForwarded(applicationID) {
      await request<void>(`${ADMISSION_FRESHMAN_APPLICATIONS_PATH}/${encodeURIComponent(applicationID)}/forwarded`, {
        method: 'POST',
      }, true)
    },

    async reviewFreshmanApplication(applicationID, input) {
      return request<FreshmanApplication>(
        `${ADMISSION_FRESHMAN_APPLICATIONS_PATH}/${encodeURIComponent(applicationID)}/review`,
        jsonPost(input),
      )
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

function withQuery(path: string, input: AdmissionPendingActionsRequest) {
  const values = new URLSearchParams()
  appendQueryValue(values, 'platform', input.platform)
  appendQueryValue(values, 'botSelfID', input.botSelfID)
  const query = values.toString()
  return query ? `${path}?${query}` : path
}

function appendQueryValue(values: URLSearchParams, key: string, value: string | undefined) {
  if (value) {
    values.set(key, value)
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
    if (!body || (!allowEmptyData && typeof body.data === 'undefined')) {
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
