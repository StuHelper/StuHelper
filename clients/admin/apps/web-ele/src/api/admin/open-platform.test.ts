import { beforeEach, describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => ({
  api: {
    approveOpenPlatformApp: vi.fn(),
    getOpenPlatformAuditEvents: vi.fn(),
    getOpenPlatformConsents: vi.fn(),
    getOpenPlatformDisclosureReport: vi.fn(),
    getOpenPlatformResourceGrants: vi.fn(),
    getOpenPlatformTokenProbeEvidence: vi.fn(),
    grantOpenPlatformResourceAccess: vi.fn(),
    importOpenPlatformCasdoorApp: vi.fn(),
    resumeOpenPlatformApp: vi.fn(),
    revokeOpenPlatformConsent: vi.fn(),
    revokeOpenPlatformApp: vi.fn(),
    revokeOpenPlatformResourceAccess: vi.fn(),
    rotateOpenPlatformAppSecret: vi.fn(),
    suspendOpenPlatformApp: vi.fn(),
  },
}));

vi.mock('@stuhelper/shared/api', () => ({
  createAdminApi: () => mocks.api,
}));

vi.mock('#/api/shared-client', () => ({
  sharedApiClient: {},
}));

vi.mock('#/api/shared-result', () => ({
  unwrapData: (result: unknown) => result,
  unwrapListData: (result: unknown) => result,
}));

describe('admin open platform API wrapper', () => {
  beforeEach(() => {
    for (const mock of Object.values(mocks.api)) {
      mock.mockResolvedValue({ data: { data: {} } });
    }
  });

  it('delegates legacy Casdoor imports to the shared OpenAPI client', async () => {
    const api = await import('./open-platform');
    const payload: Parameters<typeof api.importOpenPlatformCasdoorApp>[0] = {
      casdoorApplicationName: 'legacy-app',
      privacyPolicyURL: 'https://legacy.example.com/privacy',
      redirectURIs: ['https://legacy.example.com/callback'],
      scopes: [{ reason: 'legacy login', scope: 'profile.basic.read' }],
    };

    await api.importOpenPlatformCasdoorApp(payload);

    expect(mocks.api.importOpenPlatformCasdoorApp).toHaveBeenCalledWith(
      payload,
    );
  });

  it('delegates audit filters and one-time secret actions', async () => {
    const api = await import('./open-platform');

    await api.getOpenPlatformAuditEventList({
      appID: 12,
      eventType: 'open_platform.app.secret_rotated',
      scope: 'profile.basic.read',
      userID: 34,
    });
    await api.getOpenPlatformConsentList({
      appID: 12,
      userID: 34,
    });
    await api.approveOpenPlatformApp(12);
    await api.rotateOpenPlatformAppSecret(12, 'incident response');
    await api.getOpenPlatformDisclosureReport({ windowHours: 48 });
    await api.getOpenPlatformTokenProbeEvidenceList({
      appID: 12,
      clientID: 'client-12',
      result: 'failed',
      reviewerUserID: 34,
    });

    expect(mocks.api.getOpenPlatformAuditEvents).toHaveBeenCalledWith({
      appID: 12,
      eventType: 'open_platform.app.secret_rotated',
      scope: 'profile.basic.read',
      userID: 34,
    });
    expect(mocks.api.getOpenPlatformConsents).toHaveBeenCalledWith({
      appID: 12,
      userID: 34,
    });
    expect(mocks.api.approveOpenPlatformApp).toHaveBeenCalledWith(12);
    expect(mocks.api.rotateOpenPlatformAppSecret).toHaveBeenCalledWith(12, {
      reason: 'incident response',
    });
    expect(mocks.api.getOpenPlatformDisclosureReport).toHaveBeenCalledWith({
      windowHours: 48,
    });
    expect(mocks.api.getOpenPlatformTokenProbeEvidence).toHaveBeenCalledWith({
      appID: 12,
      clientID: 'client-12',
      result: 'failed',
      reviewerUserID: 34,
    });
  });

  it('delegates OpenFGA resource grant operations', async () => {
    const api = await import('./open-platform');
    const payload: Parameters<typeof api.grantOpenPlatformResourceAccess>[1] = {
      actions: ['read', 'write'],
      reason: 'approved dataset sharing',
      resourceID: 'resource-42',
      resourceType: 'resource_item',
    };

    await api.getOpenPlatformResourceGrants(12, 'resource_item');
    await api.grantOpenPlatformResourceAccess(12, payload);
    await api.revokeOpenPlatformResourceAccess(12, payload);

    expect(mocks.api.getOpenPlatformResourceGrants).toHaveBeenCalledWith(
      12,
      'resource_item',
    );
    expect(mocks.api.grantOpenPlatformResourceAccess).toHaveBeenCalledWith(
      12,
      payload,
    );
    expect(mocks.api.revokeOpenPlatformResourceAccess).toHaveBeenCalledWith(
      12,
      payload,
    );
  });

  it('delegates app lifecycle operations with audit reasons', async () => {
    const api = await import('./open-platform');

    await api.suspendOpenPlatformApp(12, 'risk review');
    await api.resumeOpenPlatformApp(12, 'risk cleared');
    await api.revokeOpenPlatformApp(13, 'owner request');

    expect(mocks.api.suspendOpenPlatformApp).toHaveBeenCalledWith(12, {
      reason: 'risk review',
    });
    expect(mocks.api.resumeOpenPlatformApp).toHaveBeenCalledWith(12, {
      reason: 'risk cleared',
    });
    expect(mocks.api.revokeOpenPlatformApp).toHaveBeenCalledWith(13, {
      reason: 'owner request',
    });
  });

  it('delegates admin consent revocation with audit reasons', async () => {
    const api = await import('./open-platform');

    await api.revokeOpenPlatformConsent({
      appID: 12,
      reason: 'privacy incident response',
      scopes: ['email.read'],
      userID: 34,
    });

    expect(mocks.api.revokeOpenPlatformConsent).toHaveBeenCalledWith(12, {
      reason: 'privacy incident response',
      scopes: ['email.read'],
      userID: 34,
    });
  });
});
