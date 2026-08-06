import { createHash, timingSafeEqual } from 'node:crypto'

import { h, type Context } from 'koishi'

import type {
  AdmissionPolicyTarget,
  StuhelperAlertmanagerWebhookConfig,
} from '@stuhelper/koishi-shared'

import { resolveAdmissionSubjectPlatform } from './admission-subject-platform'

export const ALERTMANAGER_WEBHOOK_PATH = '/stuhelper/internal/alertmanager'

const RAW_BODY_BYPASS = Symbol.for('noParseBody')
const MAX_BODY_BYTES = 64 * 1024
const MAX_ALERTS = 50
const MAX_RECORD_ENTRIES = 64
const MAX_MESSAGE_ALERTS = 8
const MAX_MESSAGE_LENGTH = 3500
const DEDUPE_TTL_MS = 10 * 60 * 1000
const QQ_ID_PATTERN = /^[1-9][0-9]{4,19}$/
const TOKEN_PATTERN = /^[A-Za-z0-9._~+/=-]+$/
const LABEL_NAME_PATTERN = /^[A-Za-z_][A-Za-z0-9_]*$/

type AlertStatus = 'firing' | 'resolved'

interface AlertmanagerAlert {
  readonly status: AlertStatus
  readonly labels: Readonly<Record<string, string>>
  readonly annotations: Readonly<Record<string, string>>
  readonly fingerprint: string
}

interface AlertmanagerPayload {
  readonly version: '4'
  readonly status: AlertStatus
  readonly groupKey: string
  readonly receiver: string
  readonly groupLabels: Readonly<Record<string, string>>
  readonly commonLabels: Readonly<Record<string, string>>
  readonly commonAnnotations: Readonly<Record<string, string>>
  readonly alerts: readonly AlertmanagerAlert[]
}

export interface AlertDeliveryBot {
  readonly selfId: string
  readonly platform?: string
  sendMessage(channelID: string, content: ReturnType<typeof h.text>): Promise<unknown>
}

export interface AlertmanagerDeliveryDeps {
  readonly platform: {
    listAdmissionPolicyTargets(): Promise<readonly AdmissionPolicyTarget[]>
  }
  readonly getBots: () => readonly AlertDeliveryBot[]
  readonly botSelfID?: string
  readonly now?: () => number
}

interface AlertmanagerWebhookLogger {
  info(message: string, ...args: unknown[]): void
  warn(message: string, ...args: unknown[]): void
}

export interface AlertmanagerHTTPRequest {
  readonly authorization: string
  readonly contentType: string
  readonly contentLength: string
  readonly body: AsyncIterable<Uint8Array>
}

export interface AlertmanagerHTTPResponse {
  readonly status: number
  readonly code: string
}

interface AlertmanagerKoaContext {
  readonly req: AsyncIterable<Uint8Array>
  get(name: string): string
  set(name: string, value: string): void
  status: number
  body: unknown
}

export function registerAlertmanagerWebhook(
  ctx: Context,
  config: StuhelperAlertmanagerWebhookConfig,
  deps: Omit<AlertmanagerDeliveryDeps, 'botSelfID'>,
  logger: AlertmanagerWebhookLogger,
) {
  const normalizedConfig = validateAlertmanagerWebhookConfig(config)
  const server = ctx.get('server')
  if (!server) {
    throw new Error('alertmanager_webhook_server_unavailable')
  }

  const delivery = new AlertmanagerDeliveryService({
    ...deps,
    botSelfID: normalizedConfig.botSelfID,
  })
  const middleware = async (koa: AlertmanagerKoaContext) => {
    const response = await handleAlertmanagerHTTPRequest({
      authorization: koa.get('authorization'),
      contentType: koa.get('content-type'),
      contentLength: koa.get('content-length'),
      body: koa.req,
    }, normalizedConfig.bearerToken, delivery)
    koa.status = response.status
    if (response.status === 401) {
      koa.set('WWW-Authenticate', 'Bearer')
    }
    koa.body = { success: response.status === 200, code: response.code }
    if (response.status >= 400) {
      logger.warn('alertmanager webhook request failed: category=%s', response.code)
    }
  }
  Object.defineProperty(middleware, RAW_BODY_BYPASS, { value: true })
  server.post(ALERTMANAGER_WEBHOOK_PATH, middleware)
  logger.info('Alertmanager 管理群通知入口已启用：固定路由=%s', ALERTMANAGER_WEBHOOK_PATH)
}

