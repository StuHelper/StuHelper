/**
 * API 错误类型定义
 * 统一的错误处理机制
 */

// 错误代码枚举
export enum ErrorCode {
  // 网络错误
  NETWORK_ERROR = 'NETWORK_ERROR',
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
  readonly requestId?: string

  constructor(options: {
    message: string
    code: ErrorCode
    status?: number
    severity?: ErrorSeverity
    details?: Record<string, unknown>
    requestId?: string
  }) {
    super(options.message)
    this.name = 'ApiError'
    this.code = options.code
    this.status = options.status
    this.severity = options.severity ?? 'error'
    this.details = options.details
    this.timestamp = new Date()
    this.requestId = options.requestId
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

  // 获取用户友好的错误消息
  getUserMessage(): string {
    const messages: Record<ErrorCode, string> = {
      [ErrorCode.NETWORK_ERROR]: '网络连接失败，请检查网络设置',
      [ErrorCode.TIMEOUT]: '请求超时，请稍后重试',
      [ErrorCode.UNAUTHORIZED]: '请先登录',
      [ErrorCode.TOKEN_EXPIRED]: '登录已过期，请重新登录',
      [ErrorCode.INVALID_TOKEN]: '登录信息无效，请重新登录',
      [ErrorCode.FORBIDDEN]: '没有权限执行此操作',
      [ErrorCode.BAD_REQUEST]: '请求参数错误',
      [ErrorCode.NOT_FOUND]: '请求的资源不存在',
      [ErrorCode.VALIDATION_ERROR]: '数据验证失败',
      [ErrorCode.SERVER_ERROR]: '服务器错误，请稍后重试',
      [ErrorCode.SERVICE_UNAVAILABLE]: '服务暂时不可用，请稍后重试',
      [ErrorCode.BUSINESS_ERROR]: this.message,
      [ErrorCode.UNKNOWN]: '发生未知错误'
    }
    return messages[this.code] || this.message
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
      requestId: this.requestId
    }
  }
}

// 从 HTTP 状态码创建错误
export function createErrorFromStatus(
  status: number,
  message?: string,
  details?: Record<string, unknown>
): ApiError {
  const statusMap: Record<number, { code: ErrorCode; defaultMessage: string }> = {
    400: { code: ErrorCode.BAD_REQUEST, defaultMessage: '请求参数错误' },
    401: { code: ErrorCode.UNAUTHORIZED, defaultMessage: '未授权访问' },
    403: { code: ErrorCode.FORBIDDEN, defaultMessage: '禁止访问' },
    404: { code: ErrorCode.NOT_FOUND, defaultMessage: '资源不存在' },
    408: { code: ErrorCode.TIMEOUT, defaultMessage: '请求超时' },
    422: { code: ErrorCode.VALIDATION_ERROR, defaultMessage: '数据验证失败' },
    500: { code: ErrorCode.SERVER_ERROR, defaultMessage: '服务器内部错误' },
    502: { code: ErrorCode.SERVICE_UNAVAILABLE, defaultMessage: '网关错误' },
    503: { code: ErrorCode.SERVICE_UNAVAILABLE, defaultMessage: '服务不可用' },
    504: { code: ErrorCode.TIMEOUT, defaultMessage: '网关超时' }
  }

  const errorInfo = statusMap[status] || {
    code: ErrorCode.UNKNOWN,
    defaultMessage: `HTTP 错误 ${status}`
  }

  return new ApiError({
    message: message || errorInfo.defaultMessage,
    code: errorInfo.code,
    status,
    details
  })
}

// 类型守卫
export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError
}
