import type {
  StuhelperKeywordRuleAction,
  StuhelperKeywordRuleMatchMode,
} from '@stuhelper/koishi-shared'

const MIN_POSITIVE_INTEGER = 1

type KeywordEnumValue = StuhelperKeywordRuleAction | StuhelperKeywordRuleMatchMode

export function readRecord(value: unknown, field: string): Record<string, unknown> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    throw new Error(`${field} must be an object`)
  }
  return value as Record<string, unknown>
}

export function readOptionalRecord(
  value: unknown,
  defaultValue: object,
): Record<string, unknown> {
  if (value === undefined) return structuredClone(defaultValue) as Record<string, unknown>
  return readRecord(value, 'nested config')
}

export function rejectUnknownFields(
  record: Record<string, unknown>,
  fields: readonly string[],
): void {
  const allowed = new Set(fields)
  for (const field of Object.keys(record)) {
    if (!allowed.has(field)) throw new Error(`unknown group guard config field: ${field}`)
  }
}

export function readString(value: unknown, field: string, defaultValue: string): string {
  if (value === undefined) return defaultValue
  return readRequiredString(value, field)
}

export function readText(value: unknown, field: string, defaultValue: string): string {
  if (value === undefined) return defaultValue
  if (typeof value !== 'string') throw new Error(`${field} must be a string`)
  return value
}

export function readRequiredString(value: unknown, field: string): string {
  if (typeof value !== 'string' || !value.trim()) {
    throw new Error(`${field} must be a non-empty string`)
  }
  return value
}

export function readStringArray(value: unknown, field: string): string[] {
  if (value === undefined) return []
  if (!Array.isArray(value)) throw new Error(`${field} must be a string array`)
  return value.map((item) => readRequiredString(item, `${field} item`))
}

export function readBoolean(value: unknown, field: string, defaultValue: boolean): boolean {
  if (value === undefined) return defaultValue
  if (typeof value !== 'boolean') throw new Error(`${field} must be a boolean`)
  return value
}

export function readPositiveInteger(input: {
  readonly value: unknown
  readonly field: string
  readonly defaultValue: number
  readonly minimum?: number
}): number {
  if (input.value === undefined) return input.defaultValue
  const minimum = input.minimum ?? MIN_POSITIVE_INTEGER
  if (typeof input.value !== 'number'
    || !Number.isInteger(input.value)
    || input.value < minimum) {
    throw new Error(`${input.field} must be a positive integer`)
  }
  return input.value
}

export function readNonNegativeInteger(
  value: unknown,
  field: string,
  defaultValue: number,
): number {
  if (value === undefined) return defaultValue
  if (typeof value !== 'number' || !Number.isInteger(value) || value < 0) {
    throw new Error(`${field} must be a non-negative integer`)
  }
  return value
}

export function readEnum<TValue extends KeywordEnumValue>(input: {
  readonly value: unknown
  readonly field: string
  readonly allowed: readonly TValue[]
  readonly defaultValue: TValue
}): TValue {
  if (input.value === undefined) return input.defaultValue
  if (typeof input.value !== 'string' || !input.allowed.includes(input.value as TValue)) {
    throw new Error(`${input.field} must be one of ${input.allowed.join(', ')}`)
  }
  return input.value as TValue
}
