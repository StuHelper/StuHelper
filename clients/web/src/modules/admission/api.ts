import { api } from '@/api'
import type {
  AdmissionMe,
  AdmissionSession,
  CameraCaptureRequest,
  CreateFreshmanApplicationRequest,
  FreshmanApplication,
  SchoolEmailOTPRequest,
  SchoolEmailOTPVerifyRequest,
} from '@stuhelper/shared/api'

type ApiResult<T> = {
  data?: {
    data?: T
  }
}

function requireData<T>(result: ApiResult<T>, message: string): T {
  const data = result.data?.data
  if (data === undefined || data === null) {
    throw new Error(message)
  }
  return data
}

export const admissionApi = {
  async getAdmissionSession(token: string, qq?: string): Promise<AdmissionSession> {
    const result = await api.admission.getAdmissionSession(token, qq)
    return requireData(result, 'Admission session response is empty')
  },

  async linkAdmissionSession(token: string, qq?: string): Promise<AdmissionSession> {
    const result = await api.admission.linkAdmissionSession(token, qq)
    return requireData(result, 'Admission link response is empty')
  },

  async getAdmissionMe(): Promise<AdmissionMe> {
    const result = await api.admission.getAdmissionMe()
    return requireData(result, 'Admission me response is empty')
  },

  async submitFreshmanApplication(
    payload: CreateFreshmanApplicationRequest,
  ): Promise<FreshmanApplication> {
    const result = await api.admission.submitFreshmanApplication(payload)
    return requireData(result, 'Freshman application response is empty')
  },

  async uploadCameraCapture(
    applicationID: string,
    payload: CameraCaptureRequest,
  ): Promise<FreshmanApplication> {
    const result = await api.admission.uploadCameraCapture(applicationID, payload)
    return requireData(result, 'Camera capture response is empty')
  },

  async requestSchoolEmailOTP(payload: SchoolEmailOTPRequest): Promise<void> {
    await api.admission.requestSchoolEmailOTP(payload)
  },

  async verifySchoolEmailOTP(
    payload: SchoolEmailOTPVerifyRequest,
  ): Promise<AdmissionMe> {
    const result = await api.admission.verifySchoolEmailOTP(payload)
    return requireData(result, 'School email OTP response is empty')
  },
}
