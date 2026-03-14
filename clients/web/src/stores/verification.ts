/**
 * 认证状态管理（实名认证 + 学生认证）
 */
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '@/api'

interface IdentityInfo {
  userID: number
  docType: string
  realName: string
  verified: boolean
  verifyMethod: string | null
  verifiedAt: string | null
  rejectionReason: string | null
}

interface ProfileInfo {
  userID: number
  schoolID: string | null
  studentIDs: string[] | null
  activeStudentID: string | null
  verificationStatus: 'unverified' | 'pending' | 'verified' | 'rejected'
  verificationMethod: string | null
  phone: string | null
  phoneVerified: boolean
  verifiedAt: string | null
}

interface SchoolConfig {
  schoolID: string
  schoolName: string
  verificationMethod: string
  consentText: string | null
  enabled: boolean
}

export const useVerificationStore = defineStore('verification', () => {
  // 状态
  const identity = ref<IdentityInfo | null>(null)
  const profile = ref<ProfileInfo | null>(null)
  const schools = ref<SchoolConfig[]>([])
  const loading = ref(false)

  // 计算属性
  const identityVerified = computed(() => identity.value?.verified === true)
  const studentVerified = computed(() => profile.value?.verificationStatus === 'verified')
  const canViewFullReviews = computed(() => studentVerified.value)

  // 并行获取实名认证和学生认证状态
  const fetchStatus = async () => {
    loading.value = true
    try {
      const [identityRes, profileRes] = await Promise.allSettled([
        api.verification.getIdentity(),
        api.verification.getProfile(),
      ])

      // 404 表示用户尚未提交，视为 null
      if (identityRes.status === 'fulfilled') {
        identity.value = identityRes.value.data.identity
      } else {
        const err = identityRes.reason as Error & { status?: number }
        if (err.status === 404) {
          identity.value = null
        } else {
          throw err
        }
      }

      if (profileRes.status === 'fulfilled') {
        profile.value = profileRes.value.data.profile
      } else {
        const err = profileRes.reason as Error & { status?: number }
        if (err.status === 404) {
          profile.value = null
        } else {
          throw err
        }
      }
    } finally {
      loading.value = false
    }
  }

  // 获取学校列表
  const fetchSchools = async () => {
    const res = await api.verification.getSchools()
    schools.value = res.data.schools
  }

  // 提交实名认证
  const submitIdentity = async (data: {
    docType: 'MAINLAND_ID' | 'HK_MACAU' | 'TW' | 'PASSPORT'
    docNumber: string
    realName: string
    docPhotoFront?: string
    docPhotoBack?: string
    docPhotoSelfie?: string
  }) => {
    const res = await api.verification.submitIdentity(data)
    identity.value = res.data.identity
    return res.data.identity
  }

  // 学生认证
  const verifyStudent = async (data: {
    schoolID: string
    studentID: string
    password: string
    consent: boolean
  }) => {
    const res = await api.verification.verifyStudent(data)
    profile.value = res.data.profile
    return res.data.profile
  }

  // 绑定手机
  const bindPhone = async (data: { phone: string; code?: string }) => {
    await api.verification.bindPhone(data)
  }

  // 重置状态（setup store 不支持 $reset）
  const reset = () => {
    identity.value = null
    profile.value = null
    schools.value = []
    loading.value = false
  }

  return {
    identity,
    profile,
    schools,
    loading,
    identityVerified,
    studentVerified,
    canViewFullReviews,
    fetchStatus,
    fetchSchools,
    submitIdentity,
    verifyStudent,
    bindPhone,
    reset,
  }
})
