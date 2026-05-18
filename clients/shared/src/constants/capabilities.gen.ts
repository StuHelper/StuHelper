/**
 * AUTO-GENERATED FILE. DO NOT EDIT.
 *
 * Source of truth:
 *   server/internal/pkg/capability/catalog.go
 *
 * Regenerate with:
 *   cd clients && pnpm run generate:capabilities
 */

export const ADMIN_DASHBOARD_VIEW = 'admin:dashboard:view' as const
export const ADMIN_REVIEWS_MANAGE = 'admin:reviews:manage' as const
export const ADMIN_REPORTS_MANAGE = 'admin:reports:manage' as const
export const ADMIN_TEACHERS_MANAGE = 'admin:teachers:manage' as const
export const ADMIN_SENSITIVE_WORDS_MANAGE = 'admin:sensitive_words:manage' as const
export const ADMIN_LOGS_VIEW = 'admin:logs:view' as const
export const USER_IDENTITY_READ = 'user:identity:read' as const
export const USER_IDENTITY_REVIEW = 'user:identity:review' as const
export const USER_STUDENT_READ = 'user:student:read' as const
export const USER_STUDENT_REVIEW = 'user:student:review' as const
export const USER_SCHOOL_READ = 'user:school:read' as const
export const USER_SCHOOL_UPDATE = 'user:school:update' as const
export const USER_SYSTEM_READ = 'user:system:read' as const
export const USER_SYSTEM_UPDATE = 'user:system:update' as const
export const ADMISSION_POLICY_READ = 'admission:policy:read' as const
export const ADMISSION_POLICY_UPDATE = 'admission:policy:update' as const
export const ADMISSION_FRESHMAN_READ = 'admission:freshman:read' as const
export const ADMISSION_FRESHMAN_REVIEW = 'admission:freshman:review' as const
export const ADMISSION_SESSION_READ = 'admission:session:read' as const
export const MEMBER_BLACKLIST_READ = 'member_blacklist:read' as const
export const MEMBER_BLACKLIST_MANAGE = 'member_blacklist:manage' as const
export const OPEN_PLATFORM_READ = 'open_platform:read' as const
export const OPEN_PLATFORM_MANAGE = 'open_platform:manage' as const
export const REVIEW_LIST_FULL = 'review:list:full' as const
export const REVIEW_CREATE = 'review:create' as const
export const REVIEW_EDIT_OWN = 'review:edit:own' as const
export const REVIEW_DELETE_OWN = 'review:delete:own' as const
export const REVIEW_LIST_BRIEF = 'review:list:brief' as const

export const ALL_CAPABILITIES = [
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
  ADMISSION_POLICY_READ,
  ADMISSION_POLICY_UPDATE,
  ADMISSION_FRESHMAN_READ,
  ADMISSION_FRESHMAN_REVIEW,
  ADMISSION_SESSION_READ,
  MEMBER_BLACKLIST_READ,
  MEMBER_BLACKLIST_MANAGE,
  OPEN_PLATFORM_READ,
  OPEN_PLATFORM_MANAGE,
  REVIEW_LIST_FULL,
  REVIEW_CREATE,
  REVIEW_EDIT_OWN,
  REVIEW_DELETE_OWN,
  REVIEW_LIST_BRIEF,
] as const

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
  ADMISSION_POLICY_READ,
  ADMISSION_POLICY_UPDATE,
  ADMISSION_FRESHMAN_READ,
  ADMISSION_FRESHMAN_REVIEW,
  ADMISSION_SESSION_READ,
  MEMBER_BLACKLIST_READ,
  MEMBER_BLACKLIST_MANAGE,
  OPEN_PLATFORM_READ,
  OPEN_PLATFORM_MANAGE,
] as const

export const ROLE_NAMES = ["super_admin", "school_admin", "section_admin", "section_moderator", "section_reviewer", "verified_student", "freshman_provisional", "user"] as const

export const ROLE_CAPABILITIES = {
  "super_admin": [ADMIN_DASHBOARD_VIEW, ADMIN_REVIEWS_MANAGE, ADMIN_REPORTS_MANAGE, ADMIN_TEACHERS_MANAGE, ADMIN_SENSITIVE_WORDS_MANAGE, ADMIN_LOGS_VIEW, USER_IDENTITY_READ, USER_IDENTITY_REVIEW, USER_STUDENT_READ, USER_STUDENT_REVIEW, USER_SCHOOL_READ, USER_SCHOOL_UPDATE, USER_SYSTEM_READ, USER_SYSTEM_UPDATE, ADMISSION_POLICY_READ, ADMISSION_POLICY_UPDATE, ADMISSION_FRESHMAN_READ, ADMISSION_FRESHMAN_REVIEW, ADMISSION_SESSION_READ, MEMBER_BLACKLIST_READ, MEMBER_BLACKLIST_MANAGE, OPEN_PLATFORM_READ, OPEN_PLATFORM_MANAGE],
  "school_admin": [ADMIN_REVIEWS_MANAGE, ADMIN_REPORTS_MANAGE, USER_STUDENT_READ, USER_STUDENT_REVIEW, USER_SCHOOL_READ, USER_SCHOOL_UPDATE],
  "section_admin": [ADMIN_REVIEWS_MANAGE, ADMIN_REPORTS_MANAGE],
  "section_moderator": [ADMIN_REVIEWS_MANAGE, ADMIN_REPORTS_MANAGE],
  "section_reviewer": [],
  "verified_student": [REVIEW_LIST_FULL, REVIEW_CREATE, REVIEW_EDIT_OWN, REVIEW_DELETE_OWN],
  "freshman_provisional": [REVIEW_LIST_FULL, REVIEW_CREATE, REVIEW_EDIT_OWN, REVIEW_DELETE_OWN],
  "user": [REVIEW_LIST_BRIEF],
} as const
