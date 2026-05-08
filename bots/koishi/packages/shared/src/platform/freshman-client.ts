import type {
  FreshmanApplication,
  FreshmanCommandContext,
  FreshmanForwardItem,
  FreshmanReviewRequest,
} from '../types/index'

const ADMISSION_FRESHMAN_FORWARD_PATH = '/api/v1/bot/admission/freshman/applications/pending-forward'
const ADMISSION_FRESHMAN_APPLICATIONS_PATH = '/api/v1/bot/admission/freshman/applications'
const JSON_CONTENT_TYPE = 'application/json'

export type PlatformRequest = <T>(
  path: string,
  init: RequestInit,
  allowEmptyData?: boolean,
) => Promise<T>

export function createFreshmanClient(request: PlatformRequest) {
  return {
    async listPendingFreshmanForwards(): Promise<readonly FreshmanForwardItem[]> {
      return request<readonly FreshmanForwardItem[]>(ADMISSION_FRESHMAN_FORWARD_PATH, { method: 'GET' })
    },

    async markFreshmanForwarded(applicationID: string): Promise<void> {
      await request<void>(`${ADMISSION_FRESHMAN_APPLICATIONS_PATH}/${encodeURIComponent(applicationID)}/forwarded`, {
        method: 'POST',
      }, true)
    },

    async viewFreshmanApplication(
      applicationID: string,
      input: FreshmanCommandContext,
    ): Promise<FreshmanApplication> {
      return request<FreshmanApplication>(
        `${ADMISSION_FRESHMAN_APPLICATIONS_PATH}/${encodeURIComponent(applicationID)}/view`,
        jsonPost(input),
      )
    },

    async reviewFreshmanApplication(
      applicationID: string,
      input: FreshmanReviewRequest,
    ): Promise<FreshmanApplication> {
      return request<FreshmanApplication>(
        `${ADMISSION_FRESHMAN_APPLICATIONS_PATH}/${encodeURIComponent(applicationID)}/review`,
        jsonPost(input),
      )
    },
  }
}

function jsonPost(input: unknown): RequestInit {
  return {
    method: 'POST',
    headers: { 'Content-Type': JSON_CONTENT_TYPE },
    body: JSON.stringify(input),
  }
}
