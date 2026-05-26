import { useRouter } from 'vue-router'
import { REVIEW_CREATE } from '@stuhelper/shared/constants'
import type { components } from '@stuhelper/shared/types'
import { api } from '@/api'
import { getErrorMessage } from '@/api/errors'
import i18n from '@/i18n'
import { useAuthStore } from '@/stores/auth'
import { useToast } from '@/composables/useToast'

export type UseReviewPostReturn = ReturnType<typeof useReviewPost>

type UserSurface = components['schemas']['UserSurface']

interface ReviewPostBlock {
  messageKey: string
  routeName: null | 'home' | 'identity-verification' | 'student-verification'
}

const USER_SURFACE_STATUS_VALUES = new Set([
  'none',
  'pending',
  'approved',
  'rejected',
])

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object'
}

function readString(
  record: Record<string, unknown>,
  key: string,
  message: string,
): string {
  const value = record[key]
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

function readStringArray(value: unknown, message: string): string[] {
  if (!Array.isArray(value) || value.some((item) => typeof item !== 'string')) {
    throw new Error(message)
  }
  return value
}

function readSurfaceStatus(
  record: Record<string, unknown>,
  key: string,
  message: string,
): UserSurface['identityStatus'] {
  const value = readString(record, key, message)
  if (!USER_SURFACE_STATUS_VALUES.has(value)) {
    throw new Error(message)
  }
  return value as UserSurface['identityStatus']
}

function readUserSurfacePayload(
  payload: unknown,
  message = 'Invalid user surface response',
): UserSurface {
  if (!isRecord(payload)) {
    throw new Error(message)
  }

  return {
    displayName: readString(payload, 'displayName', message),
    avatarURL: readOptionalString(payload, 'avatarURL', message),
    identityStatus: readSurfaceStatus(payload, 'identityStatus', message),
    verificationStatus: readSurfaceStatus(payload, 'verificationStatus', message),
    phoneBound: readBoolean(payload, 'phoneBound', message),
    capabilities: readStringArray(payload.capabilities, message),
  }
}

export function resolveReviewPostBlock(surface: UserSurface): null | ReviewPostBlock {
  if (surface.identityStatus !== 'approved') {
    return {
      messageKey: 'user.verification.student.identityRequired',
      routeName: 'identity-verification',
    }
  }

  if (surface.verificationStatus !== 'approved') {
    return {
      messageKey: 'review.card.verifyToView',
      routeName: 'student-verification',
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

  async function ensureCanPostReview() {
    if (!authStore.bootstrapCompleted) {
      try {
        await authStore.bootstrapSession()
      } catch (error) {
        if (!authStore.isAuthenticated) {
          await router.push({
            name: 'login',
            query: { redirect: router.currentRoute.value.fullPath },
          })
          return false
        }
        toast.error(getErrorMessage(error, i18n.global.t('common.loadFailed')))
        return false
      }
    }

    if (!authStore.isAuthenticated) {
      await router.push({
        name: 'login',
        query: { redirect: router.currentRoute.value.fullPath },
      })
      return false
    }

    try {
      const surface = readUserSurfacePayload(
        (await api.identity.getUserSurface()).data?.data,
      )

      const block = resolveReviewPostBlock(surface)
      if (!block) {
        return true
      }

      toast.error(i18n.global.t(block.messageKey))
      if (block.routeName) {
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
