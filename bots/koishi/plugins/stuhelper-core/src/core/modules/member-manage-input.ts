import { parseUserId } from '../../utils'

export interface KickInput {
  readonly userId: string | null
  readonly targetGroup?: string
  readonly black: boolean
  readonly global: boolean
}

export interface KickOptions {
  readonly black?: boolean
  readonly global?: boolean
}

export function parseKickInput(input: string, defaultGuildId?: string, options: KickOptions = {}): KickInput {
  const args = splitTargetArgs(input)
  const black = Boolean(options.black) || args.includes('-b')
  const global = Boolean(options.global) || args.includes('-g') || args.includes('--global')
  const [target, groupId] = args.filter(isKickPositionalArg)

  return {
    userId: resolveTargetUserId(target),
    targetGroup: groupId || defaultGuildId,
    black,
    global,
  }
}

export function resolveCommandUserId(user: unknown): string {
  if (!user) return ''
  if (typeof user !== 'string') return parseUserId(user)

  const raw = user.trim()
  if (!raw) return ''
  const [, platformUserId] = raw.split(':')
  return platformUserId || resolveTargetUserId(raw) || ''
}

export function splitTargetArgs(input: string | undefined): string[] {
  const source = (input || '').trim()
  if (!source.includes('<at')) return source.split(/\s+/).filter(Boolean)

  const atMatch = source.match(/<at[^>]+>/)
  if (!atMatch) return source.split(/\s+/).filter(Boolean)

  const atPart = atMatch[0]
  const restPart = source.replace(atPart, '').trim()
  return [atPart, ...restPart.split(/\s+/)].filter(Boolean)
}

export function resolveTargetUserId(target?: string): string | null {
  if (!target) return null
  try {
    if (target?.startsWith('<at')) {
      const match = target.match(/id="(\d+)"/)
      if (match) return match[1]
    } else {
      return parseUserId(target)
    }
  } catch {
    return parseUserId(target || '')
  }
  return null
}

function isKickPositionalArg(value: string): boolean {
  return value !== '-b' && value !== '-g' && value !== '--global'
}
