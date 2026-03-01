/**
 * API 错误类型定义
 * 统一的错误处理机制
 */
import i18n from '@/i18n'

// 错误代码枚举
export enum ErrorCode {
  // 网络错误
  NETWORK_ERROR = 'NETWORK_ERROR',
  OFFLINE = 'OFFLINE',
  TIMEOUT = 'TIMEOUT',

  // 认证错误
  UNAUTHORIZED = 'UNAUTHORIZED',
  TOKEN_EXPIRED = 'TOKEN_EXPIRED',
  INVALID_TOKEN = 'INVALID_TOKEN',

  // 权限错误
  FORBIDDEN = 'FORBIDDEN',

  // 客户端错误
  BAD_REQUEST = 'BAD_REQUEST',
  NOT_FOUND = 'NOT_FOUND',
  VALIDATION_ERROR = 'VALIDATION_ERROR',
  CONFLICT = 'CONFLICT',
  RATE_LIMIT_EXCEEDED = 'RATE_LIMIT_EXCEEDED',

  // 服务端错误
  SERVER_ERROR = 'SERVER_ERROR',
  SERVICE_UNAVAILABLE = 'SERVICE_UNAVAILABLE',

  // 业务错误
  BUSINESS_ERROR = 'BUSINESS_ERROR',

  // 未知错误
  UNKNOWN = 'UNKNOWN'
}

// 错误严重级别
export type ErrorSeverity = 'info' | 'warning' | 'error' | 'critical'

// API 错误类
export class ApiError extends Error {
  readonly code: ErrorCode
  readonly status?: number
  readonly severity: ErrorSeverity
  readonly details?: Record<string, unknown>
  readonly timestamp: Date
  readonly requestID?: string

  constructor(options: {
    message: string
    code: ErrorCode
    status?: number
    severity?: ErrorSeverity
    details?: Record<string, unknown>
    requestID?: string
  }) {
    super(options.message)
    // 修正原型链，确保 instanceof 检查在所有环境下可靠
    Object.setPrototypeOf(this, ApiError.prototype)
    this.name = 'ApiError'
    this.code = options.code
    this.status = options.status
    this.severity = options.severity ?? 'error'
    this.details = options.details
    this.timestamp = new Date()
    this.requestID = options.requestID
  }

  // 是否为认证错误
  isAuthError(): boolean {
    return [
      ErrorCode.UNAUTHORIZED,
      ErrorCode.TOKEN_EXPIRED,
      ErrorCode.INVALID_TOKEN
    ].includes(this.code)
  }

  // 是否为网络错误
  isNetworkError(): boolean {
    return [
      ErrorCode.NETWORK_ERROR,
      ErrorCode.OFFLINE,
      ErrorCode.TIMEOUT
    ].includes(this.code)
  }

  // 是否可重试
  isRetryable(): boolean {
    return [
      ErrorCode.NETWORK_ERROR,
      ErrorCode.TIMEOUT,
      ErrorCode.SERVICE_UNAVAILABLE
    ].includes(this.code)
  }

  // 获取用户友好的错误消息（使用 i18n 翻译）
  getUserMessage(): string {
    const { t, te } = i18n.global
    // 业务错误直接返回原始消息
    if (this.code === ErrorCode.BUSINESS_ERROR) {
      return this.message
    }
    // 其他错误使用 i18n 翻译，key 不存在时 fallback 到原始消息
    const key = `errors.${this.code}`
    return te(key) ? t(key) : this.message
  }

  // 转换为 JSON
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

// 从 HTTP 状态码创建错误
export function createErrorFromStatus(
  status: number,
  message?: string,
  details?: Record<string, unknown>
): ApiError {
  const { t } = i18n.global
  const statusMap: Record<number, ErrorCode> = {
    400: ErrorCode.BAD_REQUEST,
    401: ErrorCode.UNAUTHORIZED,
    403: ErrorCode.FORBIDDEN,
    404: ErrorCode.NOT_FOUND,
    408: ErrorCode.TIMEOUT,
    409: ErrorCode.CONFLICT,
    422: ErrorCode.VALIDATION_ERROR,
    429: ErrorCode.RATE_LIMIT_EXCEEDED,
    500: ErrorCode.SERVER_ERROR,
    502: ErrorCode.SERVICE_UNAVAILABLE,
    503: ErrorCode.SERVICE_UNAVAILABLE,
    504: ErrorCode.TIMEOUT
  }

  const code = statusMap[status] || ErrorCode.UNKNOWN
  const defaultMessage = t(`errors.${code}`)

  return new ApiError({
    message: message || defaultMessage,
    code,
    status,
    details
  })
}

// 类型守卫
export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError
}
