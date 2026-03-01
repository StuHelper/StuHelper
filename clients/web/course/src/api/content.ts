/**
 * 内容质量检查 API
 */
import api from './index'

// 内容检查结果
export interface ContentCheckResult {
  isValid: boolean
  level?: 'block' | 'warn'
  matchCount?: number
}

// 质量检查结果
export interface QualityCheckResult {
  score: number
  suggestions: string[]
}

// 完整检查响应
export interface ContentCheckResponse {
  sensitive: ContentCheckResult
  quality: QualityCheckResult
}

// 检查内容
export function checkContent(content: string) {
  return api.post<ContentCheckResponse>('/content/check', { content })
}
