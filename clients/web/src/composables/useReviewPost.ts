import { useRouter } from 'vue-router'
import { REVIEW_CREATE } from '@stuhelper/shared/constants'
import type { components } from '@stuhelper/shared/types'

import { api } from '@/api'
import { getErrorMessage } from '@/api/errors'
import i18n from '@/i18n'
import { useAuthStore } from '@/stores/auth'
import { useToast } from '@/composables/useToast'
import { accountCenterURLWithRedirect, navigateToExternalURL } from '@/utils/redirect'

export type UseReviewPostReturn = ReturnType<typeof useReviewPost>

type UserSurface = components['schemas']['UserSurface']
type PhoneStatus = components['schemas']['PhoneStatus']

interface ReviewPostBlock {
  messageKey: string
  routeName: 'home' | 'student-verification' | 'phone-binding'
}

interface EnsureReviewPostOptions {
  redirect?: string
}

const STUDENT_STATUS_VALUES = new Set(['none', 'approved'])
const PHONE_STATE_VALUES = new Set(['unbound', 'syncing', 'verified', 'review_required'])

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === 'object'
}

function readResponseData(response: unknown, message: string): unknown {
  if (!isRecord(response) || !isRecord(response.data) || !('data' in response.data)) {
    throw new Error(message)
  }
  return response.data.data
}

function readUserSurface(payload: unknown, message = 'Invalid user surface response'): UserSurface {
  if (!isRecord(payload)) throw new Error(message)
  const displayName = payload.displayName
  const status = payload.studentVerificationStatus
  const phoneBound = payload.phoneBound
  const capabilities = payload.capabilities
  if (
    typeof displayName !== 'string' ||
    typeof status !== 'string' ||
    !STUDENT_STATUS_VALUES.has(status) ||
    typeof phoneBound !== 'boolean' ||
    !Array.isArray(capabilities) ||
    capabilities.some((item) => typeof item !== 'string')
  ) {
    throw new Error(message)
  }
  return {
    displayName,
    studentVerificationStatus: status as UserSurface['studentVerificationStatus'],
    phoneBound,
    capabilities,
    ...(typeof payload.avatarURL === 'string' ? { avatarURL: payload.avatarURL } : {}),
    ...(typeof payload.phone === 'string' || payload.phone === null ? { phone: payload.phone } : {}),
  }
}

function readPhoneStatus(payload: unknown, message = 'Invalid phone status response'): PhoneStatus {
  if (!isRecord(payload)) throw new Error(message)
  const state = payload.state
  const publishingRequirementSatisfied = payload.publishingRequirementSatisfied
  const revision = payload.revision
  if (
    typeof state !== 'string' ||
    !PHONE_STATE_VALUES.has(state) ||
    typeof publishingRequirementSatisfied !== 'boolean' ||
    typeof revision !== 'number' ||
    !Number.isFinite(revision)
  ) {
    throw new Error(message)
  }
  return {
    state: state as PhoneStatus['state'],
    publishingRequirementSatisfied,
    revision,
  }
}

export function resolveReviewPostBlock(
  surface: UserSurface,
  phone: PhoneStatus,
): null | ReviewPostBlock {
  if (surface.studentVerificationStatus !== 'approved') {
    return {
      messageKey: 'review.card.verifyToView',
      routeName: 'student-verification',
    }
  }

  if (!phone.publishingRequirementSatisfied) {
    return {
      messageKey: 'user.verification.phone.platform.why.description',
      routeName: 'phone-binding',
    }
  }

  if (!surface.capabilities.includes(REVIEW_CREATE)) {
    return {
      messageKey: 'errors.A0010200',
      routeName: 'home',
    }
  }

  return null
}

export function useReviewPost() {
  const router = useRouter()
  const authStore = useAuthStore()
  const toast = useToast()

  function intendedRedirect(options?: EnsureReviewPostOptions): string {
    return options?.redirect ?? router.currentRoute.value.fullPath
  }

  async function ensureCanPostReview(options?: EnsureReviewPostOptions) {
    const redirect = intendedRedirect(options)
    if (!authStore.bootstrapCompleted) {
      try {
        await authStore.bootstrapSession()
      } catch (error) {
        if (!authStore.isAuthenticated) {
          await router.push({ name: 'login', query: { redirect } })
          return false
        }
        toast.error(getErrorMessage(error, i18n.global.t('common.loadFailed')))
        return false
      }
    }

    if (!authStore.isAuthenticated) {
      await router.push({ name: 'login', query: { redirect } })
      return false
    }

    try {
      const [surfaceResponse, phoneResponse] = await Promise.all([
        api.identity.getUserSurface(),
        api.studentVerification.getPhoneStatus(),
      ])
      const surface = readUserSurface(readResponseData(surfaceResponse, 'Invalid user surface response'))
      const phone = readPhoneStatus(readResponseData(phoneResponse, 'Invalid phone status response'))
      const block = resolveReviewPostBlock(surface, phone)
      if (!block) return true

      toast.error(i18n.global.t(block.messageKey))
      if (block.routeName === 'student-verification') {
        navigateToExternalURL(accountCenterURLWithRedirect('/user/student-verification', redirect))
      } else if (block.routeName === 'phone-binding') {
        navigateToExternalURL(accountCenterURLWithRedirect('/user/phone-binding', redirect))
      } else {
        await router.push({ name: block.routeName })
      }
      return false
    } catch (error) {
      toast.error(getErrorMessage(error, i18n.global.t('common.loadFailed')))
      return false
    }
  }

  return { ensureCanPostReview }
}