export function validateAlertmanagerWebhookConfig(config: StuhelperAlertmanagerWebhookConfig) {
  const bearerToken = config.bearerToken
  if (
    Buffer.byteLength(bearerToken, 'utf8') < 32 ||
    Buffer.byteLength(bearerToken, 'utf8') > 512 ||
    !TOKEN_PATTERN.test(bearerToken)
  ) {
    throw new Error('alertmanager_webhook_token_invalid')
  }
  const botSelfID = config.botSelfID.trim()
  if (botSelfID && !QQ_ID_PATTERN.test(botSelfID)) {
    throw new Error('alertmanager_webhook_bot_self_id_invalid')
  }
  return { ...config, bearerToken, botSelfID }
}

export async function handleAlertmanagerHTTPRequest(
  request: AlertmanagerHTTPRequest,
  expectedToken: string,
  delivery: AlertmanagerDeliveryService,
): Promise<AlertmanagerHTTPResponse> {
  if (!hasValidBearerToken(request.authorization, expectedToken)) {
    return { status: 401, code: 'unauthorized' }
  }
  if (!isJSONContentType(request.contentType)) {
    return { status: 415, code: 'unsupported_content_type' }
  }

  let body: string
  try {
    body = await readBoundedBody(request.body, request.contentLength)
  } catch (error) {
    if (error instanceof StableWebhookError) {
      return { status: error.status, code: error.category }
    }
    return { status: 400, code: 'invalid_request_body' }
  }

  let payload: unknown
  try {
    payload = JSON.parse(body)
  } catch {
    return { status: 400, code: 'invalid_json' }
  }

  try {
    const result = await delivery.deliver(payload)
    return { status: 200, code: result.deduplicated ? 'deduplicated' : 'delivered' }
  } catch (error) {
    if (error instanceof StableWebhookError) {
      return { status: error.status, code: error.category }
    }
    return { status: 503, code: 'delivery_unavailable' }
  }
}

export class AlertmanagerDeliveryService {
  private readonly delivered = new Map<string, number>()
  private readonly inFlight = new Map<string, Promise<void>>()
  private readonly now: () => number

  constructor(private readonly deps: AlertmanagerDeliveryDeps) {
    this.now = deps.now ?? Date.now
  }

  async deliver(input: unknown) {
    const payload = parseAlertmanagerPayload(input)
    const dedupeKey = createDedupeKey(payload)
    const now = this.now()
    this.pruneDelivered(now)
    if ((this.delivered.get(dedupeKey) ?? 0) > now) {
      return { deduplicated: true }
    }

    const existing = this.inFlight.get(dedupeKey)
    if (existing) {
      await existing
      return { deduplicated: true }
    }

    const delivery = this.deliverOnce(payload)
    this.inFlight.set(dedupeKey, delivery)
    try {
      await delivery
      this.delivered.set(dedupeKey, this.now() + DEDUPE_TTL_MS)
      return { deduplicated: false }
    } finally {
      this.inFlight.delete(dedupeKey)
    }
  }

  private async deliverOnce(payload: AlertmanagerPayload) {
    let targets: readonly AdmissionPolicyTarget[]
    try {
      targets = await this.deps.platform.listAdmissionPolicyTargets()
    } catch {
      throw new StableWebhookError(503, 'backend_unavailable')
    }

    const managementGuildID = resolveUniqueManagementGuildID(targets)
    const bot = resolveAlertDeliveryBot(this.deps.getBots(), this.deps.botSelfID)
    let result: unknown
    try {
      result = await bot.sendMessage(managementGuildID, h.text(formatAlertmanagerMessage(payload)))
    } catch {
      throw new StableWebhookError(503, 'qq_delivery_failed')
    }
    if (!hasMessageID(result)) {
      throw new StableWebhookError(503, 'qq_delivery_failed')
    }
  }

  private pruneDelivered(now: number) {
    for (const [key, expiresAt] of this.delivered) {
      if (expiresAt <= now) {
        this.delivered.delete(key)
      }
    }
  }
}

export function resolveUniqueManagementGuildID(targets: readonly AdmissionPolicyTarget[]) {
  const guildIDs = new Set<string>()
  for (const target of targets) {
    if (target.platform.trim() !== 'qq' || target.guardEnabled !== true) {
      continue
    }
    if (!Array.isArray(target.managementGuildIDs)) {
      throw new StableWebhookError(503, 'management_guild_configuration_invalid')
    }
    for (const rawGuildID of target.managementGuildIDs) {
      const guildID = typeof rawGuildID === 'string' ? rawGuildID.trim() : ''
      if (!QQ_ID_PATTERN.test(guildID)) {
        throw new StableWebhookError(503, 'management_guild_configuration_invalid')
      }
      guildIDs.add(guildID)
    }
  }
  if (guildIDs.size !== 1) {
    throw new StableWebhookError(503, 'management_guild_configuration_invalid')
  }
  return [...guildIDs][0]
}

