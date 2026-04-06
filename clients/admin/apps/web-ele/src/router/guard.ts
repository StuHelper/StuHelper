import type { Router } from 'vue-router';

import { LOGIN_PATH } from '@vben/constants';
import { preferences } from '@vben/preferences';
import { useAccessStore, useUserStore } from '@vben/stores';
import { startProgress, stopProgress } from '@vben/utils';

import { accessRoutes, coreRouteNames } from '#/router/routes';
import { useAuthStore } from '#/store';

import { generateAccess } from './access';

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
 * StuHelper 适配：使用 Zitadel OIDC SSO，无 token 时尝试从 /auth/me 获取会话，
 * 失败则重定向到 Zitadel 登录页（不走 Vben 内置登录页）。
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

    // 明确声明忽略权限
    if (to.meta.ignoreAccess) {
      return true;
    }

    // 没有 token 时，尝试通过 Cookie 从后端获取会话
    if (!accessStore.accessToken) {
      const userInfo = await authStore.initSession();
      if (!userInfo) {
        if (authStore.sessionForbidden) {
          return {
            path: '/auth/login',
            query: { error: 'forbidden' },
          };
        }

        // 会话不存在，重定向到 Zitadel OIDC 登录
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
        : (userInfo.roles ?? []);

    const { accessibleMenus, accessibleRoutes } = await generateAccess({
      roles: accessCodes,
      router,
      routes: accessRoutes,
    });

    accessStore.setAccessMenus(accessibleMenus);
    accessStore.setAccessRoutes(accessibleRoutes);
    accessStore.setIsAccessChecked(true);

    const redirectPath = (from.query.redirect ??
      (to.path === preferences.app.defaultHomePath
        ? userInfo.homePath || preferences.app.defaultHomePath
        : to.fullPath)) as string;

    return {
      ...router.resolve(decodeURIComponent(redirectPath)),
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
