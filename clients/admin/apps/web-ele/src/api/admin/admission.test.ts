import { beforeEach, describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => ({
  api: {
    cancelAdminAdmissionSession: vi.fn(),
    createMemberBlacklist: vi.fn(),
    listAdmissionPolicies: vi.fn(),
    listAdmissionSessions: vi.fn(),
    listFreshmanVerifications: vi.fn(),
    listMemberBlacklist: vi.fn(),
    regenerateAdminAdmissionSession: vi.fn(),
    releaseMemberBlacklist: vi.fn(),
    resendAdminAdmissionSession: vi.fn(),
    reviewFreshmanVerification: vi.fn(),
    updateAdmissionPolicy: vi.fn(),
  },
}));

vi.mock('@stuhelper/shared/api', () => ({
  createAdmissionApi: () => mocks.api,
  isResultFailure: () => false,
}));

vi.mock('#/api/shared-client', () => ({
  sharedApiClient: {},
}));

vi.mock('#/api/shared-result', () => ({
  unwrapData: (result: unknown) => result,
  unwrapListData: (result: unknown) => result,
}));

describe('admin admission API wrapper', () => {
  beforeEach(() => {
    for (const mock of Object.values(mocks.api)) {
      mock.mockResolvedValue({ data: { data: { list: [], total: 0 } } });
    }
  });

  it('delegates freshman review and policy calls to the shared OpenAPI client', async () => {
    const api = await import('./admission');

    await api.listFreshmanVerifications({
      page: 1,
      pageSize: 20,
      status: 'pending',
    });
    await api.reviewFreshmanVerification('app-1', { action: 'approve' });
    await api.listAdmissionPolicies();
    await api.updateAdmissionPolicy({ id: 'policy-1' } as never);
    await api.listAdmissionSessions({ status: 'linked' });
    await api.resendAdminAdmissionSession('session-1');
    await api.regenerateAdminAdmissionSession('session-1');
    await api.cancelAdminAdmissionSession('session-1');
    await api.listMemberBlacklist({ status: 'active' });
    await api.createMemberBlacklist({ id: 'entry-1' } as never);
    await api.releaseMemberBlacklist('entry-1', {
      releaseReasonCode: 'release_only',
    });

    expect(mocks.api.listFreshmanVerifications).toHaveBeenCalledWith({
      page: 1,
      pageSize: 20,
      status: 'pending',
    });
    expect(mocks.api.reviewFreshmanVerification).toHaveBeenCalledWith('app-1', {
      action: 'approve',
    });
    expect(mocks.api.listAdmissionPolicies).toHaveBeenCalledTimes(1);
    expect(mocks.api.updateAdmissionPolicy).toHaveBeenCalledWith('policy-1', {
      id: 'policy-1',
    });
    expect(mocks.api.listAdmissionSessions).toHaveBeenCalledWith({
      status: 'linked',
    });
    expect(mocks.api.resendAdminAdmissionSession).toHaveBeenCalledWith(
      'session-1',
    );
    expect(mocks.api.regenerateAdminAdmissionSession).toHaveBeenCalledWith(
      'session-1',
    );
    expect(mocks.api.cancelAdminAdmissionSession).toHaveBeenCalledWith(
      'session-1',
    );
    expect(mocks.api.listMemberBlacklist).toHaveBeenCalledWith({
      status: 'active',
    });
    expect(mocks.api.createMemberBlacklist).toHaveBeenCalledWith({
      id: 'entry-1',
    });
    expect(mocks.api.releaseMemberBlacklist).toHaveBeenCalledWith('entry-1', {
      releaseReasonCode: 'release_only',
    });
  });
});
