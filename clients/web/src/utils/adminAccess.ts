import {
  ADMIN_REVIEWS_EDIT_CONTENT,
  ADMIN_REVIEWS_MANAGE,
  REVIEW_LIST_FULL,
  canAccessAdmin,
  hasCapability,
} from '@stuhelper/shared/constants'

interface CapabilityUser {
  capabilities?: readonly string[] | null
  canAccessAdmin?: boolean
}

function getCapabilities(user: CapabilityUser | null | undefined) {
  return user?.capabilities ?? []
}

export function canShowAdminEntry(user: CapabilityUser | null | undefined) {
  return user?.canAccessAdmin === true || canAccessAdmin(getCapabilities(user))
}

export function canManageReviews(user: CapabilityUser | null | undefined) {
  return hasCapability(getCapabilities(user), ADMIN_REVIEWS_MANAGE)
}

export function canEditReviewContent(user: CapabilityUser | null | undefined) {
  return hasCapability(getCapabilities(user), ADMIN_REVIEWS_EDIT_CONTENT)
}

export function canListFullReviews(user: CapabilityUser | null | undefined) {
  return hasCapability(getCapabilities(user), REVIEW_LIST_FULL)
}
