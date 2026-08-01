import type { RouteRecordRaw } from 'vue-router';

import { createMemoryHistory, createRouter } from 'vue-router';

import { cloneDeep, generateRoutesByFrontend } from '@vben/utils';

import { describe, expect, it, vi } from 'vitest';

import { resolveAccessibleRedirect, resolveHomePath } from './route-resolution';
import contentRouteRecords from './routes/modules/content';
import dashboardRouteRecords from './routes/modules/dashboard';

vi.mock('#/locales', () => ({
  $t: (value: string) => value,
}));

const view = { template: '<div />' };

function contentRoutes(): RouteRecordRaw[] {
  return [
    {
      children: [
        {
          component: view,
          name: 'ReviewManage',
          path: '/content/reviews',
        },
      ],
      name: 'ContentManage',
      path: '/content',
    },
  ];
}

function dashboardRoutes(): RouteRecordRaw[] {
  return [
    {
      children: [
        {
          component: view,
          name: 'Analytics',
          path: '/analytics',
        },
      ],
      name: 'Dashboard',
      path: '/dashboard',
    },
  ];
}

function cloneRoleRoutes(): RouteRecordRaw[] {
  return cloneDeep([...dashboardRouteRecords, ...contentRouteRecords]);
}

function testRouter(accessibleRoutes: RouteRecordRaw[]) {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      ...accessibleRoutes,
      {
        children: [
          {
            component: view,
            name: 'Login',
            path: 'login',
          },
        ],
        name: 'Authentication',
        path: '/auth',
      },
      {
        component: view,
        name: 'FallbackNotFound',
        path: '/:path(.*)*',
      },
    ],
  });
}

describe('admin route resolution', () => {
  it('keeps the configured dashboard home for a role that can access it', () => {
    expect(
      resolveHomePath(
        dashboardRoutes(),
        [{ children: [{ path: '/analytics' }], path: '/dashboard' }],
        '/analytics',
      ),
    ).toBe('/analytics');
  });

  it('uses the first generated menu leaf when the dashboard is inaccessible', () => {
    expect(
      resolveHomePath(
        contentRoutes(),
        [{ children: [{ path: '/content/reviews' }], path: '/content' }],
        '/analytics',
      ),
    ).toBe('/content/reviews');
  });

  it('lands a review-only admin on its first capability-filtered route', async () => {
    const routes = await generateRoutesByFrontend(cloneRoleRoutes(), [
      'admin:reviews:manage',
    ]);

    expect(
      resolveHomePath(
        routes,
        [{ children: [{ path: '/content/reviews' }], path: '/content' }],
        '/analytics',
      ),
    ).toBe('/content/reviews');
    expect(
      routes.some((route) => route.name === 'Dashboard'),
      'dashboard must be filtered before home resolution',
    ).toBe(false);
  });

  it('preserves an accessible redirect including its query and hash', () => {
    const routes = contentRoutes();
    const resolved = resolveAccessibleRedirect(
      testRouter(routes),
      '/content/reviews?page=2#pending',
      routes,
      '/content/reviews',
    );

    expect(resolved.fullPath).toBe('/content/reviews?page=2#pending');
    expect(resolved.name).toBe('ReviewManage');
  });

  it.each([
    ['/analytics', 'a filtered protected route'],
    ['/does-not-exist', 'an unknown route'],
    ['/auth/login', 'a core route outside the generated access set'],
    ['https://example.com/admin', 'an external URL'],
    ['//example.com/admin', 'a scheme-relative URL'],
    ['%E0%A4%A', 'an invalid encoded path'],
  ])('falls back from %s (%s)', (candidate) => {
    const routes = contentRoutes();
    const resolved = resolveAccessibleRedirect(
      testRouter(routes),
      candidate,
      routes,
      '/content/reviews',
    );

    expect(resolved.fullPath).toBe('/content/reviews');
    expect(resolved.name).toBe('ReviewManage');
  });
});
