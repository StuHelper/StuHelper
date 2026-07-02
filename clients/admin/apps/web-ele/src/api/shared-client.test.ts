// @vitest-environment happy-dom

import { beforeEach, describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => ({
  baseUrl: '',
  capturedTransport: null as any,
  executeSessionRefresh: vi.fn(),
  parseApiError: vi.fn((payload: unknown) => {
    if (typeof payload === 'object' && payload !== null && 'error' in payload) {
      const error = (payload as { error?: { code?: string } }).error;
      return { code: error?.code };
    }
    return {};
  }),
  readCookie: vi.fn((_name: string): null | string => null),
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
    getBaseUrl: () => mocks.baseUrl,
    instance: {
      request: mocks.request,
    },
  },
}));

vi.mock('#/api/utils/csrf', () => ({
  CSRF_COOKIE_NAME: 'csrf_token',
  readCookie: mocks.readCookie,
}));

mockVirtualModule(
  '@stuhelper/shared/api',
  () => ({
    AUTH_REFRESH_PATH: '/api/v1/auth/refresh',
    buildSecurityHeaders: (
      method: string,
      headers: Record<string, string>,
      options?: { acceptLanguage?: string; csrfToken?: null | string },
    ) => {
      const nextHeaders = { ...headers };
      if (
        !['GET', 'HEAD', 'OPTIONS'].includes(method.toUpperCase()) &&
        options?.csrfToken
      ) {
        nextHeaders['X-CSRF-Token'] = options.csrfToken;
      }
      if (options?.acceptLanguage) {
        nextHeaders['Accept-Language'] = options.acceptLanguage;
      }
      return nextHeaders;
    },
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
    normalizeSchemaPath: (baseUrl: string, schemaPath: string) => {
      const normalizedBaseUrl = baseUrl.replace(/\/+$/, '');
      for (const prefix of ['/api/v1', '/api']) {
        if (
          normalizedBaseUrl.endsWith(prefix) &&
          (schemaPath === prefix || schemaPath.startsWith(`${prefix}/`))
        ) {
          return schemaPath.slice(prefix.length) || '/';
        }
      }
      return schemaPath;
    },
    normalizeRequestHeaders: (init?: {
      headers?: Record<string, unknown>;
      params?: { header?: Record<string, unknown> };
    }) => {
      const headers: Record<string, string> = {};
      for (const [key, value] of Object.entries(init?.headers ?? {})) {
        if (value !== null && value !== undefined) headers[key] = String(value);
      }
      for (const [key, value] of Object.entries(init?.params?.header ?? {})) {
        if (value !== null && value !== undefined) headers[key] = String(value);
      }
      return headers;
    },
    CSRF_TOKEN_INVALID_CODE: 'A0010202',
    CSRF_TOKEN_MISSING_CODE: 'A0010203',
    MFA_ENROLLMENT_REQUIRED_CODE: 'A0010204',
    parseApiError: mocks.parseApiError,
    serializePath: (schemaPath: string) => schemaPath,
    STEP_UP_REQUIRED_CODE: 'A0010205',
    STEP_UP_REQUIRED_STATUSES: new Set([412, 428]),
  }),
  { virtual: true },
);

