import type { components } from '@stuhelper/shared/types'

const MIN_SCHOOL_ID = 1

type AdmissionMe = components['schemas']['AdmissionMe']
type SchoolConfig = components['schemas']['SchoolConfig']

export type AdmissionSchoolOption = SchoolConfig & {
  readonly schoolSsoEnabled?: boolean
}

export type AdmissionCredentialState = Pick<AdmissionMe, 'credentialKind'> | null

export function schoolHasAdmissionSSO(
  school: AdmissionSchoolOption | null | undefined,
): boolean {
  return school?.enabled === true && school.schoolSsoEnabled === true
}

export function shouldShowFreshmanSubmission(
  admission: AdmissionCredentialState,
): boolean {
  return (
    admission?.credentialKind !== 'school_sso' &&
    admission?.credentialKind !== 'school_email_otp'
  )
}

export function buildSchoolSSOLoginPath(
  schoolID: number,
  returnURL: string,
): string {
  if (!Number.isInteger(schoolID) || schoolID < MIN_SCHOOL_ID) {
    throw new Error('School ID is required for admission SSO')
  }
  const encodedReturnURL = encodeURIComponent(returnURL)
  return `/api/v1/admission/school-sso/${schoolID}/login?return=${encodedReturnURL}`
}
