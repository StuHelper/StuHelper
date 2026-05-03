import type { AdmissionMe, AdmissionSession } from '@stuhelper/shared/api'

export type AdmissionPageState =
  | 'loading'
  | 'needsLogin'
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
  if (session.status === 'linked') return 'linked'
  if (session.status === 'material_submitted') return 'pendingReview'
  if (session.status === 'verified') return verifiedState(session.projectionPending)
  if (session.status === 'expired_kicked') return 'expired'
  return 'error'
}

export function stateFromAdmissionMe(
  admission: Pick<AdmissionMe, 'projectionPending' | 'status'>,
): AdmissionPageState {
  if (admission.status === 'verified') {
    return verifiedState(admission.projectionPending)
  }
  if (admission.status === 'material_submitted') return 'pendingReview'
  if (admission.status === 'linked') return 'linked'
  return 'error'
}

function verifiedState(projectionPending: boolean): AdmissionPageState {
  return projectionPending ? 'projectionPending' : 'approved'
}
