import type { OpenPlatformAuditEventType } from '@stuhelper/shared/constants'

export const developerOpenPlatformAuditTypeKeys: Record<
  OpenPlatformAuditEventType,
  string
> = {
  'open_platform.app.approved': 'developer.apps.auditTypes.appApproved',
  'open_platform.app.approved_app_ensured':
    'developer.apps.auditTypes.approvedAppEnsured',
  'open_platform.app.identity_public_smoke_bootstrapped':
    'developer.apps.auditTypes.identityPublicSmokeBootstrapped',
  'open_platform.app.profile_updated':
    'developer.apps.auditTypes.appProfileUpdated',
  'open_platform.app.redirect_uris.approved':
    'developer.apps.auditTypes.redirectApproved',
  'open_platform.app.redirect_uris.rejected':
    'developer.apps.auditTypes.redirectRejected',
  'open_platform.app.redirect_uris.requested':
    'developer.apps.auditTypes.redirectRequested',
  'open_platform.app.redirect_uris.withdrawn':
    'developer.apps.auditTypes.redirectWithdrawn',
  'open_platform.app.resumed': 'developer.apps.auditTypes.appResumed',
  'open_platform.app.revoked': 'developer.apps.auditTypes.appRevoked',
  'open_platform.app.secret_rotated':
    'developer.apps.auditTypes.secretRotated',
  'open_platform.app.suspended': 'developer.apps.auditTypes.appSuspended',
  'open_platform.app.token_probe.failed':
    'developer.apps.auditTypes.tokenProbeFailed',
  'open_platform.app.token_probe.passed':
    'developer.apps.auditTypes.tokenProbePassed',
  'open_platform.app.token_probe.runtime.failed':
    'developer.apps.auditTypes.tokenProbeFailed',
  'open_platform.app.token_probe.runtime.passed':
    'developer.apps.auditTypes.tokenProbePassed',
  'open_platform.app.withdrawn': 'developer.apps.auditTypes.appWithdrawn',
  'open_platform.consent.denied': 'developer.apps.auditTypes.consentDenied',
  'open_platform.consent.granted': 'developer.apps.auditTypes.consentGranted',
  'open_platform.consent.revoked': 'developer.apps.auditTypes.consentRevoked',
  'open_platform.disclosure.denied':
    'developer.apps.auditTypes.disclosureDenied',
  'open_platform.disclosure.granted':
    'developer.apps.auditTypes.disclosureGranted',
  'open_platform.disclosure.replay_detected':
    'developer.apps.auditTypes.replayDetected',
  'open_platform.resource_access.checked':
    'developer.apps.auditTypes.resourceChecked',
  'open_platform.resource_access.granted':
    'developer.apps.auditTypes.resourceGranted',
  'open_platform.resource_access.revoked':
    'developer.apps.auditTypes.resourceRevoked',
  'open_platform.scope.approved': 'developer.apps.auditTypes.scopeApproved',
  'open_platform.scope.rejected': 'developer.apps.auditTypes.scopeRejected',
  'open_platform.scope.requested': 'developer.apps.auditTypes.scopeRequested',
  'open_platform.scope.withdrawn': 'developer.apps.auditTypes.scopeWithdrawn',
}

export function developerOpenPlatformAuditTypeKey(eventType: string) {
  return (
    developerOpenPlatformAuditTypeKeys[
      eventType as OpenPlatformAuditEventType
    ] ?? null
  )
}
