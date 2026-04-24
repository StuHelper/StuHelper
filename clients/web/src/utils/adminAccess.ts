import {
  ADMIN_REVIEWS_MANAGE,
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
