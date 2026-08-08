import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const mockGetUserSurface = vi.fn()
const mockGetPhoneStatus = vi.fn()
const mockGetQQBinding = vi.fn()
const mockCreateQQBindingCode = vi.fn()

vi.mock('@/api', () => ({
  api: {
    identity: {
      getUserSurface: mockGetUserSurface,
      getQQBinding: mockGetQQBinding,
      createQQBindingCode: mockCreateQQBindingCode,
    },
    studentVerification: {
      getPhoneStatus: mockGetPhoneStatus,
    },
  },
}))

const { useVerificationStore } = await import('@/stores/verification')
const now = '2026-08-05T00:00:00Z'

const surface = {
  displayName: '张三',
  avatarURL: '',
  phone: '138****5678',
  studentVerificationStatus: 'approved',
  phoneBound: true,
  capabilities: ['review:create', 'review:list_full'],
}

const phone = {
  state: 'verified',
  maskedPhone: '138****5678',
  method: 'sms_possession',
  verifiedAt: now,
  expiresAt: null,
  publishingRequirementSatisfied: true,
  revision: 4,
}

const qqBinding = {
  userID: 42,
  qqID: '10001',
  boundAt: now,
  createdAt: now,
  updatedAt: now,
}

function ok(data: unknown) {
  return { data: { data } }
}

describe('useVerificationStore target projections', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    mockGetUserSurface.mockResolvedValue(ok(surface))
    mockGetPhoneStatus.mockResolvedValue(ok(phone))
    mockGetQQBinding.mockResolvedValue(ok(qqBinding))
  })

  it('loads student, phone and QQ as independent current facts', async () => {
    const store = useVerificationStore()
    await store.fetchStatus()

    expect(store.studentVerified).toBe(true)
    expect(store.phoneVerified).toBe(true)
    expect(store.qqBound).toBe(true)
    expect(store.qqBinding?.qqID).toBe('10001')
    expect(mockGetUserSurface).toHaveBeenCalledTimes(1)
    expect(mockGetPhoneStatus).toHaveBeenCalledTimes(1)
    expect(mockGetQQBinding).toHaveBeenCalledTimes(1)
  })

  it('accepts an explicitly unbound QQ projection without treating it as student failure', async () => {
    mockGetQQBinding.mockResolvedValue(ok(null))
    const store = useVerificationStore()
    await store.fetchStatus()

    expect(store.studentVerified).toBe(true)
    expect(store.qqBound).toBe(false)
    expect(store.qqBinding).toBeNull()
  })

  it('fails closed when the target student projection is malformed', async () => {
    mockGetUserSurface.mockResolvedValue(ok({ ...surface, studentVerificationStatus: 'verified' }))
    const store = useVerificationStore()

    await expect(store.fetchStatus()).rejects.toThrow('Invalid user surface response')
    expect(store.userSurface).toBeNull()
    expect(store.phoneStatus).toBeNull()
    expect(store.loading).toBe(false)
  })

  it('fails closed when the publishing-phone gate is malformed', async () => {
    mockGetPhoneStatus.mockResolvedValue(ok({ ...phone, publishingRequirementSatisfied: 'yes' }))
    const store = useVerificationStore()

    await expect(store.fetchStatus()).rejects.toThrow('Invalid phone status response')
    expect(store.phoneVerified).toBe(false)
  })

  it('creates and stores a fresh QQ binding code', async () => {
    mockCreateQQBindingCode.mockResolvedValue(ok({
      code: 'ABCD1234',
      expiresAt: '2026-08-05T00:10:00Z',
    }))
    const store = useVerificationStore()

    await expect(store.createQQBindingCode()).resolves.toMatchObject({ code: 'ABCD1234' })
    expect(store.qqBindingCode?.expiresAt).toBe('2026-08-05T00:10:00Z')
  })

  it('rejects a successful response that omits the data envelope', async () => {
    mockCreateQQBindingCode.mockResolvedValue({ data: {} })
    const store = useVerificationStore()

    await expect(store.createQQBindingCode()).rejects.toThrow(
      'Invalid QQ binding code response',
    )
  })
})
