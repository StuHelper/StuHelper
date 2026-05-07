export interface ApiResponse<T = any> {
  success: boolean
  data?: T
  error?: string
}

export function success<T>(data: T): ApiResponse<T> {
  return { success: true, data }
}

export function error(message: string): ApiResponse {
  return { success: false, error: message }
}

export function toApiErrorMessage(value: unknown): string {
  if (value instanceof Error) {
    return value.message
  }
  return String(value)
}
