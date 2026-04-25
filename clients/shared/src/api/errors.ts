/**
 * 统一 API 错误 envelope 解析。
 *
 * 后端响应格式：{ success, data, error: { code, message, details }, message, code }
 * 本模块提供平台无关的解析逻辑，各端 transport 层只需调用 parseApiError。
 */

/** 从 error envelope 中提取出的结构化错误信息 */
export interface ParsedApiError {
  code?: string
  message?: string
  details?: Record<string, unknown>
}

export function isRecord(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === 'object'
}

/**
 * 从 API 响应 payload 中解析错误信息。
 * 兼容两种后端格式：
 *   1. 嵌套：{ error: { code, message, details } }
 *   2. 扁平：{ code, message }
 */
export function parseApiError(payload: unknown): ParsedApiError {
  if (!isRecord(payload)) {
    return {}
  }

  const record = payload

  // 优先读取嵌套 error 对象
  const rawError = record.error
  if (isRecord(rawError)) {
    const errorObj = rawError
    const result: ParsedApiError = {}
    if (typeof errorObj.message === 'string' && errorObj.message) {
      result.message = errorObj.message
    }
    if (typeof errorObj.code === 'string' && errorObj.code) {
      result.code = errorObj.code
    }
    if (isRecord(errorObj.details)) {
      result.details = errorObj.details
    }
    if (result.message || result.code) {
      return result
    }
  }

  // error 为字符串
  if (typeof rawError === 'string' && rawError.trim()) {
    return { message: rawError.trim() }
  }

  // 扁平格式
  const result: ParsedApiError = {}
  if (typeof record.message === 'string' && record.message.trim()) {
    result.message = record.message.trim()
  }
  if (typeof record.code === 'string' && record.code.trim()) {
    result.code = record.code.trim()
  }
  return result
}

/**
 * 从 API 响应 payload 中提取用户可读的错误消息。
 * 优先返回 error.message，其次 payload.message，最后返回 fallback。
 */
export function extractApiErrorMessage(payload: unknown, fallback: string): string {
  const parsed = parseApiError(payload)
  return parsed.message || parsed.code || fallback
}
