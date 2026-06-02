import type { AdmissionMe, AdmissionSession } from '@stuhelper/shared/api'

export type AdmissionPageState =
  | 'loading'
  | 'needsLogin'
  | 'accountMismatch'
  | 'qqMismatch'
  | 'ready'
  | 'linked'
  | 'pendingReview'
  | 'projectionPending'
  | 'approved'
  | 'expired'
  | 'error'

export function stateFromAdmissionSession(
  session: Pick<AdmissionSession, 'projectionPending' | 'status'>,
): AdmissionPageState {
  if (session.status === 'joined_muted') return 'ready'
  if (session.projectionPending) return 'projectionPending'
  if (session.status === 'linked') return 'linked'
  if (session.status === 'material_submitted') return 'pendingReview'
  if (session.status === 'verified') return 'approved'
  if (session.status === 'expired_kicked') return 'expired'
  return 'error'
}

export function stateFromAdmissionMe(
  admission: Pick<AdmissionMe, 'projectionPending' | 'status'>,
): AdmissionPageState {
  if (admission.projectionPending) return 'projectionPending'
  if (admission.status === 'verified') return 'approved'
  if (admission.status === 'material_submitted') return 'pendingReview'
  if (admission.status === 'linked') return 'linked'
  return 'error'
}
