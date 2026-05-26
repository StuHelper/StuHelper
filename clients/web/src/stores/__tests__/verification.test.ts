import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const mockGetIdentity = vi.fn()
const mockGetProfile = vi.fn()
const mockGetQQBinding = vi.fn()
const mockCreateQQBindingCode = vi.fn()
const mockListSchools = vi.fn()
const mockSubmitIdentity = vi.fn()
const mockUploadIdentityPhoto = vi.fn()
const mockVerifyStudent = vi.fn()
const mockBindPhone = vi.fn()
const mockRequestBindPhoneOTP = vi.fn()

vi.mock('@/api', () => ({
  api: {
    identity: {
      getIdentity: mockGetIdentity,
      getProfile: mockGetProfile,
      getQQBinding: mockGetQQBinding,
      createQQBindingCode: mockCreateQQBindingCode,
      listSchools: mockListSchools,
      submitIdentity: mockSubmitIdentity,
      uploadIdentityPhoto: mockUploadIdentityPhoto,
      verifyStudent: mockVerifyStudent,
      bindPhone: mockBindPhone,
      requestBindPhoneOTP: mockRequestBindPhoneOTP,
    },
  },
}))

const { useVerificationStore } = await import('@/stores/verification')

describe('useVerificationStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('拉取认证状态时同步加载 QQ 绑定信息', async () => {
    mockGetIdentity.mockResolvedValue({
      data: {
        data: {
          verified: true,
        },
      },
    })
    mockGetProfile.mockResolvedValue({
      data: {
        data: {
          verificationStatus: 'verified',
          phoneVerified: true,
        },
      },
    })
    mockGetQQBinding.mockResolvedValue({
      data: {
        data: {
          userID: 42,
          qqID: '10001',
          qqNickname: '航小伴',
          boundAt: '2026-04-19T00:00:00Z',
          createdAt: '2026-04-19T00:00:00Z',
          updatedAt: '2026-04-19T00:00:00Z',
        },
      },
    })

    const store = useVerificationStore()
    await store.fetchStatus()

    expect(store.identityVerified).toBe(true)
    expect(store.studentVerified).toBe(true)
    expect(store.qqBinding?.qqID).toBe('10001')
  })

  it('QQ 绑定不存在时将状态归零', async () => {
    mockGetIdentity.mockRejectedValue({ status: 404 })
    mockGetProfile.mockRejectedValue({ status: 404 })
    mockGetQQBinding.mockRejectedValue({ status: 404 })

    const store = useVerificationStore()
    await store.fetchStatus()

    expect(store.identity).toBeNull()
    expect(store.profile).toBeNull()
    expect(store.qqBinding).toBeNull()
  })

  it('认证状态接口成功但缺少 data 时失败关闭', async () => {
    mockGetIdentity.mockResolvedValue({
      data: {
        data: null,
      },
    })
    mockGetProfile.mockRejectedValue({ status: 404 })
    mockGetQQBinding.mockRejectedValue({ status: 404 })

    const store = useVerificationStore()

    await expect(store.fetchStatus()).rejects.toThrow(
      'Invalid nullable resource response',
    )
    expect(store.identity).toBeNull()
    expect(store.profile).toBeNull()
    expect(store.qqBinding).toBeNull()
    expect(store.loading).toBe(false)
  })

  it('生成绑定码后会更新本地状态', async () => {
    mockCreateQQBindingCode.mockResolvedValue({
      data: {
        data: {
          code: 'ABCD1234',
          expiresAt: '2026-04-19T00:10:00Z',
        },
      },
    })

    const store = useVerificationStore()
    const result = await store.createQQBindingCode()

    expect(result?.code).toBe('ABCD1234')
    expect(store.qqBindingCode?.expiresAt).toBe('2026-04-19T00:10:00Z')
  })

  it('生成绑定码接口成功但缺少 data 时失败关闭', async () => {
    mockCreateQQBindingCode.mockResolvedValue({
      data: {
        data: null,
      },
    })

    const store = useVerificationStore()

    await expect(store.createQQBindingCode()).rejects.toThrow(
      'Invalid QQ binding code response',
    )
    expect(store.qqBindingCode).toBeNull()
  })

  it('证件照片上传成功但缺少 key 时失败关闭', async () => {
    mockUploadIdentityPhoto.mockResolvedValue({
      data: {
        data: {},
      },
    })

    const store = useVerificationStore()

    await expect(
      store.uploadIdentityPhoto({
        slot: 'front',
        filename: 'passport.png',
        contentType: 'image/png',
        dataBase64: 'AA==',
      }),
    ).rejects.toThrow('Invalid identity photo upload response')
  })

  it('reset 会清空 QQ 绑定相关状态', async () => {
    mockCreateQQBindingCode.mockResolvedValue({
      data: {
        data: {
          code: 'ABCD1234',
          expiresAt: '2026-04-19T00:10:00Z',
        },
      },
    })

    const store = useVerificationStore()
    await store.createQQBindingCode()
    store.reset()

    expect(store.qqBinding).toBeNull()
    expect(store.qqBindingCode).toBeNull()
    expect(store.loading).toBe(false)
  })
})
