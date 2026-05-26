import type {
  OpenPlatformAppListResponse,
  OpenPlatformAppRedirectURIRequest,
  OpenPlatformAppScopeRequest,
  OpenPlatformAppWithScopes,
  OpenPlatformDeveloperAppAuditEvent,
  OpenPlatformDeveloperAppAuditEventsResponse,
  OpenPlatformRotatedSecretResponse,
  OpenPlatformUserAuthorizedApp,
  OpenPlatformUserConsentAuditEvent,
  OpenPlatformUserConsentAuditEventsResponse,
  OpenPlatformUserConsentScope,
  OpenPlatformUserConsentsResponse,
} from '@stuhelper/shared/api'

type OpenPlatformApp = OpenPlatformAppWithScopes['app']

const APP_STATUS_VALUES = new Set(['pending', 'approved', 'suspended', 'revoked'])
const REQUEST_STATUS_VALUES = new Set(['pending', 'approved', 'rejected', 'withdrawn'])
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

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value)
}

function readString(record: Record<string, unknown>, key: string, message: string): string {
  const value = record[key]
  if (typeof value !== 'string') {
    throw new Error(message)
  }
  return value
}

function readNonEmptyString(
  record: Record<string, unknown>,
  key: string,
  message: string,
): string {
  const value = readString(record, key, message)
  if (!value) {
    throw new Error(message)
  }
  return value
}

function readNullableString(
  record: Record<string, unknown>,
  key: string,
  message: string,
): string | null | undefined {
  const value = record[key]
  if (value === undefined || value === null) {
    return value
  }
  if (typeof value !== 'string') {
    throw new Error(message)
  }
  return value
}

function readOptionalString(
  record: Record<string, unknown>,
  key: string,
  message: string,
): string | undefined {
  const value = record[key]
  if (value === undefined) {
    return undefined
  }
  if (typeof value !== 'string') {
    throw new Error(message)
  }
  return value
}

function readInteger(record: Record<string, unknown>, key: string, message: string): number {
  const value = record[key]
  if (typeof value !== 'number' || !Number.isInteger(value)) {
    throw new Error(message)
  }
  return value
}

function readNullableInteger(
  record: Record<string, unknown>,
  key: string,
  message: string,
): number | null | undefined {
  const value = record[key]
  if (value === undefined || value === null) {
    return value
  }
  if (typeof value !== 'number' || !Number.isInteger(value)) {
    throw new Error(message)
  }
  return value
}

function readEnum<T extends string>(
  record: Record<string, unknown>,
  key: string,
  values: Set<string>,
  message: string,
): T {
  const value = readString(record, key, message)
  if (!values.has(value)) {
    throw new Error(message)
  }
  return value as T
}

