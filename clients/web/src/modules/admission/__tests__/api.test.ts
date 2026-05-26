import { beforeEach, describe, expect, it, vi } from 'vitest'

const mockAdmissionApi = vi.hoisted(() => ({
  getAdmissionSession: vi.fn(),
  linkAdmissionSession: vi.fn(),
  getAdmissionMe: vi.fn(),
  submitFreshmanApplication: vi.fn(),
  uploadCameraCapture: vi.fn(),
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
  guildID: 'guild-1',
  channelID: 'channel-1',
  qqID: '123456',
  qqNickname: 'Stu',
  userID: 'user-1',
  status: 'linked',
  tokenExpiresAt: '2026-06-01T10:00:00Z',
  linkWaitDeadlineAt: '2026-06-01T10:10:00Z',
  submissionWaitDeadlineAt: '2026-06-01T10:30:00Z',
  manualReviewDeadlineAt: null,
  initialMuteUntil: '2026-06-01T10:05:00Z',
  projectionPending: false,
  authURL: 'https://stuhelper.com/admission/a/session-1',
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
        schoolID: 1001,
        materialType: 'admission_notice',
      }),
    ).rejects.toThrow('Invalid freshman application response')
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
        schoolID: 1001,
        email: 'student@example.edu',
        code: '123456',
      }),
    ).rejects.toThrow('Invalid school email OTP response')
  })
})
