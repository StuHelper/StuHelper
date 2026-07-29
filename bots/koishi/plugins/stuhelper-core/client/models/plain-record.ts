export type PlainRecord = Record<string, unknown>

const FORBIDDEN_MERGE_KEYS = new Set(['__proto__', 'constructor', 'prototype'])

export function deepMerge<T extends PlainRecord>(target: T, source: unknown): T {
  if (!isPlainRecord(source)) {
    return target
  }

  for (const [key, value] of Object.entries(source)) {
    if (FORBIDDEN_MERGE_KEYS.has(key)) {
      continue
    }

    const current = target[key]
    if (isPlainRecord(current) && isPlainRecord(value)) {
      deepMerge(current, value)
    } else {
      target[key] = value
    }
  }

  return target
}

export function isPlainRecord(value: unknown): value is PlainRecord {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) {
    return false
  }
  const prototype = Object.getPrototypeOf(value)
  return prototype === Object.prototype || prototype === null
}
