import { describe, expect, it, vi } from 'vitest';

import { extractErrorMessage, unwrapVoid } from './shared-result';

const mocks = vi.hoisted(() => ({
  error: vi.fn(),
  t: vi.fn((key: string) => key),
}));

vi.mock('element-plus', () => ({
  ElMessage: {
    error: mocks.error,
  },
}));

vi.mock('#/locales', () => ({
  $t: mocks.t,
}));

describe('admin shared result helpers', () => {
  it('treats 204 responses as successful void mutations', () => {
    expect(() => unwrapVoid({ response: { status: 204 } })).not.toThrow();
    expect(mocks.error).not.toHaveBeenCalled();
  });

  it('keeps MFA enrollment and step-up errors distinct', () => {
    expect(
      extractErrorMessage({
        error: {
          error: {
            code: 'A0010204',
          },
        },
        response: { status: 403 },
      }),
    ).toBe('admin.result.mfaEnrollmentRequired');
    expect(
      extractErrorMessage({
        error: {
          error: {
            code: 'A0010205',
          },
        },
        response: { status: 412 },
      }),
    ).toBe('admin.result.stepUpRequired');
  });

  it('maps school configuration business errors to actionable messages', () => {
    expect(
      extractErrorMessage({
        error: {
          error: {
            code: 'A0040013',
          },
        },
        response: { status: 400 },
      }),
    ).toBe('admin.result.ldapConfigRequired');
  });
});
