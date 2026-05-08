import { parseUserId } from '../../utils'

export interface KickInput {
  readonly userId: string | null
  readonly targetGroup?: string
  readonly black: boolean
}

export interface KickOptions {
  readonly black?: boolean
}

export function parseKickInput(input: string, defaultGuildId?: string, options: KickOptions = {}): KickInput {
  const black = Boolean(options.black) || input.includes('-b')
  const normalized = input.replace(/-b/g, '').replace(/\s+/g, ' ').trim()
  const [target, groupId] = splitKickArgs(normalized)

  return {
    userId: resolveTargetUserId(target),
    targetGroup: groupId || defaultGuildId,
    black,
  }
}

export function resolveCommandUserId(user: unknown): string {
  const raw = String(user || '').trim()
  if (!raw) return ''
  const [, platformUserId] = raw.split(':')
  return platformUserId || resolveTargetUserId(raw) || ''
}

function splitKickArgs(input: string): string[] {
  const source = input.trim()
  if (!source.includes('<at')) return source.split(/\s+/).filter(Boolean)

  const atMatch = source.match(/<at[^>]+>/)
  if (!atMatch) return source.split(/\s+/).filter(Boolean)

  const atPart = atMatch[0]
  const restPart = source.replace(atPart, '').trim()
  return [atPart, ...restPart.split(/\s+/)].filter(Boolean)
}

function resolveTargetUserId(target: string): string | null {
  try {
    if (target?.startsWith('<at')) {
      const match = target.match(/id="(\d+)"/)
      if (match) return match[1]
    } else {
      return parseUserId(target)
    }
  } catch {
    return parseUserId(target)
  }
  return null
}
