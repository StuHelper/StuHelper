import { describe, expect, it, vi } from 'vitest';

vi.mock('#/locales', () => ({
  $t: (value: string) => value,
}));

import routes from './user-system';

describe('user-system routes', () => {
  it('lets scoped admin entries keep the parent menu visible', () => {
    const route = routes[0]!;

    expect(route.meta?.authority).toEqual(
      expect.arrayContaining([
        'user:identity:read',
        'user:identity:review',
        'user:student:review',
        'user:school:read',
        'user:system:read',
        'admission:freshman:review',
        'admission:policy:update',
        'member_blacklist:manage',
      ]),
    );
  });

  it('registers admission review and policy routes with admission capabilities', () => {
    const children = routes[0]!.children ?? [];
    const freshman = children.find((route) => route.name === 'FreshmanVerification');
    const policy = children.find((route) => route.name === 'AdmissionPolicy');

    expect(freshman?.path).toBe('/users/freshman-verification');
    expect(freshman?.meta?.authority).toEqual(['admission:freshman:review']);
    expect(policy?.path).toBe('/users/admission-policy');
    expect(policy?.meta?.authority).toEqual(['admission:policy:update']);
  });

  it('registers a dedicated member blacklist route gated by blacklist capabilities', () => {
    const children = routes[0]!.children ?? [];
    const blacklist = children.find((route) => route.name === 'MemberBlacklist');

    expect(blacklist?.path).toBe('/users/member-blacklist');
    expect(blacklist?.meta?.authority).toEqual([
      'member_blacklist:read',
      'member_blacklist:manage',
    ]);
  });
});
