export function readArrayPayload<T>(payload: unknown, message: string): T[] {
  if (!Array.isArray(payload)) {
    throw new Error(message)
  }

  return payload as T[]
}

export function readListPayload<T>(payload: unknown, message: string): T[] {
  if (!payload || typeof payload !== 'object') {
    throw new Error(message)
  }

  const { list } = payload as { list?: unknown }
  if (!Array.isArray(list)) {
    throw new Error(message)
  }

  return list as T[]
}

export function readPaginatedPayload<T>(payload: unknown, message: string): {
  list: T[]
  total: number
} {
  if (!payload || typeof payload !== 'object') {
    throw new Error(message)
  }

  const { list, total } = payload as { list?: unknown; total?: unknown }
  if (!Array.isArray(list) || typeof total !== 'number' || !Number.isFinite(total) || total < 0) {
    throw new Error(message)
  }

  return { list: list as T[], total }
}
