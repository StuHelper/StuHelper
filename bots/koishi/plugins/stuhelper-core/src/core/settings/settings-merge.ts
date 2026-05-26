type PlainRecord = Record<string, unknown>
type StringKeyOf<T> = Extract<keyof T, string>
type DeepPartial<T> = {
  [K in keyof T]?: T[K] extends readonly unknown[]
    ? T[K]
    : T[K] extends object
      ? DeepPartial<T[K]>
      : T[K]
}

export function deepMerge<T extends object>(defaults: T, overrides: DeepPartial<T>): T {
  const result = { ...defaults }

  for (const key of Object.keys(overrides) as Array<StringKeyOf<T>>) {
    const value = overrides[key]
    if (value === undefined) continue
    result[key] = shouldMergeNested(defaults[key], value)
      ? deepMerge(defaults[key] as PlainRecord, value as DeepPartial<PlainRecord>) as T[typeof key]
      : value as T[typeof key]
  }

  return result
}

export function getDiff<T extends object>(defaults: T, current: T): Partial<T> {
  const diff: Partial<T> = {}

  for (const key of Object.keys(current) as Array<StringKeyOf<T>>) {
    const defaultValue = defaults[key]
    const currentValue = current[key]
    const nestedDiff = readNestedDiff(defaultValue, currentValue)

    if (nestedDiff !== undefined) {
      diff[key] = nestedDiff as T[typeof key]
    }
  }

  return diff
}

function readNestedDiff(defaultValue: unknown, currentValue: unknown) {
  if (isPlainObject(currentValue)) {
    if (!isPlainObject(defaultValue)) return currentValue
    const nestedDiff = getDiff(defaultValue, currentValue)
    return Object.keys(nestedDiff).length > 0 ? nestedDiff : undefined
  }
  if (Array.isArray(currentValue)) {
    return arraysEqual(currentValue, defaultValue) ? undefined : currentValue
  }
  return currentValue !== defaultValue ? currentValue : undefined
}

function shouldMergeNested(defaultValue: unknown, value: unknown) {
  return isPlainObject(value) && isPlainObject(defaultValue)
}

function isPlainObject(value: unknown): value is PlainRecord {
  return Boolean(value && typeof value === 'object' && !Array.isArray(value))
}

function arraysEqual(a: readonly unknown[], b: unknown): boolean {
  if (!Array.isArray(b)) return false
  if (a.length !== b.length) return false
  return a.every((value, index) => value === b[index])
}
