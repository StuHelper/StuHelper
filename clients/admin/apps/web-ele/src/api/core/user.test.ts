import { describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => ({
  getMeApi: vi.fn(),
}));

vi.mock('./auth', () => ({
  getMeApi: mocks.getMeApi,
}));

vi.mock('@vben/preferences', () => ({
  preferences: {
    app: {
      defaultHomePath: '/dashboard',
    },
  },
}));

const { getAccessCodesApi } = await import('./user');

describe('getAccessCodesApi', () => {
  it('returns full capabilities instead of globalCapabilities fallback', async () => {
    mocks.getMeApi.mockResolvedValue({
      capabilities: ['user:school:read'],
      globalCapabilities: [],
    });

    await expect(getAccessCodesApi()).resolves.toEqual(['user:school:read']);
  });
});
