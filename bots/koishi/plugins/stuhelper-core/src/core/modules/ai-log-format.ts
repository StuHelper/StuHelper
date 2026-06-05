export function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

export function unknownErrorMessage(error: unknown): string {
  if (error instanceof Error) return error.message
  if (typeof error === 'string') return error
  return toLogSnippet(error)
}

export function toLogSnippet(value: unknown, maxLength = 1000): string {
  const text = stringifyUnknown(value)
  if (text.length <= maxLength) return text
  return `${text.slice(0, maxLength)}...`
}

function stringifyUnknown(value: unknown): string {
  if (typeof value === 'string') return value

  try {
    const json = JSON.stringify(value)
    return json ?? String(value)
  } catch (error) {
    const reason = error instanceof Error ? error.message : String(error)
    return `[unserializable value: ${reason}]`
  }
}
