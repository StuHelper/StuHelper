/**
 * API 错误类型定义
 * 统一的错误处理机制 — 错误码即后端 8 位码或客户端专属码
 *
 * 错误码值域:
 *   - API 错误: 后端 8 位结构化码 (A0110001, B0000001 等)
 *   - 客户端专属: NETWORK_ERROR, OFFLINE, TIMEOUT
 *
 * 行为判断通过前缀:
 *   - A001xxxx = 认证授权模块
 *   - Bxxxx = 系统错误（可重试）
 *   - Cxxxx = 第三方服务错误（可重试）
 * 详见 docs/reference/error-codes.md
 */
import i18n from '@/i18n'
import { defaultHttpStatusErrorCode } from '@stuhelper/shared/api'

// 错误严重级别
export type ErrorSeverity = 'info' | 'warning' | 'error' | 'critical'

// API 错误类
export class ApiError extends Error {
  readonly code: string
  readonly status?: number
  readonly severity: ErrorSeverity
  readonly details?: Record<string, unknown>
  readonly timestamp: Date
  readonly requestID?: string

  constructor(options: {
    message: string
    code: string
    status?: number
    severity?: ErrorSeverity
    details?: Record<string, unknown>
    requestID?: string
  }) {
    super(options.message)
    Object.setPrototypeOf(this, ApiError.prototype)
    this.name = 'ApiError'
    this.code = options.code
    this.status = options.status
    this.severity = options.severity ?? 'error'
    this.details = options.details
    this.timestamp = new Date()
    this.requestID = options.requestID
  }

  // 获取用户友好的错误消息（单层 i18n 查找）
  getUserMessage(): string {
    const { t, te } = i18n.global
    const key = `errors.${this.code}`
    return te(key) ? t(key) : this.message
  }

  toJSON() {
    return {
      name: this.name,
      message: this.message,
      code: this.code,
      status: this.status,
      severity: this.severity,
      details: this.details,
      timestamp: this.timestamp.toISOString(),
      requestID: this.requestID
    }
  }
}

// ---- 行为判断工具函数（基于错误码前缀） ----

// 认证相关错误 (A001xxxx)
export function isAuthError(code: string): boolean {
  return code.startsWith('A001')
}

// CSRF 错误 (A0010202 = invalid, A0010203 = missing)
// CSRF 403 意味着请求被中间件拦截，但服务端会话仍然有效
export function isCsrfError(code: string): boolean {
  return code === 'A0010202' || code === 'A0010203'
}

// 网络/客户端错误（后端不会返回）
export function isNetworkError(code: string): boolean {
  return code === 'NETWORK_ERROR' || code === 'OFFLINE' || code === 'TIMEOUT'
}

// 可重试错误（系统错误、第三方服务错误、网络错误）
export function isRetryable(code: string): boolean {
  return code.startsWith('B') || code.startsWith('C') || isNetworkError(code)
}

// 类型守卫
export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError
}

// 从未知错误对象中提取 HTTP 状态码
export function getErrorStatus(error: unknown): number | undefined {
  if (error instanceof ApiError) return error.status
  if (typeof error === 'object' && error !== null && 'status' in error) {
    const status = (error as { status?: unknown }).status
    return typeof status === 'number' ? status : undefined
  }
  return undefined
}

// 后端未返回 code 时的 HTTP 状态码兜底映射
export function httpStatusToDefaultCode(status: number): string {
  return defaultHttpStatusErrorCode(status)
}

export function getErrorMessage(error: unknown, fallback: string): string {
  if (error instanceof ApiError) {
    const { t, te } = i18n.global
    const key = `errors.${error.code}`

    if (te(key)) {
      const localized = t(key).trim()
      return localized || fallback
    }

    return fallback
  }
  return fallback
}

export function classifyApiError<TType extends string>(
  error: unknown,
  options: {
    networkType: TType
    apiType: TType
    unknownType: TType
    networkMessage?: string
    fallbackMessage: string
  },
): { type: TType, message: string } {
  if (error instanceof ApiError) {
    if (isNetworkError(error.code)) {
      return {
        type: options.networkType,
        message: options.networkMessage ?? i18n.global.t('errors.NETWORK_ERROR'),
      }
    }
    return {
      type: options.apiType,
      message: error.getUserMessage(),
    }
  }
  if (error instanceof Error) {
    return {
      type: options.unknownType,
      message: error.message,
    }
  }
  return {
    type: options.unknownType,
    message: options.fallbackMessage,
  }
}
