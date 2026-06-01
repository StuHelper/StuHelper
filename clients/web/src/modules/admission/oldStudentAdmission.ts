import type { components } from '@stuhelper/shared/types'

type AdmissionMe = components['schemas']['AdmissionMe']
type SchoolConfig = components['schemas']['SchoolConfig']

export type AdmissionSchoolOption = SchoolConfig

export type AdmissionCredentialState = Pick<AdmissionMe, 'credentialKind'> | null

export function schoolHasAdmissionSSO(
  school: AdmissionSchoolOption | null | undefined,
): boolean {
  return school?.enabled === true && school.schoolSsoEnabled === true
}

export function schoolHasAdmissionEmailOTP(
  school: AdmissionSchoolOption | null | undefined,
): boolean {
  return school?.enabled === true && school.schoolEmailOtpEnabled === true
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
  schoolCode: string,
  returnURL: string,
): string {
  const normalizedSchoolCode = schoolCode.trim()
  if (!/^\d{10}$/.test(normalizedSchoolCode)) {
    throw new Error('School code is required for admission SSO')
  }
  const encodedReturnURL = encodeURIComponent(returnURL)
  return `/api/v1/admission/school-sso/${normalizedSchoolCode}/login?return=${encodedReturnURL}`
}
