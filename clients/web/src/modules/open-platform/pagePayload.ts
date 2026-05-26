import type {
  OpenPlatformConsentPageResponse,
  OpenPlatformProfileCompletionPageResponse,
} from '@stuhelper/shared/api'

type ConsentApp = OpenPlatformConsentPageResponse['app']
type ScopeDefinition = OpenPlatformConsentPageResponse['scopes'][number]
type ProfileCompletionField =
  OpenPlatformProfileCompletionPageResponse['missingFields'][number]

const SENSITIVITY_VALUES = new Set(['low', 'medium', 'high', 'very_high'])
const SCOPE_VALUES = new Set([
  'profile.basic.read',
  'email.read',
  'phone.read',
  'stu.identity.status.read',
  'stu.identity.type.read',
  'stu.student.status.read',
  'stu.student.school.read',
  'resource.read',
  'resource.write',
  'offline_access',
])
const PROFILE_FIELD_KEYS = new Set([
  'profile.username',
  'profile.email',
  'profile.avatar',
  'profile.phone',
  'profile.identity',
  'profile.student',
  'profile.school',
])

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object'
}

function readString(record: Record<string, unknown>, key: string, message: string): string {
  const value = record[key]
  if (typeof value !== 'string') {
    throw new Error(message)
  }
  return value
}

function readNumber(record: Record<string, unknown>, key: string, message: string): number {
  const value = record[key]
  if (typeof value !== 'number' || !Number.isFinite(value)) {
    throw new Error(message)
  }
  return value
}

function readAbsoluteURL(record: Record<string, unknown>, key: string, message: string): string {
  const value = readString(record, key, message)
  try {
    const url = new URL(value)
    if (url.protocol !== 'http:' && url.protocol !== 'https:') {
      throw new Error(message)
    }
  } catch {
    throw new Error(message)
  }
  return value
}

function readStringArray(value: unknown, message: string): string[] {
  if (!Array.isArray(value) || value.some((item) => typeof item !== 'string')) {
    throw new Error(message)
  }
  return value
}

function readConsentApp(value: unknown, message: string): ConsentApp {
  if (!isRecord(value)) {
    throw new Error(message)
  }

  return {
    id: readNumber(value, 'id', message),
    clientID: readString(value, 'clientID', message),
    displayName: readString(value, 'displayName', message),
    description: readString(value, 'description', message),
    homepageURL: readAbsoluteURL(value, 'homepageURL', message),
    privacyPolicyURL: readAbsoluteURL(value, 'privacyPolicyURL', message),
  }
}

function readScopeDefinition(value: unknown, message: string): ScopeDefinition {
  if (!isRecord(value)) {
    throw new Error(message)
  }

  const sensitivity = readString(value, 'sensitivity', message)
  if (!SENSITIVITY_VALUES.has(sensitivity)) {
    throw new Error(message)
  }
  const scope = readString(value, 'scope', message)
  if (!SCOPE_VALUES.has(scope)) {
    throw new Error(message)
  }

  return {
    scope: scope as ScopeDefinition['scope'],
    displayName: readString(value, 'displayName', message),
    sensitivity: sensitivity as ScopeDefinition['sensitivity'],
    fields: readStringArray(value.fields, message),
    reason: readString(value, 'reason', message),
  }
}

function readProfileCompletionField(value: unknown, message: string): ProfileCompletionField {
  if (!isRecord(value)) {
    throw new Error(message)
  }

  const key = readString(value, 'key', message)
  if (!PROFILE_FIELD_KEYS.has(key)) {
    throw new Error(message)
  }

  return {
    key: key as ProfileCompletionField['key'],
    displayName: readString(value, 'displayName', message),
    actionURL: readString(value, 'actionURL', message),
  }
}

function readScopeDefinitions(value: unknown, message: string): ScopeDefinition[] {
  if (!Array.isArray(value)) {
    throw new Error(message)
  }
  return value.map((item) => readScopeDefinition(item, message))
}

export function readConsentPagePayload(
  payload: unknown,
  message = 'Invalid consent response',
): OpenPlatformConsentPageResponse {
  if (!isRecord(payload)) {
    throw new Error(message)
  }

  return {
    token: readString(payload, 'token', message),
    app: readConsentApp(payload.app, message),
    scopes: readScopeDefinitions(payload.scopes, message),
    redirectURI: readAbsoluteURL(payload, 'redirectURI', message),
    expiresAt: readString(payload, 'expiresAt', message),
  }
}

export function readProfileCompletionPagePayload(
  payload: unknown,
  message = 'Invalid profile completion response',
): OpenPlatformProfileCompletionPageResponse {
  if (!isRecord(payload)) {
    throw new Error(message)
  }
  if (!Array.isArray(payload.missingFields)) {
    throw new Error(message)
  }

  return {
    token: readString(payload, 'token', message),
    app: readConsentApp(payload.app, message),
    scopes: readScopeDefinitions(payload.scopes, message),
    missingFields: payload.missingFields.map((item) =>
      readProfileCompletionField(item, message),
    ),
    redirectURI: readAbsoluteURL(payload, 'redirectURI', message),
    expiresAt: readString(payload, 'expiresAt', message),
  }
}

export function readRedirectURLPayload(
  payload: unknown,
  message = 'Invalid redirect response',
): string {
  if (!isRecord(payload)) {
    throw new Error(message)
  }
  return readString(payload, 'redirectURL', message)
}

export function readAuthorizationTargetPayload(
  payload: unknown,
  message = 'Invalid authorization response',
): string {
  if (!isRecord(payload)) {
    throw new Error(message)
  }

  for (const key of ['redirectURL', 'consentURL', 'profileCompletionURL']) {
    const value = payload[key]
    if (value !== undefined) {
      if (typeof value !== 'string' || !value) {
        throw new Error(message)
      }
      return value
    }
  }

  throw new Error(message)
}
