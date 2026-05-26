import type { Reply } from '@stuhelper/shared/reply'

export function readReplyPage(payload: unknown): { list: Reply[]; total: number } {
  if (!payload || typeof payload !== 'object') {
    throw new Error('Invalid reply list response')
  }

  const { list, total } = payload as { list?: unknown; total?: unknown }
  if (!Array.isArray(list) || typeof total !== 'number' || !Number.isFinite(total) || total < 0) {
    throw new Error('Invalid reply list response')
  }

  return { list: list as Reply[], total }
}

export function readReply(payload: unknown): Reply {
  if (!payload || typeof payload !== 'object') {
    throw new Error('Invalid reply response')
  }

  return payload as Reply
}
