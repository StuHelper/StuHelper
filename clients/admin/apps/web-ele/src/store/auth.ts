import type { UserInfo } from '@vben/types';

import { ref } from 'vue';
import { useRouter } from 'vue-router';

import { preferences } from '@vben/preferences';
import { resetAllStores, useAccessStore, useUserStore } from '@vben/stores';

import { ElNotification } from 'element-plus';
import { defineStore } from 'pinia';

import {
  getAccountSettingsUrl,
  logoutApi,
  redirectToOIDCLogin,
  tryGetMe,
} from '#/api/core/auth';
import { getUserInfoApi, mapMeToUserInfo } from '#/api/core/user';
import { $t } from '#/locales';

class SessionBootstrapError extends Error {}

export const useAuthStore = defineStore('auth', () => {
  const accessStore = useAccessStore();
  const userStore = useUserStore();
  const router = useRouter();

  const loginLoading = ref(false);
  const sessionForbidden = ref(false);
  /** 后端返回的身份提供方账户设置页地址。 */
  const accountSettingsUrl = ref('');

  /**
   * OIDC 登录：调用后端获取授权 URL，然后浏览器跳转到 Zitadel
   */
  async function redirectToLogin(redirectPath?: string) {
    await redirectToOIDCLogin(redirectPath);
  }

  /**
   * 尝试初始化会话（用 tryGetMe，不触发 token 刷新和 CSRF）
   * 首次无 session 时静默返回 null，不报错
   */
  async function initSession(): Promise<null | UserInfo> {
    try {
      loginLoading.value = true;
      sessionForbidden.value = false;

      // 先用 baseRequestClient 探测 session 是否存在
      const probe = await tryGetMe();
      const clearAccess = () => {
        userStore.setUserInfo(null);
        accessStore.setAccessToken(null);
        accessStore.setAccessCodes([]);
        accessStore.setAccessMenus([]);
        accessStore.setAccessRoutes([]);
        accessStore.setIsAccessChecked(false);
      };

      if (probe.kind === 'unauthenticated') {
        clearAccess();
        return null;
      }

      if (probe.kind === 'forbidden') {
        sessionForbidden.value = true;
        clearAccess();
        return null;
      }

      if (probe.kind === 'retryable_error' || probe.kind === 'fatal_error') {
        clearAccess();
        ElNotification({
          message: $t(probe.message),
          title: $t('authentication.loginFailed'),
          type: 'error',
        });
        throw new SessionBootstrapError($t(probe.message));
      }

      const me = probe.me;
      if (!me.canAccessAdmin) {
        sessionForbidden.value = true;
        clearAccess();
        return null;
      }

      // StuHelper 使用基于 Cookie 的 Zitadel OIDC 会话。
      // Vben 默认 auth 流程仍要求存在一个 truthy token，因此这里用
      // `cookie-session` 作为占位值，仅用于通过其“已登录”守卫判断。
      const userInfo = mapMeToUserInfo(me);

      userStore.setUserInfo(userInfo);
      accessStore.setAccessToken('cookie-session');
      accessStore.setAccessCodes(
        me.globalCapabilities?.length ? me.globalCapabilities : me.capabilities,
      );
      accountSettingsUrl.value = getAccountSettingsUrl(me);

      if (accessStore.loginExpired) {
        accessStore.setLoginExpired(false);
      }

      if (userInfo.realName) {
        ElNotification({
          message: `${$t('authentication.loginSuccessDesc')}:${userInfo.realName}`,
          title: $t('authentication.loginSuccess'),
          type: 'success',
        });
      }

      return userInfo;
    } catch (error) {
      if (error instanceof SessionBootstrapError) {
        throw error;
      }
      console.warn('[admin-auth] initSession failed unexpectedly', error);
      throw error;
    } finally {
      loginLoading.value = false;
    }
  }

  /**
   * 兼容 Vben 原有 authLogin 接口
   */
  async function authLogin(
    _params: Record<string, unknown>,
    onSuccess?: () => Promise<void> | void,
  ) {
    const userInfo = await initSession();
    if (sessionForbidden.value) {
      await router.replace({
        path: '/auth/login',
        query: { error: 'forbidden' },
      });
      return { userInfo: null };
    }
    if (userInfo) {
      onSuccess
        ? await onSuccess()
        : await router.push(
            userInfo.homePath || preferences.app.defaultHomePath,
          );
    }
    return { userInfo };
  }

  async function logout(redirect: boolean = true) {
    const result = await logoutApi();
    if (result.kind === 'error') {
      ElNotification({
        message: $t(result.message),
        title: $t('common.logout'),
        type: 'error',
      });
      throw new Error($t(result.message));
    }

    sessionForbidden.value = false;
    resetAllStores();
    accessStore.setLoginExpired(false);

    if (redirect) {
      await redirectToLogin(router.currentRoute.value.fullPath);
    }
  }

  async function fetchUserInfo() {
    const { userInfo, me } = await getUserInfoApi();
    userStore.setUserInfo(userInfo);
    accessStore.setAccessCodes(
      me.globalCapabilities?.length ? me.globalCapabilities : me.capabilities,
    );
    accountSettingsUrl.value = getAccountSettingsUrl(me);
    return userInfo;
  }

  function $reset() {
    loginLoading.value = false;
    sessionForbidden.value = false;
    accountSettingsUrl.value = '';
  }

  return {
    $reset,
    accountSettingsUrl,
    authLogin,
    fetchUserInfo,
    initSession,
    loginLoading,
    logout,
    sessionForbidden,
    redirectToLogin,
  };
});
