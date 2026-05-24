import type { OpenPlatformAuditEventType } from '@stuhelper/shared/constants';

import { OPEN_PLATFORM_AUDIT_EVENT_TYPES } from '@stuhelper/shared/constants';

type TagType = 'danger' | 'info' | 'success' | 'warning';

export const knownOpenPlatformAuditEventTypes = [
  ...OPEN_PLATFORM_AUDIT_EVENT_TYPES,
];

export const openPlatformAuditEventTypeLabelKeys: Record<
  OpenPlatformAuditEventType,
  string
> = {
  'open_platform.app.approved': 'admin.openPlatform.audit.event.appApproved',
  'open_platform.app.approved_app_ensured':
    'admin.openPlatform.audit.event.approvedAppEnsured',
  'open_platform.app.identity_public_smoke_bootstrapped':
    'admin.openPlatform.audit.event.identityPublicSmokeBootstrapped',
  'open_platform.app.profile_updated':
    'admin.openPlatform.audit.event.appProfileUpdated',
  'open_platform.app.redirect_uris.approved':
    'admin.openPlatform.audit.event.redirectApproved',
  'open_platform.app.redirect_uris.rejected':
    'admin.openPlatform.audit.event.redirectRejected',
  'open_platform.app.redirect_uris.requested':
    'admin.openPlatform.audit.event.redirectRequested',
  'open_platform.app.redirect_uris.withdrawn':
    'admin.openPlatform.audit.event.redirectWithdrawn',
  'open_platform.app.resumed': 'admin.openPlatform.audit.event.appResumed',
  'open_platform.app.revoked': 'admin.openPlatform.audit.event.appRevoked',
  'open_platform.app.secret_rotated':
    'admin.openPlatform.audit.event.appSecretRotated',
  'open_platform.app.suspended': 'admin.openPlatform.audit.event.appSuspended',
  'open_platform.app.token_probe.failed':
    'admin.openPlatform.audit.event.tokenProbeFailed',
  'open_platform.app.token_probe.passed':
    'admin.openPlatform.audit.event.tokenProbePassed',
  'open_platform.app.token_probe.runtime.failed':
    'admin.openPlatform.audit.event.runtimeTokenProbeFailed',
  'open_platform.app.token_probe.runtime.passed':
    'admin.openPlatform.audit.event.runtimeTokenProbePassed',
  'open_platform.app.withdrawn': 'admin.openPlatform.audit.event.appWithdrawn',
  'open_platform.consent.denied':
    'admin.openPlatform.audit.event.consentDenied',
  'open_platform.consent.granted':
    'admin.openPlatform.audit.event.consentGranted',
  'open_platform.consent.revoked':
    'admin.openPlatform.audit.event.consentRevoked',
  'open_platform.disclosure.denied':
    'admin.openPlatform.audit.event.disclosureDenied',
  'open_platform.disclosure.granted':
    'admin.openPlatform.audit.event.disclosureGranted',
  'open_platform.disclosure.replay_detected':
    'admin.openPlatform.audit.event.disclosureReplayDetected',
  'open_platform.resource_access.checked':
    'admin.openPlatform.audit.event.resourceChecked',
  'open_platform.resource_access.granted':
    'admin.openPlatform.audit.event.resourceGranted',
  'open_platform.resource_access.revoked':
    'admin.openPlatform.audit.event.resourceRevoked',
  'open_platform.scope.approved':
    'admin.openPlatform.audit.event.scopeApproved',
  'open_platform.scope.rejected':
    'admin.openPlatform.audit.event.scopeRejected',
  'open_platform.scope.requested':
    'admin.openPlatform.audit.event.scopeRequested',
  'open_platform.scope.withdrawn':
    'admin.openPlatform.audit.event.scopeWithdrawn',
};

export function eventTagType(eventType: string): TagType {
  if (
    eventType.includes('.revoked') ||
    eventType.includes('.rejected') ||
    eventType.includes('.denied') ||
    eventType.includes('.failed') ||
    eventType.includes('replay_detected')
  ) {
    return 'danger';
  }
  if (
    eventType.includes('.approved') ||
    eventType.includes('.granted') ||
    eventType.includes('.passed') ||
    eventType.includes('.resumed')
  ) {
    return 'success';
  }
  if (eventType.includes('.requested') || eventType.includes('.rotated')) {
    return 'warning';
  }
  return 'info';
}
