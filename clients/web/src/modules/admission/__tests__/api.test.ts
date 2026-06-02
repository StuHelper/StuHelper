import { beforeEach, describe, expect, it, vi } from 'vitest'

const mockAdmissionApi = vi.hoisted(() => ({
  getAdmissionSession: vi.fn(),
  linkAdmissionSession: vi.fn(),
  getAdmissionMe: vi.fn(),
  submitFreshmanApplication: vi.fn(),
  uploadCameraCapture: vi.fn(),
  createFreshmanCameraHandoff: vi.fn(),
  getFreshmanCameraHandoff: vi.fn(),
  previewFreshmanMobileCameraHandoff: vi.fn(),
  uploadFreshmanMobileCameraCapture: vi.fn(),
  chooseFreshmanMobileCameraContinuation: vi.fn(),
  requestSchoolEmailOTP: vi.fn(),
  verifySchoolEmailOTP: vi.fn(),
}))

vi.mock('@/api', () => ({
  api: {
    admission: mockAdmissionApi,
  },
}))

const { admissionApi } = await import('../api')

const session = {
  id: 'session-1',
  platform: 'qq',
  botSelfID: '2118785781',
  guildID: 'guild-1',
  channelID: 'channel-1',
  qqID: '123456',
  userID: 'user-1',
  status: 'linked',
  tokenExpiresAt: '2026-06-01T10:00:00Z',
  tokenConsumedAt: '2026-06-01T10:01:00Z',
  linkWaitDeadlineAt: '2026-06-01T10:10:00Z',
  submissionWaitDeadlineAt: '2026-06-01T10:30:00Z',
  manualReviewDeadlineAt: null,
  initialMuteUntil: '2026-06-01T10:05:00Z',
  verifiedAt: null,
  cancelledAt: null,
  lastBotError: 'last bot sync failed',
  projectionPending: false,
  authURL: 'https://join.stuhelper.com/verify/session-1',
  maxMaterialBytes: 5_242_880,
}

const application = {
  id: 'application-1',
  userID: 'user-1',
  schoolID: 1001,
  admissionSessionID: 'session-1',
  applicantName: '张三',
  applicantNameMasked: '张*',
  departmentOrMajor: '计算机科学与技术',
  materialType: 'admission_notice',
  materialURL: 'https://stuhelper.com/materials/application-1.jpg',
  qqID: '123456',
  failureCount: 0,
  status: 'pending',
  provisionalExpiresAt: null,
  reviewedAt: null,
  createdAt: '2026-06-01T10:00:00Z',
}

const handoff = {
  id: 'handoff-1',
  applicationID: 'application-1',
  userID: 'user-1',
  status: 'pending',
  mobileURL: 'https://join.stuhelper.com/admission/freshman/camera/token-1',
  expiresAt: '2026-06-01T10:30:00Z',
  createdAt: '2026-06-01T10:00:00Z',
}

function ok(data: unknown) {
  return {
    data: {
      data,
    },
  }
}

describe('admissionApi response parsing', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('returns a valid admission session response', async () => {
    mockAdmissionApi.getAdmissionSession.mockResolvedValue(ok(session))

    await expect(admissionApi.getAdmissionSession('token')).resolves.toEqual(
      session,
    )
  })

  it('normalizes numeric admission user ids returned by the backend', async () => {
    mockAdmissionApi.linkAdmissionSession.mockResolvedValue(ok({
      ...session,
      userID: 42,
    }))

    await expect(admissionApi.linkAdmissionSession('token')).resolves.toMatchObject({
      userID: '42',
    })
  })

  it('rejects malformed admission session responses', async () => {
    mockAdmissionApi.getAdmissionSession.mockResolvedValue(
      ok({
        ...session,
        projectionPending: undefined,
      }),
    )

    await expect(admissionApi.getAdmissionSession('token')).rejects.toThrow(
      'Invalid admission session response',
    )
  })

  it('propagates structured admission API errors instead of rewriting them as empty data', async () => {
    const error = {
      success: false,
      error: {
        code: 'admission.token_not_found',
        message: 'admission token not found',
      },
    }
    mockAdmissionApi.getAdmissionSession.mockResolvedValue({ error })

    await expect(admissionApi.getAdmissionSession('token')).rejects.toEqual(error)
  })

  it('validates admission me and nested session responses', async () => {
    mockAdmissionApi.getAdmissionMe.mockResolvedValue(
      ok({
        status: 'linked',
        projectionPending: false,
        session,
        credentialKind: 'school_email_otp',
        provisionalExpiresAt: null,
      }),
    )

    await expect(admissionApi.getAdmissionMe()).resolves.toEqual({
      status: 'linked',
      projectionPending: false,
      session,
      credentialKind: 'school_email_otp',
      provisionalExpiresAt: null,
    })
  })

  it('rejects malformed freshman application responses', async () => {
    mockAdmissionApi.submitFreshmanApplication.mockResolvedValue(
      ok({
        ...application,
        createdAt: undefined,
      }),
    )

    await expect(
      admissionApi.submitFreshmanApplication({
        schoolCode: '4111010006',
        applicantName: '张三',
        materialType: 'admission_notice',
      }),
    ).rejects.toThrow('Invalid freshman application response')
  })

  it('normalizes numeric freshman application user ids returned by the backend', async () => {
    mockAdmissionApi.submitFreshmanApplication.mockResolvedValue(ok({
      ...application,
      userID: 42,
    }))

    await expect(admissionApi.submitFreshmanApplication({
      schoolCode: '4111010006',
      applicantName: '张三',
      materialType: 'admission_notice',
    })).resolves.toMatchObject({
      userID: '42',
    })
  })

  it('rejects malformed school email OTP responses', async () => {
    mockAdmissionApi.verifySchoolEmailOTP.mockResolvedValue(
      ok({
        status: 'linked',
        projectionPending: false,
        credentialKind: 'unknown',
      }),
    )

    await expect(
      admissionApi.verifySchoolEmailOTP({
        schoolCode: '4111010006',
        email: 'student@example.edu',
        code: '123456',
      }),
    ).rejects.toThrow('Invalid school email OTP response')
  })

  it('normalizes freshman camera handoff responses', async () => {
    mockAdmissionApi.createFreshmanCameraHandoff.mockResolvedValue(ok({
      ...handoff,
      userID: 42,
    }))

    await expect(
      admissionApi.createFreshmanCameraHandoff('application-1'),
    ).resolves.toMatchObject({
      id: 'handoff-1',
      userID: '42',
      mobileURL: 'https://join.stuhelper.com/admission/freshman/camera/token-1',
      status: 'pending',
    })
  })

  it('rejects malformed freshman camera handoff responses', async () => {
    mockAdmissionApi.previewFreshmanMobileCameraHandoff.mockResolvedValue(ok({
      ...handoff,
      status: 'unknown',
    }))

    await expect(
      admissionApi.previewFreshmanMobileCameraHandoff('token-1'),
    ).rejects.toThrow('Invalid freshman camera handoff response')
  })
})