function readAbsoluteURLValue(value: unknown, message: string): string {
  if (typeof value !== 'string') {
    throw new Error(message)
  }
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

function readAbsoluteURL(
  record: Record<string, unknown>,
  key: string,
  message: string,
): string {
  return readAbsoluteURLValue(record[key], message)
}

function readStringArray(value: unknown, message: string): string[] {
  if (!Array.isArray(value) || value.some((item) => typeof item !== 'string')) {
    throw new Error(message)
  }
  return value
}

function readScopeArray(value: unknown, message: string): OpenPlatformUserConsentScope['scope'][] {
  const scopes = readStringArray(value, message)
  if (scopes.some(scope => !SCOPE_VALUES.has(scope))) {
    throw new Error(message)
  }
  return scopes as OpenPlatformUserConsentScope['scope'][]
}

function readAbsoluteURLArray(value: unknown, message: string): string[] {
  if (!Array.isArray(value)) {
    throw new Error(message)
  }
  return value.map(item => readAbsoluteURLValue(item, message))
}

function readDetails(record: Record<string, unknown>, key: string, message: string) {
  const value = record[key]
  if (!isRecord(value)) {
    throw new Error(message)
  }
  return value
}

function readOpenPlatformApp(value: unknown, message: string): OpenPlatformApp {
  if (!isRecord(value)) {
    throw new Error(message)
  }

  return {
    id: readInteger(value, 'id', message),
    clientID: readString(value, 'clientID', message),
    displayName: readString(value, 'displayName', message),
    description: readString(value, 'description', message),
    homepageURL: readAbsoluteURL(value, 'homepageURL', message),
    privacyPolicyURL: readAbsoluteURL(value, 'privacyPolicyURL', message),
    redirectURIs: readAbsoluteURLArray(value.redirectURIs, message),
    status: readEnum<OpenPlatformApp['status']>(value, 'status', APP_STATUS_VALUES, message),
    createdAt: readString(value, 'createdAt', message),
    updatedAt: readString(value, 'updatedAt', message),
  }
}

function readAppScopeRequest(value: unknown, message: string): OpenPlatformAppScopeRequest {
  if (!isRecord(value)) {
    throw new Error(message)
  }

  return {
    id: readInteger(value, 'id', message),
    scope: readEnum<OpenPlatformAppScopeRequest['scope']>(
      value,
      'scope',
      SCOPE_VALUES,
      message,
    ),
    displayName: readString(value, 'displayName', message),
    sensitivity: readEnum<OpenPlatformAppScopeRequest['sensitivity']>(
      value,
      'sensitivity',
      SENSITIVITY_VALUES,
      message,
    ),
    fields: readStringArray(value.fields, message),
    reason: readString(value, 'reason', message),
    status: readEnum<OpenPlatformAppScopeRequest['status']>(
      value,
      'status',
      REQUEST_STATUS_VALUES,
      message,
    ),
    reviewerUserID: readNullableInteger(value, 'reviewerUserID', message),
    reviewedAt: readNullableString(value, 'reviewedAt', message),
    decisionNote: readNullableString(value, 'decisionNote', message),
    createdAt: readString(value, 'createdAt', message),
    updatedAt: readString(value, 'updatedAt', message),
  }
}

function readRedirectURIRequest(
  value: unknown,
  message: string,
): OpenPlatformAppRedirectURIRequest {
  if (!isRecord(value)) {
    throw new Error(message)
  }

  return {
    id: readInteger(value, 'id', message),
    redirectURIs: readAbsoluteURLArray(value.redirectURIs, message),
    reason: readString(value, 'reason', message),
    status: readEnum<OpenPlatformAppRedirectURIRequest['status']>(
      value,
      'status',
      REQUEST_STATUS_VALUES,
      message,
    ),
    reviewerUserID: readNullableInteger(value, 'reviewerUserID', message),
    reviewedAt: readNullableString(value, 'reviewedAt', message),
    decisionNote: readNullableString(value, 'decisionNote', message),
    createdAt: readString(value, 'createdAt', message),
    updatedAt: readString(value, 'updatedAt', message),
  }
}

function readAppWithScopes(value: unknown, message: string): OpenPlatformAppWithScopes {
  if (!isRecord(value)) {
    throw new Error(message)
  }
  if (!Array.isArray(value.scopes) || !Array.isArray(value.redirectURIRequests)) {
    throw new Error(message)
  }

  return {
    app: readOpenPlatformApp(value.app, message),
    scopes: value.scopes.map(item => readAppScopeRequest(item, message)),
    redirectURIRequests: value.redirectURIRequests.map(item =>
      readRedirectURIRequest(item, message),
    ),
  }
}

function readDeveloperAuditEvent(
  value: unknown,
  message: string,
): OpenPlatformDeveloperAppAuditEvent {
  if (!isRecord(value)) {
    throw new Error(message)
  }

  return {
    id: readInteger(value, 'id', message),
    eventType: readString(value, 'eventType', message),
    scope: readNullableString(value, 'scope', message),
    scopes: readScopeArray(value.scopes, message),
    endpoint: readNullableString(value, 'endpoint', message),
    result: readNullableString(value, 'result', message),
    requestID: readNullableString(value, 'requestID', message),
    details: readDetails(value, 'details', message),
    createdAt: readString(value, 'createdAt', message),
  }
}

function readUserConsentScope(value: unknown, message: string): OpenPlatformUserConsentScope {
  if (!isRecord(value)) {
    throw new Error(message)
  }

  return {
    scope: readEnum<OpenPlatformUserConsentScope['scope']>(
      value,
      'scope',
      SCOPE_VALUES,
      message,
    ),
    displayName: readString(value, 'displayName', message),
    sensitivity: readEnum<OpenPlatformUserConsentScope['sensitivity']>(
      value,
      'sensitivity',
      SENSITIVITY_VALUES,
      message,
    ),
    fields: readStringArray(value.fields, message),
    grantedAt: readString(value, 'grantedAt', message),
    lastUsedAt: readOptionalString(value, 'lastUsedAt', message),
    grantSource: readString(value, 'grantSource', message),
    reason: readString(value, 'reason', message),
  }
}

function readUserAuthorizedApp(
  value: unknown,
  message: string,
): OpenPlatformUserAuthorizedApp {
  if (!isRecord(value) || !Array.isArray(value.scopes)) {
    throw new Error(message)
  }

  return {
    app: readOpenPlatformApp(value.app, message),
    scopes: value.scopes.map(item => readUserConsentScope(item, message)),
  }
}

function readUserConsentAuditEvent(
  value: unknown,
  message: string,
): OpenPlatformUserConsentAuditEvent {
  if (!isRecord(value)) {
    throw new Error(message)
  }

  return {
    id: readInteger(value, 'id', message),
    appID: readNullableInteger(value, 'appID', message),
    appDisplayName: readNullableString(value, 'appDisplayName', message),
    clientID: readNullableString(value, 'clientID', message),
    eventType: readString(value, 'eventType', message),
    scope: readNullableString(value, 'scope', message),
    scopes: readScopeArray(value.scopes, message),
    endpoint: readNullableString(value, 'endpoint', message),
    result: readNullableString(value, 'result', message),
    requestID: readNullableString(value, 'requestID', message),
    details: readDetails(value, 'details', message),
    createdAt: readString(value, 'createdAt', message),
  }
}

export function readOpenPlatformAppListPayload(
  payload: unknown,
  message = 'Invalid developer app list response',
): OpenPlatformAppListResponse {
  if (!isRecord(payload) || !Array.isArray(payload.list)) {
    throw new Error(message)
  }

  return {
    list: payload.list.map(item => readAppWithScopes(item, message)),
    total: readInteger(payload, 'total', message),
  }
}

export function readDeveloperAppAuditEventsPayload(
  payload: unknown,
  message = 'Invalid developer app audit response',
): OpenPlatformDeveloperAppAuditEventsResponse {
  if (!isRecord(payload) || !Array.isArray(payload.list)) {
    throw new Error(message)
  }

  return {
    list: payload.list.map(item => readDeveloperAuditEvent(item, message)),
    total: readInteger(payload, 'total', message),
  }
}

export function readRotatedSecretPayload(
  payload: unknown,
  message = 'Invalid rotated secret response',
): OpenPlatformRotatedSecretResponse {
  if (!isRecord(payload)) {
    throw new Error(message)
  }

  return {
    app: readOpenPlatformApp(payload.app, message),
    clientSecret: readNonEmptyString(payload, 'clientSecret', message),
  }
}

export function readUserConsentsPayload(
  payload: unknown,
  message = 'Invalid authorized apps response',
): OpenPlatformUserConsentsResponse {
  if (!isRecord(payload) || !Array.isArray(payload.apps)) {
    throw new Error(message)
  }

  return {
    apps: payload.apps.map(item => readUserAuthorizedApp(item, message)),
  }
}

export function readUserConsentAuditEventsPayload(
  payload: unknown,
  message = 'Invalid authorization activity response',
): OpenPlatformUserConsentAuditEventsResponse {
  if (!isRecord(payload) || !Array.isArray(payload.list)) {
    throw new Error(message)
  }

  return {
    list: payload.list.map(item => readUserConsentAuditEvent(item, message)),
    total: readInteger(payload, 'total', message),
  }
}
