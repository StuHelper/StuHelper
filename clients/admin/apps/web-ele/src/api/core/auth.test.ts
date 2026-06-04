// @vitest-environment happy-dom

import { beforeEach, describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => ({
  authApi: {
    me: vi.fn(),
    refresh: vi.fn(),
  },
  baseAuthApi: {
    login: vi.fn(),
    logout: vi.fn(),
    me: vi.fn(),
  },
}));

vi.mock('@stuhelper/shared/api', () => ({
  createAuthApi: (client: { kind: string }) =>
    client.kind === 'base' ? mocks.baseAuthApi : mocks.authApi,
  extractResultErrorCode: vi.fn(() => undefined),
}));

vi.mock('#/api/shared-client', () => ({
  sharedApiClient: { kind: 'shared' },
  sharedBaseApiClient: { kind: 'base' },
}));

vi.mock('#/api/shared-result', () => ({
  unwrapData: (value: any) => value?.data?.data ?? value?.data ?? value,
}));

const { redirectToOIDCLogin, resolveOIDCRedirectURL } = await import('./auth');

describe('admin auth redirect URL', () => {
  beforeEach(() => {
    vi.stubEnv('BASE_URL', '/admin/');
    window.history.replaceState({}, '', '/admin/auth/login');
    mocks.baseAuthApi.login.mockReset();
  });

  it('prefixes router-internal paths with the admin deployment base', () => {
    expect(resolveOIDCRedirectURL('/analytics', '/admin/')).toBe(
      `${window.location.origin}/admin/analytics`,
    );
    expect(
      resolveOIDCRedirectURL(
        '/users/admission-sessions?status=pending',
        '/admin/',
      ),
    ).toBe(
      `${window.location.origin}/admin/users/admission-sessions?status=pending`,
    );
  });

  it('does not double-prefix paths that already include the admin base', () => {
    expect(resolveOIDCRedirectURL('/admin/analytics', '/admin/')).toBe(
      `${window.location.origin}/admin/analytics`,
    );
  });

  it('keeps absolute and scheme-relative redirects unchanged for backend validation', () => {
    expect(
      resolveOIDCRedirectURL('https://stuhelper.com/admin/analytics', '/admin/'),
    ).toBe('https://stuhelper.com/admin/analytics');
    expect(resolveOIDCRedirectURL('//evil.example.com/path', '/admin/')).toBe(
      '//evil.example.com/path',
    );
  });

  it('sends the admin-base URL to the backend login endpoint', async () => {
    const origin = window.location.origin;
    mocks.baseAuthApi.login.mockResolvedValue({
      data: { data: { url: 'https://sso.stuhelper.com/login/oauth/authorize' } },
    });

    await redirectToOIDCLogin('/analytics');

    expect(mocks.baseAuthApi.login).toHaveBeenCalledWith(
      `${origin}/admin/analytics`,
      undefined,
      'admin',
    );
  });
});
