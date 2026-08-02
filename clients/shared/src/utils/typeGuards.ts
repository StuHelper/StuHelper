/**
 * 将未知值收窄为普通 JSON object；数组必须由调用方显式按数组处理。
 */
export function isNonArrayRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value)
}
