import type { RouteRecordRaw } from 'vue-router';

import { generateRoutesByFrontend } from '@vben/utils';

import { describe, expect, it, vi } from 'vitest';

import routes from './open-platform';

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

function routeByName(name: string): RouteRecordRaw {
  const route = findRouteByName(routes, name);
  if (!route) {
    throw new Error(`expected route ${name}`);
  }
  return route;
}

// generateRoutesByFrontend mutates child arrays, so each assertion gets an isolated clone.
function cloneRoutes(): RouteRecordRaw[] {
  // eslint-disable-next-line unicorn/prefer-structured-clone -- route records contain component loader functions; structuredClone cannot clone them for this fixture.
  return JSON.parse(JSON.stringify(routes)) as RouteRecordRaw[];
}

describe('open platform admin routes', () => {
  it('keeps all enterprise Open Platform operations under one managed parent', () => {
    const parent = routeByName('OpenPlatform');
    expect(parent.path).toBe('/open-platform');
    expect(parent.meta?.authority).toEqual(['open_platform:manage']);

    const children = [
      ['OpenPlatformApps', '/open-platform/apps'],
      ['OpenPlatformAuditEvents', '/open-platform/audit-events'],
      ['OpenPlatformConsents', '/open-platform/consents'],
      ['OpenPlatformTokenProbeEvidence', '/open-platform/token-probe-evidence'],
      ['OpenPlatformDisclosureReport', '/open-platform/disclosure-report'],
    ] as const;

    for (const [name, path] of children) {
      const route = routeByName(name);
      expect(route.path).toBe(path);
      expect(route.meta?.authority).toEqual(['open_platform:manage']);
      expect(route.component).toBeTypeOf('function');
    }
  });

  it('shows the full Open Platform operations menu to operators with manage capability', async () => {
    const filtered = await generateRoutesByFrontend(cloneRoutes(), [
      'open_platform:manage',
    ]);

    expect(findRouteByName(filtered, 'OpenPlatform')).toBeDefined();
    expect(childNames(filtered)).toEqual([
      'OpenPlatformApps',
      'OpenPlatformAuditEvents',
      'OpenPlatformConsents',
      'OpenPlatformTokenProbeEvidence',
      'OpenPlatformDisclosureReport',
    ]);
  });

  it('hides Open Platform operations from unrelated operators', async () => {
    const filtered = await generateRoutesByFrontend(cloneRoutes(), [
      'user:system:read',
    ]);

    expect(findRouteByName(filtered, 'OpenPlatform')).toBeUndefined();
    expect(findRouteByName(filtered, 'OpenPlatformApps')).toBeUndefined();
  });
});
