import { hasAnyCapability } from '@stuhelper/shared/constants'

export interface ProtectedRouteAuthFailureInput {
  redirect: string
  refreshFailed: boolean
  requiresAuthRoute: boolean
  sessionBootstrapFailed: boolean
  stillAuthenticated: boolean
}

export function hasRequiredRouteCapabilityAccess(
  userCapabilities: string[],
  requiredCapabilities: string[],
) {
  if (requiredCapabilities.length === 0) {
    return true
  }

  return hasAnyCapability(userCapabilities, requiredCapabilities)
}

export function resolveProtectedRouteAuthFailure(
  input: ProtectedRouteAuthFailureInput,
) {
  if (!input.requiresAuthRoute) {
    return null
  }

  if (input.sessionBootstrapFailed) {
    return false
  }

  if (!input.refreshFailed) {
    return null
  }

  if (input.stillAuthenticated) {
    return false
  }

  return {
    name: 'login',
    query: { redirect: input.redirect },
    replace: true,
  }
}
