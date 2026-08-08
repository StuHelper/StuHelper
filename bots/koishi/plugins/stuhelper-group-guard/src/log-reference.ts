import { createHmac, randomBytes } from 'node:crypto'

// References remain correlatable only for the lifetime of one bot process.
// This keeps raw platform identifiers out of logs and avoids creating a stable,
// brute-forceable pseudonymous user identifier across restarts.
const processLogSalt = randomBytes(32)

export function opaqueLogReference(kind: string, value: string | null | undefined): string | undefined {
  const normalized = value?.trim()
  if (!normalized) return undefined
  const digest = createHmac('sha256', processLogSalt)
    .update(kind)
    .update('\0')
    .update(normalized)
    .digest('hex')
    .slice(0, 16)
  return `${kind}_${digest}`
}
