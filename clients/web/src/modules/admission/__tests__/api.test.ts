import { beforeEach, describe, expect, it, vi } from 'vitest'

const mockAdmissionApi = vi.hoisted(() => ({
  getAdmissionSession: vi.fn(),
  linkAdmissionSession: vi.fn(),
  getAdmissionMe: vi.fn(),
}))

vi.mock('@/api', () => ({ api: { admission: mockAdmissionApi } }))

const { admissionApi } = await import('../api')

const session = {
  id: 'session-1',
  platform: 'qq',
  botSelfID: '2118785781',
  guildID: 'guild-1',
  channelID: 'channel-1',
  qqID: '123456',
  userID: 'user-1',
  status: 'awaiting_requirements',
  tokenExpiresAt: '2026-06-01T10:00:00Z',
  tokenConsumedAt: '2026-06-01T10:01:00Z',
  linkWaitDeadlineAt: '2026-06-01T10:10:00Z',
  submissionWaitDeadlineAt: '2026-06-01T10:30:00Z',
  manualReviewDeadlineAt: null,
  initialMuteUntil: '2026-06-01T10:05:00Z',
  verifiedAt: null,
  cancelledAt: null,
  lastBotError: null,
  eligibilityRevision: 7,
  eligibilityEvaluatedAt: '2026-06-01T10:01:05Z',
  projectionPending: false,
  authURL: 'https://join.stuhelper.com/verify/session-1',
}

const ok = (data: unknown) => ({ data: { data } })

describe('admissionApi target response parsing', () => {
  beforeEach(() => vi.clearAllMocks())

  it('parses target-state sessions and revision fencing', async () => {
    mockAdmissionApi.getAdmissionSession.mockResolvedValue(ok(session))
    await expect(admissionApi.getAdmissionSession('token')).resolves.toEqual(session)
  })

  it('normalizes numeric internal user ids without exposing legacy profile data', async () => {
    mockAdmissionApi.linkAdmissionSession.mockResolvedValue(ok({ ...session, userID: 42 }))
    await expect(admissionApi.linkAdmissionSession('token')).resolves.toMatchObject({ userID: '42' })
  })

  it('parses admission-me without credential or freshman evidence details', async () => {
    mockAdmissionApi.getAdmissionMe.mockResolvedValue(ok({
      status: 'awaiting_requirements',
      projectionPending: false,
      session,
    }))
    await expect(admissionApi.getAdmissionMe('session-1')).resolves.toEqual({
      status: 'awaiting_requirements',
      projectionPending: false,
      session,
    })
    expect(mockAdmissionApi.getAdmissionMe).toHaveBeenCalledWith('session-1')
  })

  it('rejects retired status values instead of silently accepting compatibility states', async () => {
    mockAdmissionApi.getAdmissionSession.mockResolvedValue(ok({ ...session, status: 'linked' }))
    await expect(admissionApi.getAdmissionSession('token')).rejects.toThrow(
      'Invalid admission session response',
    )
  })

  it('fails closed when a required projection field is missing', async () => {
    mockAdmissionApi.getAdmissionSession.mockResolvedValue(ok({
      ...session,
      projectionPending: undefined,
    }))
    await expect(admissionApi.getAdmissionSession('token')).rejects.toThrow(
      'Invalid admission session response',
    )
  })
})
