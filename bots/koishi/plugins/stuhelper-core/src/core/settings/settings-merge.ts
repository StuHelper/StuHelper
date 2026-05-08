export function deepMerge<T extends Record<string, any>>(defaults: T, overrides: Partial<T>): T {
  const result = { ...defaults }

  for (const key of Object.keys(overrides) as Array<keyof T>) {
    const value = overrides[key]
    if (value === undefined) continue
    result[key] = shouldMergeNested(defaults[key], value)
      ? deepMerge(defaults[key] as any, value as any)
      : value as T[keyof T]
  }

  return result
}

export function getDiff<T extends Record<string, any>>(defaults: T, current: T): Partial<T> {
  const diff: Partial<T> = {}

  for (const key of Object.keys(current) as Array<keyof T>) {
    const defaultValue = defaults[key]
    const currentValue = current[key]
    const nestedDiff = readNestedDiff(defaultValue, currentValue)

    if (nestedDiff !== undefined) {
      diff[key] = nestedDiff as T[keyof T]
    }
  }

  return diff
}

function readNestedDiff(defaultValue: unknown, currentValue: unknown) {
  if (isPlainObject(currentValue)) {
    if (!isPlainObject(defaultValue)) return currentValue
    const nestedDiff = getDiff(defaultValue as Record<string, any>, currentValue as Record<string, any>)
    return Object.keys(nestedDiff).length > 0 ? nestedDiff : undefined
  }
  if (Array.isArray(currentValue)) {
    return arraysEqual(currentValue, defaultValue as any[]) ? undefined : currentValue
  }
  return currentValue !== defaultValue ? currentValue : undefined
}

function shouldMergeNested(defaultValue: unknown, value: unknown) {
  return isPlainObject(value) && isPlainObject(defaultValue)
}

function isPlainObject(value: unknown): value is Record<string, any> {
  return Boolean(value && typeof value === 'object' && !Array.isArray(value))
}

function arraysEqual(a: any[], b: any[]): boolean {
  if (!Array.isArray(b)) return false
  if (a.length !== b.length) return false
  return a.every((value, index) => value === b[index])
}
