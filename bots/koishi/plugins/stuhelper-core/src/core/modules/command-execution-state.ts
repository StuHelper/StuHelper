import type { Session } from 'koishi'

export interface CommandExecutionState {
  readonly failed: boolean
  readonly error?: string
}

const startedAtByArgv = new WeakMap<object, number>()
const stateBySession = new WeakMap<Session, CommandExecutionState>()

export function markCommandExecutionStarted(argv: unknown, startedAt = Date.now()): void {
  if (isObject(argv)) {
    startedAtByArgv.set(argv, startedAt)
  }
}

export function commandExecutionDuration(argv: unknown, now = Date.now()): number {
  if (!isObject(argv)) return 0

  const startedAt = startedAtByArgv.get(argv)
  if (startedAt === undefined) return 0

  return Math.max(0, now - startedAt)
}

export function markCommandExecutionFailed(session: Session, error?: string): void {
  stateBySession.set(session, { failed: true, error })
}

export function getCommandExecutionState(session: Session): CommandExecutionState {
  return stateBySession.get(session) ?? { failed: false }
}

export function clearCommandExecutionState(session: Session): void {
  stateBySession.delete(session)
}

function isObject(value: unknown): value is object {
  return typeof value === 'object' && value !== null
}
