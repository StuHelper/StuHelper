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
  | 'invalid'
  | 'expired'
  | 'error'

export function stateFromAdmissionSession(
  session: Pick<AdmissionSession, 'projectionPending' | 'status'>,
): AdmissionPageState {
  const status = String(session.status)
  if (status === 'created' || status === 'awaiting_account_link') return 'ready'
  if (session.projectionPending) return 'projectionPending'
  if (status === 'awaiting_requirements') return 'linked'
  if (status === 'pending_manual_review') return 'pendingReview'
  if (
    status === 'eligible' ||
    status === 'action_pending' ||
    status === 'admitted' ||
    status === 'released'
  ) return 'approved'
  if (status === 'expired' || status === 'cancelled') return 'expired'
  if (status === 'rejected') return 'invalid'
  return 'error'
}

export function stateFromAdmissionMe(
  admission: Pick<AdmissionMe, 'projectionPending' | 'status'>,
): AdmissionPageState {
  const status = String(admission.status)
  if (admission.projectionPending) return 'projectionPending'
  if (
    status === 'eligible' ||
    status === 'action_pending' ||
    status === 'admitted' ||
    status === 'released'
  ) return 'approved'
  if (status === 'pending_manual_review') return 'pendingReview'
  if (status === 'awaiting_requirements') return 'linked'
  if (status === 'created' || status === 'awaiting_account_link') return 'ready'
  if (status === 'expired' || status === 'cancelled') return 'expired'
  if (status === 'rejected') return 'invalid'
  return 'error'
}
