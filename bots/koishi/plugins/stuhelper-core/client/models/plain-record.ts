export type PlainRecord = Record<string, unknown>

export function deepMerge<T extends PlainRecord>(target: T, source: unknown): T {
  if (!isPlainRecord(source)) {
    return target
  }

  for (const [key, value] of Object.entries(source)) {
    // Keep these comparisons explicit so both runtime review and static
    // analysis can prove that built-in prototype paths are unreachable.
    if (key === '__proto__' || key === 'constructor' || key === 'prototype') {
      continue
    }

    const targetOwnsKey = Object.prototype.hasOwnProperty.call(target, key)
    const current = targetOwnsKey ? target[key] : undefined
    if (targetOwnsKey && isPlainRecord(current) && isPlainRecord(value)) {
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
