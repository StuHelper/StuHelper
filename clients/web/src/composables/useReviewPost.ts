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

const REVIEW_POST_ROUTE = '/courses/reviews/post'

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

  async function redirectToLoginForReviewPost() {
    await router.push({
      name: 'login',
      query: { redirect: REVIEW_POST_ROUTE },
    })
  }

  async function ensureCanPostReview() {
    if (!authStore.bootstrapCompleted) {
      try {
        await authStore.bootstrapSession()
      } catch (error) {
        if (!authStore.isAuthenticated) {
          await redirectToLoginForReviewPost()
          return false
        }
        toast.error(getErrorMessage(error, i18n.global.t('common.loadFailed')))
        return false
      }
    }

    if (!authStore.isAuthenticated) {
      await redirectToLoginForReviewPost()
      return false
    }

    try {
      const surface = (await api.identity.getUserSurface()).data?.data
      if (!surface) {
        throw new Error('Invalid user surface response')
      }

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
