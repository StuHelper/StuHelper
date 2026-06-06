import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const mockGetIdentity = vi.fn()
const mockGetProfile = vi.fn()
const mockGetUserSurface = vi.fn()
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
      getUserSurface: mockGetUserSurface,
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

const now = '2026-04-19T00:00:00Z'

const identity = {
  userID: 42,
  docType: 'MAINLAND_ID',
  realName: '张三',
  verified: true,
  verifyMethod: 'manual',
  reviewedAt: now,
  verifiedAt: now,
  rejectionReason: null,
  createdAt: now,
  updatedAt: now,
}

const profile = {
  userID: 42,
  schoolID: 4111010006,
  studentIDs: ['20260001'],
  activeStudentID: '20260001',
  verificationStatus: 'verified',
  verificationMethod: 'manual',
  rejectionReason: null,
  reviewedAt: now,
  phone: '138****5678',
  phoneVerified: true,
  consentGivenAt: now,
  verifiedAt: now,
  createdAt: now,
  updatedAt: now,
}

const userSurface = {
  displayName: '张三',
  avatarURL: '',
  phone: '138****5678',
  identityStatus: 'approved',
  verificationStatus: 'approved',
  phoneBound: true,
  capabilities: ['review:read_full'],
}

const qqBinding = {
  userID: 42,
  qqID: '10001',
  boundAt: now,
  createdAt: now,
  updatedAt: now,
}

describe('useVerificationStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('拉取认证状态时同步加载 QQ 绑定信息', async () => {
    mockGetIdentity.mockResolvedValue({
      data: {
        data: identity,
      },
    })
    mockGetProfile.mockResolvedValue({
      data: {
        data: profile,
      },
    })
    mockGetQQBinding.mockResolvedValue({
      data: {
        data: qqBinding,
      },
    })

    const store = useVerificationStore()
    await store.fetchStatus()

    expect(store.identityVerified).toBe(true)
    expect(store.studentVerified).toBe(true)
    expect(store.qqBinding?.qqID).toBe('10001')
  })

  it('状态接口返回 data:null 时将状态归零', async () => {
    mockGetIdentity.mockResolvedValue({
      data: {
        data: null,
      },
    })
    mockGetProfile.mockResolvedValue({
      data: {
        data: null,
      },
    })
    mockGetQQBinding.mockResolvedValue({
      data: {
        data: null,
      },
    })

    const store = useVerificationStore()
    await store.fetchStatus()

    expect(store.identity).toBeNull()
    expect(store.profile).toBeNull()
    expect(store.qqBinding).toBeNull()
  })

  it('兼容旧后端 404 空状态响应', async () => {
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
      data: {},
    })
    mockGetProfile.mockRejectedValue({ status: 404 })
    mockGetQQBinding.mockRejectedValue({ status: 404 })

    const store = useVerificationStore()

    await expect(store.fetchStatus()).rejects.toThrow(
      'Invalid identity response',
    )
    expect(store.identity).toBeNull()
    expect(store.profile).toBeNull()
    expect(store.qqBinding).toBeNull()
    expect(store.loading).toBe(false)
  })

  it('认证状态接口成功但字段畸形时失败关闭', async () => {
    mockGetIdentity.mockResolvedValue({
      data: {
        data: {
          ...identity,
          docType: 'UNKNOWN',
        },
      },
    })
    mockGetProfile.mockResolvedValue({
      data: {
        data: profile,
      },
    })
    mockGetQQBinding.mockResolvedValue({
      data: {
        data: qqBinding,
      },
    })

    const store = useVerificationStore()

    await expect(store.fetchStatus()).rejects.toThrow(
      'Invalid identity response',
    )
    expect(store.identity).toBeNull()
    expect(store.profile).toBeNull()
    expect(store.qqBinding).toBeNull()
  })

  it('学校列表接口成功但字段畸形时失败关闭', async () => {
    mockListSchools.mockResolvedValue({
      data: {
        data: [
          {
            schoolID: 4111010006,
            schoolName: '测试大学',
            verificationMethod: 'manual',
            manualFormFields: [
              {
                key: 'studentNo',
                label: '学号',
                type: 'text',
                required: true,
                options: null,
                placeholder: null,
              },
            ],
          },
        ],
      },
    })

    const store = useVerificationStore()

    await expect(store.fetchSchools()).rejects.toThrow(
      'Invalid school list response',
    )
    expect(store.schools).toEqual([])
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

  it('提交实名认证成功但响应字段畸形时失败关闭', async () => {
    mockSubmitIdentity.mockResolvedValue({
      data: {
        data: {
          ...identity,
          userID: '42',
        },
      },
    })

    const store = useVerificationStore()

    await expect(
      store.submitIdentity({
        docType: 'MAINLAND_ID',
        docNumber: '110101199001011237',
        realName: '张三',
      }),
    ).rejects.toThrow('Invalid identity response')
    expect(store.identity).toBeNull()
  })

  it('绑定手机刷新资料成功但响应字段畸形时失败关闭', async () => {
    mockBindPhone.mockResolvedValue({})
    mockGetUserSurface.mockResolvedValue({
      data: {
        data: userSurface,
      },
    })
    mockGetProfile.mockResolvedValue({
      data: {
        data: {
          ...profile,
          verificationStatus: 'approved',
        },
      },
    })

    const store = useVerificationStore()

    await expect(
      store.bindPhone({
        phone: '13812345678',
        otpCode: '123456',
      }),
    ).rejects.toThrow('Invalid profile response after phone binding')
    expect(store.profile).toBeNull()
    expect(store.userSurface).toBeNull()
  })

  it('绑定手机成功后通过账号聚合状态展示手机号，不要求学生档案存在', async () => {
    mockBindPhone.mockResolvedValue({})
    mockGetUserSurface.mockResolvedValue({
      data: {
        data: userSurface,
      },
    })
    mockGetProfile.mockResolvedValue({
      data: {
        data: null,
      },
    })

    const store = useVerificationStore()
    const surface = await store.bindPhone({
      phone: '13812345678',
      otpCode: '123456',
    })

    expect(surface.phoneBound).toBe(true)
    expect(store.userSurface?.phone).toBe('138****5678')
    expect(store.profile).toBeNull()
  })

  it('绑定手机成功但账号聚合响应畸形时失败关闭', async () => {
    mockBindPhone.mockResolvedValue({})
    mockGetUserSurface.mockResolvedValue({
      data: {
        data: {
          ...userSurface,
          phoneBound: 'yes',
        },
      },
    })
    mockGetProfile.mockRejectedValue({ status: 404 })

    const store = useVerificationStore()

    await expect(
      store.bindPhone({
        phone: '13812345678',
        otpCode: '123456',
      }),
    ).rejects.toThrow('Invalid user surface response after phone binding')
    expect(store.userSurface).toBeNull()
    expect(store.profile).toBeNull()
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
