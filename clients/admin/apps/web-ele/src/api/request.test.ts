import { describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => ({
  useAppConfig: vi.fn(() => ({
    apiURL: '/api/v1',
    auth: {},
  })),
}));

vi.mock('@vben/hooks', () => ({
  useAppConfig: mocks.useAppConfig,
}));

describe('admin base request client', () => {
  it('uses the API URL resolved from Vben app config', async () => {
    vi.resetModules();
    mocks.useAppConfig.mockReturnValue({
      apiURL: '/admin-api',
      auth: {},
    });

    const { baseRequestClient } = await import('./request');

    expect(baseRequestClient.getBaseUrl()).toBe('/admin-api');
    expect(mocks.useAppConfig).toHaveBeenCalledWith(
      expect.any(Object),
      expect.any(Boolean),
    );
  });
});