function formatAlertmanagerMessage(payload: AlertmanagerPayload) {
  const state = payload.status === 'firing' ? 'FIRING' : 'RESOLVED'
  const lines = [
    `[StuHelper 监控告警] ${state}`,
    `告警数量：${payload.alerts.length}`,
  ]
  appendCommonLabel(lines, '级别', payload.commonLabels.severity)
  appendCommonLabel(lines, '服务', payload.commonLabels.service ?? payload.commonLabels.job)

  for (const alert of payload.alerts.slice(0, MAX_MESSAGE_ALERTS)) {
    const alertName = sanitizeMessageText(alert.labels.alertname, 120)
    const severity = sanitizeMessageText(alert.labels.severity ?? payload.commonLabels.severity ?? '', 32)
    const instance = sanitizeMessageText(alert.labels.instance ?? '', 120)
    const summary = sanitizeMessageText(
      alert.annotations.summary ?? payload.commonAnnotations.summary ?? '',
      240,
    )
    const details = [severity && `［${severity}］`, alertName, instance && `· ${instance}`]
      .filter(Boolean)
      .join(' ')
    lines.push(`- ${details}${summary ? `：${summary}` : ''}`)
  }
  if (payload.alerts.length > MAX_MESSAGE_ALERTS) {
    lines.push(`- 另有 ${payload.alerts.length - MAX_MESSAGE_ALERTS} 条告警未展开`)
  }
  return truncateText(lines.join('\n'), MAX_MESSAGE_LENGTH)
}

function parseAlertmanagerPayload(input: unknown): AlertmanagerPayload {
  const root = requireRecord(input)
  const version = requireString(root.version, 8)
  if (version !== '4') {
    throw invalidPayload()
  }
  const status = requireStatus(root.status)
  const alertsInput = root.alerts
  if (!Array.isArray(alertsInput) || alertsInput.length < 1 || alertsInput.length > MAX_ALERTS) {
    throw invalidPayload()
  }
  const alerts = alertsInput.map(parseAlertmanagerAlert)
  if (status === 'resolved' && alerts.some((alert) => alert.status !== 'resolved')) {
    throw invalidPayload()
  }
  if (status === 'firing' && !alerts.some((alert) => alert.status === 'firing')) {
    throw invalidPayload()
  }
  validateOptionalInteger(root.truncatedAlerts)
  validateOptionalString(root.externalURL, 2048)
  return {
    version,
    status,
    groupKey: requireString(root.groupKey, 2048),
    receiver: requireString(root.receiver, 256),
    groupLabels: requireStringRecord(root.groupLabels, 512),
    commonLabels: requireStringRecord(root.commonLabels, 512),
    commonAnnotations: requireStringRecord(root.commonAnnotations, 2048),
    alerts,
  }
}

function parseAlertmanagerAlert(input: unknown): AlertmanagerAlert {
  const alert = requireRecord(input)
  const labels = requireStringRecord(alert.labels, 512)
  if (!labels.alertname) {
    throw invalidPayload()
  }
  validateOptionalString(alert.startsAt, 64)
  validateOptionalString(alert.endsAt, 64)
  validateOptionalString(alert.generatorURL, 2048)
  const fingerprint = requireString(alert.fingerprint, 128)
  if (!/^[A-Fa-f0-9]{8,128}$/.test(fingerprint)) {
    throw invalidPayload()
  }
  return {
    status: requireStatus(alert.status),
    labels,
    annotations: requireStringRecord(alert.annotations, 2048),
    fingerprint,
  }
}

function requireRecord(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    throw invalidPayload()
  }
  const prototype = Object.getPrototypeOf(value)
  if (prototype !== Object.prototype && prototype !== null) {
    throw invalidPayload()
  }
  return value as Record<string, unknown>
}

function requireStringRecord(value: unknown, maxValueLength: number) {
  const record = requireRecord(value)
  const entries = Object.entries(record)
  if (entries.length > MAX_RECORD_ENTRIES) {
    throw invalidPayload()
  }
  const output: Record<string, string> = Object.create(null)
  for (const [key, item] of entries) {
    if (!LABEL_NAME_PATTERN.test(key) || typeof item !== 'string' || item.length > maxValueLength) {
      throw invalidPayload()
    }
    output[key] = item
  }
  return output
}

