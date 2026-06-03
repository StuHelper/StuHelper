import { api } from '@/api'
import type {
  AdmissionMe,
  AdmissionSession,
  CameraCaptureRequest,
  CreateFreshmanApplicationRequest,
  FreshmanApplication,
  FreshmanCameraHandoff,
  FreshmanCameraHandoffContinuationRequest,
  SchoolEmailAcademicMatchRequest,
  SchoolEmailAcademicMatchResponse,
  SchoolEmailOTPRequest,
  SchoolEmailOTPResponse,
  SchoolEmailOTPVerifyRequest,
} from '@stuhelper/shared/api'

type ApiResult<T> = {
  data?: {
    data?: T
    error?: unknown
    success?: boolean
  }
  error?: unknown
}
type FreshmanCameraContinuation = NonNullable<FreshmanCameraHandoff['continueOn']>

const ADMISSION_STATUS_VALUES = new Set([
  'joined_muted',
  'linked',
  'material_submitted',
  'verified',
  'expired_kicked',
  'cancelled',
])
const FRESHMAN_APPLICATION_STATUS_VALUES = new Set([
  'pending',
  'approved',
  'rejected',
])
const FRESHMAN_CAMERA_HANDOFF_STATUS_VALUES = new Set([
  'pending',
  'uploaded',
  'locked',
  'expired',
])
const FRESHMAN_CAMERA_CONTINUATION_VALUES = new Set(['desktop', 'mobile'])
const ADMISSION_CREDENTIAL_KIND_VALUES = new Set([
  'school_sso',
  'school_email_otp',
  'freshman_material_manual',
])
const FRESHMAN_MATERIAL_TYPE_VALUES = new Set([
  'admission_notice',
  'admission_certificate',
])

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object'
}

function requireData<T>(result: ApiResult<T>, message: string): T {
  if (result.error !== undefined) {
    throw result.error
  }
  if (result.data?.success === false || result.data?.error !== undefined) {
    throw result.data
  }
  const data = result.data?.data
  if (data === undefined || data === null) {
    throw new Error(message)
  }
  return data
}

function readString(record: Record<string, unknown>, key: string, message: string): string {
  const value = record[key]
  if (typeof value !== 'string') {
    throw new Error(message)
  }
  return value
}

function readOptionalString(
  record: Record<string, unknown>,
  key: string,
  message: string,
): string | undefined {
  const value = record[key]
  if (value === undefined) {
    return undefined
  }
  if (typeof value !== 'string') {
    throw new Error(message)
  }
  return value
}

function readOptionalIDString(
  record: Record<string, unknown>,
  key: string,
  message: string,
): string | undefined {
  const value = record[key]
  if (value === undefined) {
    return undefined
  }
  if (typeof value === 'string') {
    return value
  }
  if (typeof value === 'number' && Number.isInteger(value)) {
    return String(value)
  }
  throw new Error(message)
}

function readIDString(
  record: Record<string, unknown>,
  key: string,
  message: string,
): string {
  const value = readOptionalIDString(record, key, message)
  if (value === undefined) {
    throw new Error(message)
  }
  return value
}

function readNullableString(
  record: Record<string, unknown>,
  key: string,
  message: string,
): string | null | undefined {
  const value = record[key]
  if (value === undefined || value === null) {
    return value
  }
  if (typeof value !== 'string') {
    throw new Error(message)
  }
  return value
}

function readBoolean(
  record: Record<string, unknown>,
  key: string,
  message: string,
): boolean {
  const value = record[key]
  if (typeof value !== 'boolean') {
    throw new Error(message)
  }
  return value
}

function readInteger(
  record: Record<string, unknown>,
  key: string,
  message: string,
): number {
  const value = record[key]
  if (typeof value !== 'number' || !Number.isInteger(value)) {
    throw new Error(message)
  }
  return value
}

