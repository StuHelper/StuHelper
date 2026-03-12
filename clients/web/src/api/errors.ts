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

// 后端未返回 code 时的 HTTP 状态码兜底映射
export function httpStatusToDefaultCode(status: number): string {
  const map: Record<number, string> = {
    400: 'A0000400',
    401: 'A0010100',
    403: 'A0010200',
    404: 'A0000404',
    409: 'A0000409',
    422: 'A0000422',
    429: 'A0000429',
    500: 'B0000001',
    502: 'C0000001',
    503: 'B0000004',
    504: 'B0000006',
  }
  return map[status] || 'B0000001'
}