function requireString(value: unknown, maxLength: number) {
  if (typeof value !== 'string' || value.length < 1 || value.length > maxLength) {
    throw invalidPayload()
  }
  return value
}

function validateOptionalString(value: unknown, maxLength: number) {
  if (value === undefined) {
    return
  }
  if (typeof value !== 'string' || value.length > maxLength) {
    throw invalidPayload()
  }
}

function validateOptionalInteger(value: unknown) {
  if (value === undefined) {
    return
  }
  if (!Number.isInteger(value) || (value as number) < 0) {
    throw invalidPayload()
  }
}

function requireStatus(value: unknown): AlertStatus {
  if (value !== 'firing' && value !== 'resolved') {
    throw invalidPayload()
  }
  return value
}

function resolveAlertDeliveryBot(bots: readonly AlertDeliveryBot[], configuredBotSelfID?: string) {
  const compatible = bots.filter((bot) => resolveAdmissionSubjectPlatform(bot.platform) === 'qq')
  const selected = configuredBotSelfID
    ? compatible.filter((bot) => bot.selfId === configuredBotSelfID)
    : compatible
  if (selected.length !== 1) {
    throw new StableWebhookError(503, 'qq_bot_unavailable')
  }
  return selected[0]
}

function createDedupeKey(payload: AlertmanagerPayload) {
  const alerts = payload.alerts
    .map((alert) => `${alert.fingerprint}:${alert.status}`)
    .sort()
  return createHash('sha256')
    .update(JSON.stringify([payload.version, payload.status, payload.groupKey, alerts]))
    .digest('hex')
}

function hasMessageID(result: unknown) {
  if (typeof result === 'string') {
    return result.trim().length > 0
  }
  return Array.isArray(result) && result.some((item) => typeof item === 'string' && item.trim().length > 0)
}

function hasValidBearerToken(header: string, expectedToken: string) {
  const match = /^Bearer ([^\s]+)$/.exec(header)
  const supplied = match?.[1] ?? ''
  const expectedDigest = createHash('sha256').update(expectedToken).digest()
  const suppliedDigest = createHash('sha256').update(supplied).digest()
  return timingSafeEqual(expectedDigest, suppliedDigest) && Boolean(match)
}

function isJSONContentType(value: string) {
  return /^application\/json(?:\s*;\s*charset=[A-Za-z0-9._-]+)?$/i.test(value.trim())
}

async function readBoundedBody(body: AsyncIterable<Uint8Array>, contentLength: string) {
  if (contentLength) {
    if (!/^[0-9]+$/.test(contentLength)) {
      throw new StableWebhookError(400, 'invalid_content_length')
    }
    if (Number(contentLength) > MAX_BODY_BYTES) {
      throw new StableWebhookError(413, 'payload_too_large')
    }
  }
  const chunks: Buffer[] = []
  let total = 0
  let tooLarge = false
  for await (const chunk of body) {
    total += chunk.byteLength
    if (total > MAX_BODY_BYTES) {
      tooLarge = true
      continue
    }
    chunks.push(Buffer.from(chunk))
  }
  if (tooLarge) {
    throw new StableWebhookError(413, 'payload_too_large')
  }
  if (total === 0) {
    throw new StableWebhookError(400, 'invalid_request_body')
  }
  return Buffer.concat(chunks, total).toString('utf8')
}

function sanitizeMessageText(value: string, maxLength: number) {
  const sanitized = value
    .normalize('NFKC')
    .replace(/[\u0000-\u001f\u007f-\u009f]/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
    .replace(/&/g, '＆')
    .replace(/</g, '＜')
    .replace(/>/g, '＞')
    .replace(/@/g, '＠')
    .replace(/\[/g, '［')
    .replace(/\]/g, '］')
  return truncateText(sanitized, maxLength)
}

function truncateText(value: string, maxLength: number) {
  const characters = [...value]
  if (characters.length <= maxLength) {
    return value
  }
  return `${characters.slice(0, Math.max(0, maxLength - 1)).join('')}…`
}

function appendCommonLabel(lines: string[], title: string, value: string | undefined) {
  if (value) {
    lines.push(`${title}：${sanitizeMessageText(value, 120)}`)
  }
}

function invalidPayload() {
  return new StableWebhookError(400, 'invalid_alertmanager_payload')
}

class StableWebhookError extends Error {
  constructor(
    readonly status: number,
    readonly category: string,
  ) {
    super(category)
  }
}
