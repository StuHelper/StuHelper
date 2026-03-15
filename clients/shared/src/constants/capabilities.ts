export const ADMIN_DASHBOARD_VIEW = 'admin:dashboard:view'
export const ADMIN_REVIEWS_MANAGE = 'admin:reviews:manage'
export const ADMIN_REPORTS_MANAGE = 'admin:reports:manage'
export const ADMIN_TEACHERS_MANAGE = 'admin:teachers:manage'
export const ADMIN_SENSITIVE_WORDS_MANAGE = 'admin:sensitive_words:manage'
export const ADMIN_LOGS_VIEW = 'admin:logs:view'

export const USER_IDENTITY_READ = 'user:identity:read'
export const USER_IDENTITY_REVIEW = 'user:identity:review'
export const USER_STUDENT_READ = 'user:student:read'
export const USER_STUDENT_REVIEW = 'user:student:review'
export const USER_SCHOOL_READ = 'user:school:read'
export const USER_SCHOOL_UPDATE = 'user:school:update'
export const USER_SYSTEM_READ = 'user:system:read'
export const USER_SYSTEM_UPDATE = 'user:system:update'

export const RBAC_ROLE_READ = 'rbac:role:read'
export const RBAC_ROLE_CREATE = 'rbac:role:create'
export const RBAC_ROLE_UPDATE = 'rbac:role:update'
export const RBAC_ROLE_DELETE = 'rbac:role:delete'
export const RBAC_PERMISSION_READ = 'rbac:permission:read'
export const RBAC_USER_READ = 'rbac:user:read'
export const RBAC_USER_UPDATE = 'rbac:user:update'
export const RBAC_GROUP_READ = 'rbac:group:read'
export const RBAC_GROUP_CREATE = 'rbac:group:create'
export const RBAC_GROUP_UPDATE = 'rbac:group:update'
export const RBAC_GROUP_DELETE = 'rbac:group:delete'

export const ADMIN_ENTRY_CAPABILITIES = [
  ADMIN_DASHBOARD_VIEW,
  ADMIN_REVIEWS_MANAGE,
  ADMIN_REPORTS_MANAGE,
  ADMIN_TEACHERS_MANAGE,
  ADMIN_SENSITIVE_WORDS_MANAGE,
  ADMIN_LOGS_VIEW,
  USER_IDENTITY_READ,
  USER_IDENTITY_REVIEW,
  USER_STUDENT_READ,
  USER_STUDENT_REVIEW,
  USER_SCHOOL_READ,
  USER_SCHOOL_UPDATE,
  USER_SYSTEM_READ,
  USER_SYSTEM_UPDATE,
  RBAC_ROLE_READ,
  RBAC_ROLE_CREATE,
  RBAC_ROLE_UPDATE,
  RBAC_ROLE_DELETE,
  RBAC_PERMISSION_READ,
  RBAC_USER_READ,
  RBAC_USER_UPDATE,
  RBAC_GROUP_READ,
  RBAC_GROUP_CREATE,
  RBAC_GROUP_UPDATE,
  RBAC_GROUP_DELETE,
] as const

export const WEB_ADMIN_ENTRY_CAPABILITIES = [
  ADMIN_DASHBOARD_VIEW,
  ADMIN_REVIEWS_MANAGE,
  ADMIN_REPORTS_MANAGE,
  ADMIN_TEACHERS_MANAGE,
  ADMIN_SENSITIVE_WORDS_MANAGE,
  ADMIN_LOGS_VIEW,
] as const

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