function readOptionalInteger(
  record: Record<string, unknown>,
  key: string,
  message: string,
): number | undefined {
  const value = record[key]
  if (value === undefined) {
    return undefined
  }
  if (typeof value !== 'number' || !Number.isInteger(value)) {
    throw new Error(message)
  }
  return value
}

function readEnum<T extends string>(
  record: Record<string, unknown>,
  key: string,
  values: Set<string>,
  message: string,
): T {
  const value = readString(record, key, message)
  if (!values.has(value)) {
    throw new Error(message)
  }
  return value as T
}

function readOptionalEnum<T extends string>(
  record: Record<string, unknown>,
  key: string,
  values: Set<string>,
  message: string,
): T | undefined {
  const value = record[key]
  if (value === undefined) {
    return undefined
  }
  if (typeof value !== 'string' || !values.has(value)) {
    throw new Error(message)
  }
  return value as T
}

function readOptionalAbsoluteURL(
  record: Record<string, unknown>,
  key: string,
  message: string,
): string | undefined {
  const value = readOptionalString(record, key, message)
  if (value === undefined) {
    return undefined
  }
  try {
    const url = new URL(value)
    if (url.protocol !== 'http:' && url.protocol !== 'https:') {
      throw new Error(message)
    }
  } catch {
    throw new Error(message)
  }
  return value
}

function readAdmissionSession(
  payload: unknown,
  message: string,
): AdmissionSession {
  if (!isRecord(payload)) {
    throw new Error(message)
  }

  return {
    id: readString(payload, 'id', message),
    platform: readString(payload, 'platform', message),
    botSelfID: readOptionalString(payload, 'botSelfID', message),
    guildID: readString(payload, 'guildID', message),
    channelID: readOptionalString(payload, 'channelID', message),
    qqID: readString(payload, 'qqID', message),
    userID: readOptionalIDString(payload, 'userID', message),
    status: readEnum<AdmissionSession['status']>(
      payload,
      'status',
      ADMISSION_STATUS_VALUES,
      message,
    ),
    tokenExpiresAt: readString(payload, 'tokenExpiresAt', message),
    linkWaitDeadlineAt: readString(payload, 'linkWaitDeadlineAt', message),
    submissionWaitDeadlineAt: readString(
      payload,
      'submissionWaitDeadlineAt',
      message,
    ),
    tokenConsumedAt: readNullableString(payload, 'tokenConsumedAt', message),
    manualReviewDeadlineAt: readNullableString(
      payload,
      'manualReviewDeadlineAt',
      message,
    ),
    initialMuteUntil: readString(payload, 'initialMuteUntil', message),
    verifiedAt: readNullableString(payload, 'verifiedAt', message),
    cancelledAt: readNullableString(payload, 'cancelledAt', message),
    lastBotError: readNullableString(payload, 'lastBotError', message),
    projectionPending: readBoolean(payload, 'projectionPending', message),
    authURL: readOptionalAbsoluteURL(payload, 'authURL', message),
    maxMaterialBytes: readOptionalInteger(payload, 'maxMaterialBytes', message),
  }
}

function readAdmissionMe(payload: unknown, message: string): AdmissionMe {
  if (!isRecord(payload)) {
    throw new Error(message)
  }

  const session =
    payload.session === undefined
      ? undefined
      : readAdmissionSession(payload.session, message)

  return {
    status: readEnum<AdmissionMe['status']>(
      payload,
      'status',
      ADMISSION_STATUS_VALUES,
      message,
    ),
    projectionPending: readBoolean(payload, 'projectionPending', message),
    session,
    credentialKind: readOptionalEnum<NonNullable<AdmissionMe['credentialKind']>>(
      payload,
      'credentialKind',
      ADMISSION_CREDENTIAL_KIND_VALUES,
      message,
    ),
    provisionalExpiresAt: readNullableString(
      payload,
      'provisionalExpiresAt',
      message,
    ),
  }
}

