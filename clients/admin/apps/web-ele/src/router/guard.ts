import type { Router } from 'vue-router';

import { LOGIN_PATH } from '@vben/constants';
import { preferences } from '@vben/preferences';
import { useAccessStore, useUserStore } from '@vben/stores';
import { startProgress, stopProgress } from '@vben/utils';

import { getAccessCodesApi } from '#/api/core/user';
import { accessRoutes, coreRouteNames } from '#/router/routes';
import { useAuthStore } from '#/store';

import { generateAccess } from './access';
import {
  collectAbsoluteRoutePaths,
  resolveAccessibleRedirect,
  resolveHomePath,
} from './route-resolution';

const fallbackNotFoundRouteName = 'FallbackNotFound';

function normalizeRoutePath(path: string) {
  if (!path || path === '/') {
    return '/';
  }
  return path.endsWith('/') ? path.slice(0, -1) : path;
}

const protectedRoutePaths = collectAbsoluteRoutePaths(accessRoutes);

function isKnownProtectedRoutePath(path: string) {
  const normalizedPath = normalizeRoutePath(path);
  for (const protectedPath of protectedRoutePaths) {
    if (
      protectedPath !== '/' &&
      (normalizedPath === protectedPath ||
        normalizedPath.startsWith(`${protectedPath}/`))
    ) {
      return true;
    }
  }
  return false;
}

/**
 * 通用守卫配置
 */
function setupCommonGuard(router: Router) {
  const loadedPaths = new Set<string>();

  router.beforeEach((to) => {
    to.meta.loaded = loadedPaths.has(to.path);
    if (!to.meta.loaded && preferences.transition.progress) {
      startProgress();
    }
    return true;
  });

  router.afterEach((to) => {
    loadedPaths.add(to.path);
    if (preferences.transition.progress) {
      stopProgress();
    }
  });
}

/**
 * 权限访问守卫配置
 *
 * StuHelper 适配：使用外部 OIDC SSO，无 token 时尝试从 /auth/me 获取会话，
 * 失败则重定向到 OIDC 登录页（不走 Vben 内置登录页）。
 */
function setupAccessGuard(router: Router) {
  router.beforeEach(async (to, from) => {
    const accessStore = useAccessStore();
    const userStore = useUserStore();
    const authStore = useAuthStore();

    // 基本路由不拦截（登录页、404 等）
    if (coreRouteNames.includes(to.name as string)) {
      if (to.path === LOGIN_PATH && accessStore.accessToken) {
        return decodeURIComponent(
          (to.query?.redirect as string) ||
            userStore.userInfo?.homePath ||
            preferences.app.defaultHomePath,
        );
      }
      return true;
    }

    // 未知路径命中 catch-all 时直接显示 404；已知受保护路由仍继续走登录/权限流程。
    if (
      to.name === fallbackNotFoundRouteName &&
      !isKnownProtectedRoutePath(to.path)
    ) {
      return true;
    }

    // 明确声明忽略权限
    if (to.meta.ignoreAccess) {
      return true;
    }

    // 没有 token 时，尝试通过 Cookie 从后端获取会话
    if (!accessStore.accessToken) {
      const sessionUser = await authStore.initSession().catch((error) => {
        console.warn(
          '[admin-auth] access guard session bootstrap failed',
          error,
        );
        return false as const;
      });

      if (sessionUser === false) {
        return false;
      }

      if (!sessionUser) {
        if (authStore.sessionForbidden) {
          return {
            path: '/auth/login',
            query: { error: 'forbidden' },
          };
        }

        // 会话不存在，重定向到 OIDC 登录
        authStore.redirectToLogin(to.fullPath);
        return false;
      }
    }

    // 已经生成过动态路由
    if (accessStore.isAccessChecked) {
      return true;
    }

    // 生成路由表
    const userInfo = userStore.userInfo || (await authStore.fetchUserInfo());
    const accessCodes =
      accessStore.accessCodes.length > 0
        ? accessStore.accessCodes
        : await getAccessCodesApi();

    if (accessStore.accessCodes.length === 0) {
      accessStore.setAccessCodes(accessCodes);
    }

    const { accessibleMenus, accessibleRoutes } = await generateAccess({
      roles: accessCodes,
      router,
      routes: accessRoutes,
    });

    accessStore.setAccessMenus(accessibleMenus);
    accessStore.setAccessRoutes(accessibleRoutes);
    accessStore.setIsAccessChecked(true);

    const homePath = resolveHomePath(
      accessibleRoutes,
      accessibleMenus,
      preferences.app.defaultHomePath,
    );
    if (userInfo.homePath !== homePath) {
      userStore.setUserInfo({ ...userInfo, homePath });
    }

    const redirectCandidate =
      from.query.redirect ??
      (to.path === preferences.app.defaultHomePath ? homePath : to.fullPath);
    const redirect = resolveAccessibleRedirect(
      router,
      redirectCandidate,
      accessibleRoutes,
      homePath,
    );

    return {
      ...redirect,
      replace: true,
    };
  });
}

/**
 * 项目守卫配置
 */
function createRouterGuard(router: Router) {
  setupCommonGuard(router);
  setupAccessGuard(router);
}

export { createRouterGuard };