describe('admin shared client reauthentication', () => {
  beforeEach(() => {
    vi.resetModules();
    mocks.baseUrl = '';
    mocks.capturedTransport = null;
    mocks.executeSessionRefresh.mockReset();
    mocks.parseApiError.mockClear();
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

    await expect(mocks.capturedTransport.onUnauthorized()).rejects.toThrow(
      'network down',
    );
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
    expect(replaceSpy).toHaveBeenCalledWith('https://sso.example.com/login');

    replaceSpy.mockRestore();
  });

  it('redirects step-up required responses to the fetched MFA URL', async () => {
    mocks.request
      .mockRejectedValueOnce({
        response: {
          data: {
            error: {
              code: 'A0010205',
            },
          },
          status: 428,
        },
      })
      .mockResolvedValueOnce({
        data: {
          data: {
            url: 'https://sso.example.com/step-up',
          },
        },
        status: 200,
      });
    const replaceSpy = vi
      .spyOn(window.location, 'replace')
      .mockImplementation(() => undefined);

    await import('./shared-client');

    const result = await mocks.capturedTransport.request(
      'GET',
      '/api/v1/course/review/admin/reviews',
    );

    expect(replaceSpy).toHaveBeenCalledWith('https://sso.example.com/step-up');
    expect(mocks.request).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({
        method: 'GET',
        params: {
          platform: 'web',
          redirect: window.location.href,
        },
        url: '/api/v1/auth/step-up',
      }),
    );
    expect(result.response.status).toBe(428);

    replaceSpy.mockRestore();
  });

  it('redirects MFA enrollment required responses to account settings', async () => {
    mocks.request
      .mockRejectedValueOnce({
        response: {
          data: {
            error: {
              code: 'A0010204',
            },
          },
          status: 403,
        },
      })
      .mockResolvedValueOnce({
        data: {
          data: {
            accountSettingsUrl: 'https://sso.example.com/account',
          },
        },
        status: 200,
      });
    const replaceSpy = vi
      .spyOn(window.location, 'replace')
      .mockImplementation(() => undefined);

    await import('./shared-client');

    const result = await mocks.capturedTransport.request(
      'DELETE',
      '/api/v1/admin/users/1',
    );

    expect(replaceSpy).toHaveBeenCalledWith('https://sso.example.com/account');
    expect(mocks.request).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({
        method: 'GET',
        url: '/api/v1/auth/me',
      }),
    );
    expect(result.response.status).toBe(403);

    replaceSpy.mockRestore();
  });

  it('passes top-level request headers to the Vben transport', async () => {
    mocks.request.mockResolvedValue({
      data: {
        data: { ok: true },
      },
      status: 200,
    });

    await import('./shared-client');

    const result = await mocks.capturedTransport.request(
      'POST',
      '/api/v1/open-platform/resources/access/check',
      {
        body: {
          action: 'read',
          resourceID: 'resource-42',
          resourceType: 'resource_item',
        },
        headers: {
          Authorization: 'Bearer resource-access-token',
        },
      },
    );

    expect(mocks.request).toHaveBeenCalledWith(
      expect.objectContaining({
        data: {
          action: 'read',
          resourceID: 'resource-42',
          resourceType: 'resource_item',
        },
        headers: expect.objectContaining({
          Authorization: 'Bearer resource-access-token',
        }),
        method: 'POST',
        url: '/api/v1/open-platform/resources/access/check',
      }),
    );
    expect(result.response.status).toBe(200);
  });

  it('sends the browser CSRF header on refresh and treats CSRF failures as reauthentication', async () => {
    const csrfError = {
      error: {
        code: 'A0010202',
        message: 'csrf invalid',
      },
    };
    mocks.readCookie.mockReturnValue('admin-csrf-token');
    mocks.executeSessionRefresh.mockImplementationOnce(
      async (options: {
        request: (init?: unknown) => Promise<{
          error?: unknown;
          response: { status?: number };
        }>;
        shouldTreatAsUnauthorized: (
          result: {
            error?: unknown;
            response: { status?: number };
          },
          status: number | undefined,
        ) => boolean;
      }) => {
        const result = await options.request();
        return options.shouldTreatAsUnauthorized(result, result.response.status)
          ? {
              kind: 'unauthorized',
              status: result.response.status,
            }
          : {
              kind: 'error',
              error: result.error,
              status: result.response.status,
            };
      },
    );
    mocks.request.mockRejectedValueOnce({
      response: {
        data: csrfError,
        status: 403,
      },
    });

    await import('./shared-client');

    await expect(mocks.capturedTransport.refresh()).resolves.toEqual({
      kind: 'unauthorized',
      status: 401,
    });
    expect(mocks.request).toHaveBeenCalledWith(
      expect.objectContaining({
        headers: expect.objectContaining({
          'Accept-Language': 'zh-CN',
          'X-CSRF-Token': 'admin-csrf-token',
        }),
        method: 'POST',
        url: '/api/v1/auth/refresh',
        withCredentials: true,
      }),
    );
    expect(mocks.resetAllStores).not.toHaveBeenCalled();
  });

  it('normalizes schema paths when the request base URL already ends with /api', async () => {
    mocks.baseUrl = '/api';
    mocks.request.mockResolvedValue({
      data: {
        data: { ok: true },
      },
      status: 200,
    });

    await import('./shared-client');

    await mocks.capturedTransport.request('GET', '/api/v1/auth/me');

    expect(mocks.request).toHaveBeenCalledWith(
      expect.objectContaining({
        method: 'GET',
        url: '/v1/auth/me',
      }),
    );
  });
});