function readFreshmanApplication(
  payload: unknown,
  message: string,
): FreshmanApplication {
  if (!isRecord(payload)) {
    throw new Error(message)
  }

  return {
    id: readString(payload, 'id', message),
    userID: readIDString(payload, 'userID', message),
    schoolID: readInteger(payload, 'schoolID', message),
    admissionSessionID: readOptionalString(
      payload,
      'admissionSessionID',
      message,
    ),
    applicantName: readOptionalString(payload, 'applicantName', message),
    applicantNameMasked: readString(payload, 'applicantNameMasked', message),
    departmentOrMajor: readOptionalString(
      payload,
      'departmentOrMajor',
      message,
    ),
    materialType: readEnum<FreshmanApplication['materialType']>(
      payload,
      'materialType',
      FRESHMAN_MATERIAL_TYPE_VALUES,
      message,
    ),
    materialURL: readOptionalAbsoluteURL(payload, 'materialURL', message),
    qqID: readOptionalString(payload, 'qqID', message),
    failureCount: readOptionalInteger(payload, 'failureCount', message),
    status: readEnum<FreshmanApplication['status']>(
      payload,
      'status',
      FRESHMAN_APPLICATION_STATUS_VALUES,
      message,
    ),
    provisionalExpiresAt: readNullableString(
      payload,
      'provisionalExpiresAt',
      message,
    ),
    reviewedAt: readNullableString(payload, 'reviewedAt', message),
    createdAt: readString(payload, 'createdAt', message),
  }
}

function readFreshmanCameraHandoff(
  payload: unknown,
  message: string,
): FreshmanCameraHandoff {
  if (!isRecord(payload)) {
    throw new Error(message)
  }
  return {
    id: readString(payload, 'id', message),
    applicationID: readString(payload, 'applicationID', message),
    userID: readIDString(payload, 'userID', message),
    status: readEnum<FreshmanCameraHandoff['status']>(
      payload,
      'status',
      FRESHMAN_CAMERA_HANDOFF_STATUS_VALUES,
      message,
    ),
    continueOn: readOptionalEnum<FreshmanCameraContinuation>(
      payload,
      'continueOn',
      FRESHMAN_CAMERA_CONTINUATION_VALUES,
      message,
    ),
    mobileURL: readOptionalAbsoluteURL(payload, 'mobileURL', message),
    expiresAt: readString(payload, 'expiresAt', message),
    uploadedAt: readOptionalString(payload, 'uploadedAt', message),
    chosenAt: readOptionalString(payload, 'chosenAt', message),
    createdAt: readString(payload, 'createdAt', message),
  }
}

function readSchoolEmailOTPResponse(
  payload: unknown,
  message: string,
): SchoolEmailOTPResponse {
  if (!isRecord(payload)) {
    throw new Error(message)
  }
  return {
    email: readString(payload, 'email', message),
    studentID: readOptionalString(payload, 'studentID', message),
    cooldownSeconds: readInteger(payload, 'cooldownSeconds', message),
  }
}

function readSchoolEmailAcademicMatchResponse(
  payload: unknown,
  message: string,
): SchoolEmailAcademicMatchResponse {
  if (!isRecord(payload)) {
    throw new Error(message)
  }
  return {
    matched: readBoolean(payload, 'matched', message),
    email: readOptionalString(payload, 'email', message),
    studentID: readOptionalString(payload, 'studentID', message),
    message: readOptionalString(payload, 'message', message),
  }
}

