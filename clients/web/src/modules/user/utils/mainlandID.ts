const CHECKSUM_WEIGHTS = [7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2] as const
const CHECKSUM_CHARS = '10X98765432'

export function normalizeMainlandIDNumber(value: string): string {
  return value.trim().toUpperCase()
}

export function isValidMainlandIDNumber(value: string): boolean {
  const id = normalizeMainlandIDNumber(value)
  if (!/^[1-9]\d{16}[\dX]$/.test(id)) {
    return false
  }

  const birth = id.slice(6, 14)
  if (!isValidBirthDate(birth)) {
    return false
  }

  let sum = 0
  for (let i = 0; i < CHECKSUM_WEIGHTS.length; i += 1) {
    sum += Number(id[i]) * CHECKSUM_WEIGHTS[i]
  }
  return CHECKSUM_CHARS[sum % 11] === id[17]
}

function isValidBirthDate(value: string): boolean {
  const year = Number(value.slice(0, 4))
  const month = Number(value.slice(4, 6))
  const day = Number(value.slice(6, 8))
  const date = new Date(Date.UTC(year, month - 1, day))
  return (
    date.getUTCFullYear() === year &&
    date.getUTCMonth() === month - 1 &&
    date.getUTCDate() === day
  )
}
