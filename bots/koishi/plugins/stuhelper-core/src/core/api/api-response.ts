export interface ApiResponse<T = unknown> {
  success: boolean
  data?: T
  error?: string
}

export function success<T>(data: T): ApiResponse<T> {
  return { success: true, data }
}

export function error<T = never>(message: string): ApiResponse<T> {
  return { success: false, error: message }
}

export function toApiErrorMessage(value: unknown): string {
  if (value instanceof Error) {
    return value.message
  }
  return String(value)
}