export const admissionApi = {
  async getAdmissionSession(token: string): Promise<AdmissionSession> {
    const result = await api.admission.getAdmissionSession(token)
    return readAdmissionSession(
      requireData(result, 'Admission session response is empty'),
      'Invalid admission session response',
    )
  },

  async linkAdmissionSession(token: string): Promise<AdmissionSession> {
    const result = await api.admission.linkAdmissionSession(token)
    return readAdmissionSession(
      requireData(result, 'Admission link response is empty'),
      'Invalid admission link response',
    )
  },

  async getAdmissionMe(admissionSessionID?: string): Promise<AdmissionMe> {
    const result = admissionSessionID
      ? await api.admission.getAdmissionMe(admissionSessionID)
      : await api.admission.getAdmissionMe()
    return readAdmissionMe(
      requireData(result, 'Admission me response is empty'),
      'Invalid admission me response',
    )
  },

  async submitFreshmanApplication(
    payload: CreateFreshmanApplicationRequest,
  ): Promise<FreshmanApplication> {
    const result = await api.admission.submitFreshmanApplication(payload)
    return readFreshmanApplication(
      requireData(result, 'Freshman application response is empty'),
      'Invalid freshman application response',
    )
  },

  async uploadCameraCapture(
    applicationID: string,
    payload: CameraCaptureRequest,
  ): Promise<FreshmanApplication> {
    const result = await api.admission.uploadCameraCapture(applicationID, payload)
    return readFreshmanApplication(
      requireData(result, 'Camera capture response is empty'),
      'Invalid camera capture response',
    )
  },

  async createFreshmanCameraHandoff(
    applicationID: string,
  ): Promise<FreshmanCameraHandoff> {
    const result = await api.admission.createFreshmanCameraHandoff(applicationID)
    return readFreshmanCameraHandoff(
      requireData(result, 'Freshman camera handoff response is empty'),
      'Invalid freshman camera handoff response',
    )
  },

  async getFreshmanCameraHandoff(
    handoffID: string,
  ): Promise<FreshmanCameraHandoff> {
    const result = await api.admission.getFreshmanCameraHandoff(handoffID)
    return readFreshmanCameraHandoff(
      requireData(result, 'Freshman camera handoff response is empty'),
      'Invalid freshman camera handoff response',
    )
  },

  async previewFreshmanMobileCameraHandoff(
    token: string,
  ): Promise<FreshmanCameraHandoff> {
    const result = await api.admission.previewFreshmanMobileCameraHandoff(token)
    return readFreshmanCameraHandoff(
      requireData(result, 'Freshman camera handoff response is empty'),
      'Invalid freshman camera handoff response',
    )
  },

  async uploadFreshmanMobileCameraCapture(
    token: string,
    payload: CameraCaptureRequest,
  ): Promise<FreshmanCameraHandoff> {
    const result = await api.admission.uploadFreshmanMobileCameraCapture(token, payload)
    return readFreshmanCameraHandoff(
      requireData(result, 'Freshman camera handoff response is empty'),
      'Invalid freshman camera handoff response',
    )
  },

  async chooseFreshmanMobileCameraContinuation(
    token: string,
    payload: FreshmanCameraHandoffContinuationRequest,
  ): Promise<FreshmanCameraHandoff> {
    const result = await api.admission.chooseFreshmanMobileCameraContinuation(token, payload)
    return readFreshmanCameraHandoff(
      requireData(result, 'Freshman camera handoff response is empty'),
      'Invalid freshman camera handoff response',
    )
  },

  async matchSchoolEmailAcademicStudent(
    payload: SchoolEmailAcademicMatchRequest,
  ): Promise<SchoolEmailAcademicMatchResponse> {
    const result = await api.admission.matchSchoolEmailAcademicStudent(payload)
    return readSchoolEmailAcademicMatchResponse(
      requireData(result, 'Academic match response is empty'),
      'Invalid academic match response',
    )
  },

  async requestSchoolEmailOTP(
    payload: SchoolEmailOTPRequest,
  ): Promise<SchoolEmailOTPResponse> {
    const result = await api.admission.requestSchoolEmailOTP(payload)
    return readSchoolEmailOTPResponse(
      requireData(result, 'School email OTP response is empty'),
      'Invalid school email OTP response',
    )
  },

  async verifySchoolEmailOTP(
    payload: SchoolEmailOTPVerifyRequest,
  ): Promise<AdmissionMe> {
    const result = await api.admission.verifySchoolEmailOTP(payload)
    return readAdmissionMe(
      requireData(result, 'School email OTP response is empty'),
      'Invalid school email OTP response',
    )
  },
}
