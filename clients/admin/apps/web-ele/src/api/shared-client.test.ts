// @vitest-environment happy-dom

import { beforeEach, describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => ({
  capturedTransport: null as any,
  executeSessionRefresh: vi.fn(),
  readCookie: vi.fn(() => null),
  request: vi.fn(),
  resetAllStores: vi.fn(),
}));

const mockVirtualModule = vi.mock as unknown as (
  path: string,
  factory: () => unknown,
  options?: { virtual?: boolean },
) => void;

vi.mock('@vben/preferences', () => ({
  preferences: {
    app: {
      locale: 'zh-CN',
    },
  },
}));

vi.mock('@vben/stores', () => ({
  resetAllStores: mocks.resetAllStores,
}));

vi.mock('#/api/request', () => ({
  baseRequestClient: {
    getBaseUrl: () => '',
    instance: {
      request: mocks.request,
    },
  },
}));

vi.mock('#/api/utils/csrf', () => ({
  CSRF_COOKIE_NAME: 'csrf_token',
  readCookie: mocks.readCookie,
}));

mockVirtualModule('@stuhelper/shared/api', () => ({
  AUTH_REFRESH_PATH: '/api/v1/auth/refresh',
  buildSecurityHeaders: (
    _method: string,
    headers: Record<string, string>,
  ) => headers,
  createSessionApiClient: (transport: unknown) => {
    mocks.capturedTransport = transport;
    return {
      DELETE: vi.fn(),
      GET: vi.fn(),
      PATCH: vi.fn(),
      POST: vi.fn(),
      PUT: vi.fn(),
    };
  },
  executeSessionRefresh: mocks.executeSessionRefresh,
  normalizeSchemaPath: (_baseUrl: string, schemaPath: string) => schemaPath,
  serializePath: (schemaPath: string) => schemaPath,
}), { virtual: true });

describe('admin shared client reauthentication', () => {
  beforeEach(() => {
    vi.resetModules();
    mocks.capturedTransport = null;
    mocks.executeSessionRefresh.mockReset();
    mocks.readCookie.mockReset();
    mocks.readCookie.mockReturnValue(null);
    mocks.request.mockReset();
    mocks.resetAllStores.mockReset();
  });

  it('throws instead of silently redirecting to /admin when login url fetch fails', async () => {
    mocks.request.mockRejectedValue(new Error('network down'));
    const replaceSpy = vi
      .spyOn(window.location, 'replace')
      .mockImplementation(() => undefined);

    await import('./shared-client');

    await expect(
      mocks.capturedTransport.onUnauthorized(),
    ).rejects.toThrow('network down');
    expect(replaceSpy).not.toHaveBeenCalledWith('/admin/');
    expect(mocks.resetAllStores).toHaveBeenCalledTimes(1);

    replaceSpy.mockRestore();
  });

  it('redirects to the fetched OIDC login url when available', async () => {
    mocks.request.mockResolvedValue({
      data: {
        data: {
          url: 'https://sso.example.com/login',
        },
      },
    });
    const replaceSpy = vi
      .spyOn(window.location, 'replace')
      .mockImplementation(() => undefined);

    await import('./shared-client');

    await expect(mocks.capturedTransport.onUnauthorized()).resolves.toBe(
      undefined,
    );
    expect(replaceSpy).toHaveBeenCalledWith(
      'https://sso.example.com/login',
    );

    replaceSpy.mockRestore();
  });
});
