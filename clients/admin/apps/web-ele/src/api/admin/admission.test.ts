import { beforeEach, describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => ({
  api: {
    getFreshmanVerification: vi.fn(),
    listAdmissionPolicies: vi.fn(),
    listAdmissionSessions: vi.fn(),
    listFreshmanVerifications: vi.fn(),
    releaseAdmissionBlacklist: vi.fn(),
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

    await api.listFreshmanVerifications({ page: 1, pageSize: 20, status: 'pending' });
    await api.getFreshmanVerification('app-1');
    await api.reviewFreshmanVerification('app-1', { action: 'approve' });
    await api.listAdmissionPolicies();
    await api.updateAdmissionPolicy({ id: 'policy-1' } as never);
    await api.listAdmissionSessions({ status: 'linked' });
    await api.releaseAdmissionBlacklist('10001');

    expect(mocks.api.listFreshmanVerifications).toHaveBeenCalledWith({
      page: 1,
      pageSize: 20,
      status: 'pending',
    });
    expect(mocks.api.getFreshmanVerification).toHaveBeenCalledWith('app-1');
    expect(mocks.api.reviewFreshmanVerification).toHaveBeenCalledWith(
      'app-1',
      { action: 'approve' },
    );
    expect(mocks.api.listAdmissionPolicies).toHaveBeenCalledTimes(1);
    expect(mocks.api.updateAdmissionPolicy).toHaveBeenCalledWith(
      'policy-1',
      { id: 'policy-1' },
    );
    expect(mocks.api.listAdmissionSessions).toHaveBeenCalledWith({
      status: 'linked',
    });
    expect(mocks.api.releaseAdmissionBlacklist).toHaveBeenCalledWith('10001');
  });
});
