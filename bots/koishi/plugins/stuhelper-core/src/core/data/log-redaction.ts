import type { CommandLogRecord } from './command-log-records'

const REDACTED = '[REDACTED]'
const MAX_REDACTION_DEPTH = 12
const SENSITIVE_KEY_PATTERN = /^(?:authorization|cookie|set-cookie|setCookie|x-api-key|xApiKey|api[-_]?key|apiKey|access[-_]?token|accessToken|refresh[-_]?token|refreshToken|service[-_]?token|serviceToken|token|secret|password|credential)$/i
const SENSITIVE_ARG_PATTERN = /^--?(?:authorization|cookie|set-cookie|x-api-key|api[-_]?key|access[-_]?token|refresh[-_]?token|service[-_]?token|token|secret|password|credential)$/i

export function redactCommandLogRecord(log: CommandLogRecord): CommandLogRecord {
  return {
    ...log,
    args: redactCommandArgs(Array.isArray(log.args) ? log.args : []),
    options: isRecord(log.options) ? redactSensitiveValue(log.options) as Record<string, unknown> : {},
    error: log.error ? redactSensitiveText(log.error) : undefined,
    result: log.result ? redactSensitiveText(log.result) : undefined,
  }
}

export function redactCommandLogRecords(logs: readonly CommandLogRecord[]): CommandLogRecord[] {
  return logs.map(redactCommandLogRecord)
}

export function redactSensitiveValue(value: unknown): unknown {
  return redactValue(value, new WeakSet<object>(), 0)
}

export function redactSensitiveText(value: string): string {
  return value
    .replace(/\b(authorization\s*[=:]\s*)(?:Bearer|Basic)\s+[A-Za-z0-9._~+/=-]+/gi, `$1${REDACTED}`)
    .replace(/\bBearer\s+[A-Za-z0-9._~+/=-]+/gi, `Bearer ${REDACTED}`)
    .replace(/\b(Authorization)\s*:\s*[^\n\r,;]+/gi, `$1: ${REDACTED}`)
    .replace(/\b(Cookie|Set-Cookie)\s*:\s*[^\n\r]+/gi, `$1: ${REDACTED}`)
    .replace(/([?&](?:authorization|cookie|set-cookie|x-api-key|api[-_]?key|access[-_]?token|refresh[-_]?token|service[-_]?token|token|secret|password|credential)=)[^&\s]+/gi, `$1${REDACTED}`)
    .replace(/\b((?:authorization|cookie|set-cookie|x-api-key|api[-_]?key|access[-_]?token|refresh[-_]?token|service[-_]?token|token|secret|password|credential)\s*[=:]\s*)[^\s,;&"']+/gi, `$1${REDACTED}`)
    .replace(/("?(?:authorization|cookie|set-cookie|x-api-key|apiKey|api_key|access_token|refresh_token|serviceToken|service_token|token|secret|password|credential)"?\s*:\s*")([^"]+)(")/gi, `$1${REDACTED}$3`)
    .replace(/\bsk-[A-Za-z0-9_-]{5,}/g, 'sk-[REDACTED]')
}

export function redactCommandArgs(args: readonly string[]): string[] {
  const redacted: string[] = []
  let redactNext = false
  for (const arg of args) {
    if (redactNext) {
      redacted.push(REDACTED)
      redactNext = false
      continue
    }

    const normalized = String(arg)
    redacted.push(redactSensitiveText(normalized))
    if (SENSITIVE_ARG_PATTERN.test(normalized)) {
      redactNext = true
    }
  }
  return redacted
}

function redactValue(value: unknown, seen: WeakSet<object>, depth: number): unknown {
  if (typeof value === 'string') return redactSensitiveText(value)
  if (typeof value !== 'object' || value === null) return value
  if (depth >= MAX_REDACTION_DEPTH) return '[MaxDepth]'
  if (seen.has(value)) return '[Circular]'
  seen.add(value)

  if (Array.isArray(value)) {
    return value.map((item) => redactValue(item, seen, depth + 1))
  }

  const result: Record<string, unknown> = {}
  for (const [key, entry] of Object.entries(value as Record<string, unknown>)) {
    result[key] = SENSITIVE_KEY_PATTERN.test(key)
      ? REDACTED
      : redactValue(entry, seen, depth + 1)
  }
  return result
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}
