import type { RouteRecordRaw } from 'vue-router';

import { generateRoutesByFrontend } from '@vben/utils';

import { describe, expect, it, vi } from 'vitest';

import routes from './user-system';

vi.mock('#/locales', () => ({
  $t: (value: string) => value,
}));

function findRouteByName(
  tree: RouteRecordRaw[],
  name: string,
): RouteRecordRaw | undefined {
  for (const node of tree) {
    if (node.name === name) return node;
    if (node.children) {
      const hit = findRouteByName(node.children, name);
      if (hit) return hit;
    }
  }
  return undefined;
}

function childNames(tree: RouteRecordRaw[]): string[] {
  return (tree[0]?.children ?? [])
    .map((child) => child.name)
    .filter((name): name is string => typeof name === 'string');
}

// generateRoutesByFrontend → filterTree mutates `node.children` in place,
// so each test must work on a deep clone to stay isolated.
function cloneRoutes(): RouteRecordRaw[] {
  // eslint-disable-next-line unicorn/prefer-structured-clone -- route records contain component loader functions; structuredClone cannot clone them for this test fixture.
  return JSON.parse(JSON.stringify(routes)) as RouteRecordRaw[];
}

describe('user-system routes', () => {
  it('lets scoped admin entries keep the parent menu visible', () => {
    const route = routes[0];
    expect(route).toBeDefined();
    if (!route) {
      throw new Error('expected UserSystem route');
    }

    expect(route.meta?.authority).toEqual(
      expect.arrayContaining([
        'student:manual_review:read',
        'student:manual_review:decide',
        'student:verification_config:read',
        'student:credential:read',
        'student:subject_conflict:read',
        'campus_connector:health:read',
        'user:system:read',
        'admission:session:read',
        'admission:session:manage',
        'admission:policy:update',
        'member_blacklist:manage',
      ]),
    );
  });

  it('registers target verification and admission routes with separate capabilities', () => {
    const route = routes[0];
    expect(route).toBeDefined();
    if (!route) {
      throw new Error('expected UserSystem route');
    }
    const children = route.children ?? [];
    const sessions = children.find(
      (route) => route.name === 'AdmissionSessions',
    );
    const manualReview = children.find(
      (route) => route.name === 'StudentVerification',
    );
    const credentials = children.find(
      (route) => route.name === 'StudentCredentialGovernance',
    );
    const policy = children.find((route) => route.name === 'AdmissionPolicy');

    expect(sessions?.path).toBe('/users/admission-sessions');
    expect(sessions?.meta?.authority).toEqual(['admission:session:read']);
    expect(manualReview?.path).toBe('/users/student-verification');
    expect(manualReview?.meta?.authority).toEqual([
      'student:manual_review:read',
      'student:manual_review:decide',
    ]);
    expect(credentials?.path).toBe('/users/student-credentials');
    expect(policy?.path).toBe('/users/admission-policy');
    expect(policy?.meta?.authority).toEqual(['admission:policy:update']);
  });

  it('registers a dedicated member blacklist route gated by blacklist capabilities', () => {
    const route = routes[0];
    expect(route).toBeDefined();
    if (!route) {
      throw new Error('expected UserSystem route');
    }
    const children = route.children ?? [];
    const blacklist = children.find(
      (route) => route.name === 'MemberBlacklist',
    );

    expect(blacklist?.path).toBe('/users/member-blacklist');
    expect(blacklist?.meta?.authority).toEqual([
      'member_blacklist:read',
      'member_blacklist:manage',
    ]);
  });

  it('keeps the parent UserSystem entry visible for blacklist-only operators', async () => {
    const filtered = await generateRoutesByFrontend(cloneRoutes(), [
      'member_blacklist:read',
    ]);

    const parent = findRouteByName(filtered, 'UserSystem');
    expect(parent, 'parent menu must remain after filtering').toBeDefined();

    const visibleChildren = childNames(filtered);
    expect(visibleChildren).toContain('MemberBlacklist');
    // operators without admission/student-verification codes should not see those children
    expect(visibleChildren).not.toContain('StudentVerification');
    expect(visibleChildren).not.toContain('StudentCredentialGovernance');
    expect(visibleChildren).not.toContain('AdmissionPolicy');
    expect(visibleChildren).not.toContain('SystemConfig');
  });

  it('filters out the parent menu entirely when no overlapping codes are present', async () => {
    const filtered = await generateRoutesByFrontend(cloneRoutes(), [
      'unrelated:capability',
    ]);

    expect(findRouteByName(filtered, 'UserSystem')).toBeUndefined();
    expect(findRouteByName(filtered, 'MemberBlacklist')).toBeUndefined();
  });

  it('exposes admission children to admission-only operators while keeping the parent', async () => {
    const filtered = await generateRoutesByFrontend(cloneRoutes(), [
      'admission:policy:update',
      'admission:session:read',
    ]);

    expect(findRouteByName(filtered, 'UserSystem')).toBeDefined();
    const visibleChildren = childNames(filtered);
    expect(visibleChildren).toEqual(
      expect.arrayContaining(['AdmissionSessions', 'AdmissionPolicy']),
    );
    expect(visibleChildren).not.toContain('MemberBlacklist');
    expect(visibleChildren).not.toContain('StudentVerification');
  });

  it('keeps every child visible for the super-admin code set', async () => {
    const filtered = await generateRoutesByFrontend(cloneRoutes(), [
      'student:manual_review:read',
      'student:verification_config:read',
      'student:credential:read',
      'student:subject_conflict:read',
      'student:roster:read',
      'campus_connector:health:read',
      'user:system:read',
      'admission:session:read',
      'admission:policy:update',
      'member_blacklist:manage',
    ]);

    const visibleChildren = childNames(filtered);
    expect(visibleChildren).toEqual(
      expect.arrayContaining([
        'StudentVerification',
        'SchoolConfig',
        'StudentCredentialGovernance',
        'AdmissionSessions',
        'AdmissionPolicy',
        'MemberBlacklist',
        'SystemConfig',
      ]),
    );
  });
});
