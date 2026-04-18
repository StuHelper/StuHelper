import {
  ADMIN_DASHBOARD_VIEW,
  ADMIN_ENTRY_CAPABILITIES,
  ADMIN_REVIEWS_MANAGE,
  ADMIN_REPORTS_MANAGE,
  ADMIN_TEACHERS_MANAGE,
  ADMIN_SENSITIVE_WORDS_MANAGE,
  ADMIN_LOGS_VIEW,
  ALL_CAPABILITIES,
  ROLE_CAPABILITIES,
} from './capabilities.gen'

export * from './capabilities.gen'

export type Capability = (typeof ALL_CAPABILITIES)[number]
export type CapabilityRole = keyof typeof ROLE_CAPABILITIES

export const WEB_ADMIN_ENTRY_CAPABILITIES = [
  ADMIN_DASHBOARD_VIEW,
  ADMIN_REVIEWS_MANAGE,
  ADMIN_REPORTS_MANAGE,
  ADMIN_TEACHERS_MANAGE,
  ADMIN_SENSITIVE_WORDS_MANAGE,
  ADMIN_LOGS_VIEW,
] as const satisfies readonly Capability[]

export function hasCapability(
  capabilities: readonly string[] | null | undefined,
  expected: string,
): boolean {
  return capabilities?.includes(expected) === true
}

export function hasAnyCapability(
  capabilities: readonly string[] | null | undefined,
  expected: readonly string[],
): boolean {
  return expected.some((capability) => hasCapability(capabilities, capability))
}

export function canAccessAdmin(
  capabilities: readonly string[] | null | undefined,
): boolean {
  return hasAnyCapability(capabilities, ADMIN_ENTRY_CAPABILITIES)
}
