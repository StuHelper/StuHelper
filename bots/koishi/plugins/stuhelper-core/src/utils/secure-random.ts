import { randomInt } from 'node:crypto'

const RANDOM_UNIT_SCALE = 0x1_0000_0000

/**
 * Return a cryptographically strong value in the half-open interval [0, 1).
 */
export function secureRandomUnit(): number {
  return randomInt(RANDOM_UNIT_SCALE) / RANDOM_UNIT_SCALE
}

/**
 * Return a cryptographically strong integer in [minInclusive, maxExclusive).
 *
 * Equal bounds intentionally return the sole valid value. This keeps callers
 * deterministic when a configured duration range collapses to one value.
 */
export function secureRandomInt(minInclusive: number, maxExclusive: number): number {
  if (!Number.isSafeInteger(minInclusive) || !Number.isSafeInteger(maxExclusive)) {
    throw new RangeError('secure random bounds must be safe integers')
  }
  if (maxExclusive < minInclusive) {
    throw new RangeError('secure random upper bound must not be below lower bound')
  }
  if (maxExclusive === minInclusive) {
    return minInclusive
  }
  return randomInt(minInclusive, maxExclusive)
}
