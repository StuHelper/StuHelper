import { describe, expect, it, vi } from 'vitest';

import { extractErrorMessage, unwrapData, unwrapVoid } from './shared-result';

const mocks = vi.hoisted(() => ({
  t: vi.fn((key: string) => key),
}));

vi.mock('#/locales', () => ({
  $t: mocks.t,
}));

describe('admin shared result helpers', () => {
  it('treats 204 responses as successful void mutations', () => {
    expect(() => unwrapVoid({ response: { status: 204 } })).not.toThrow();
  });

  it('rejects business failures on 2xx void mutation responses', () => {
    expect(() =>
      unwrapVoid({
        data: {
          success: false,
          error: {
            code: 'A0090002',
            message: '教师删除失败，请稍后重试',
          },
        },
        response: { status: 200 },
      }),
    ).toThrow('教师删除失败，请稍后重试');
  });

  it('throws typed errors without rendering toasts in the API layer', () => {
    // Presentation is owned by the view layer; unwrap* must stay side-effect
    // free so a single failure never produces duplicate feedback.
    expect(() =>
      unwrapData({
        error: {
          error: {
            code: 'A0090003',
            message: '评课不存在',
          },
        },
        response: { status: 404 },
      }),
    ).toThrow('评课不存在');
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

  it('preserves backend error messages for unmapped admin business failures', () => {
    expect(
      extractErrorMessage({
        error: {
          error: {
            code: 'A0090001',
            message: '学生认证已被其他管理员处理',
          },
        },
        response: { status: 409 },
      }),
    ).toBe('学生认证已被其他管理员处理');
  });
});
