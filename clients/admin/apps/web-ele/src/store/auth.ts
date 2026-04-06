import type { UserInfo } from '@vben/types';

import { ref } from 'vue';
import { useRouter } from 'vue-router';

import { preferences } from '@vben/preferences';
import { resetAllStores, useAccessStore, useUserStore } from '@vben/stores';

import { ElNotification } from 'element-plus';
import { defineStore } from 'pinia';

import { logoutApi, redirectToOIDCLogin, tryGetMe } from '#/api/core/auth';
import { getUserInfoApi } from '#/api/core/user';
import { $t } from '#/locales';

export const useAuthStore = defineStore('auth', () => {
  const accessStore = useAccessStore();
  const userStore = useUserStore();
  const router = useRouter();

  const loginLoading = ref(false);
  const sessionForbidden = ref(false);

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
      const me = await tryGetMe();
      if (!me) {
        userStore.setUserInfo(null);
        accessStore.setAccessToken(null);
        accessStore.setAccessCodes([]);
        accessStore.setAccessMenus([]);
        accessStore.setAccessRoutes([]);
        accessStore.setIsAccessChecked(false);
        return null;
      }

      if (!me.canAccessAdmin) {
        sessionForbidden.value = true;
        accessStore.setAccessToken(null);
        accessStore.setAccessCodes([]);
        accessStore.setAccessMenus([]);
        accessStore.setAccessRoutes([]);
        accessStore.setIsAccessChecked(false);
        userStore.setUserInfo(null);
        return null;
      }

      // session 存在，构造 UserInfo
      const userInfo: UserInfo = {
        userId: String(me.id),
        username: me.name,
        realName: me.displayName,
        avatar: me.avatar ?? '',
        desc: '',
        token: '',
        roles: me.roles,
        homePath: preferences.app.defaultHomePath,
      };

      userStore.setUserInfo(userInfo);
      accessStore.setAccessToken('cookie-session');
      accessStore.setAccessCodes(
        me.globalCapabilities?.length ? me.globalCapabilities : me.capabilities,
      );

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
    } catch {
      return null;
    } finally {
      loginLoading.value = false;
    }
  }

  /**
   * 兼容 Vben 原有 authLogin 接口
   */
  async function authLogin(
    _params: Record<string, any>,
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
    try {
      await logoutApi();
    } catch {
      // ignore
    }
    sessionForbidden.value = false;
    resetAllStores();
    accessStore.setLoginExpired(false);

    if (redirect) {
      await redirectToLogin(router.currentRoute.value.fullPath);
    }
  }

  async function fetchUserInfo() {
    const userInfo = await getUserInfoApi();
    userStore.setUserInfo(userInfo);
    return userInfo;
  }

  function $reset() {
    loginLoading.value = false;
  }

  return {
    $reset,
    authLogin,
    fetchUserInfo,
    initSession,
    loginLoading,
    logout,
    sessionForbidden,
    redirectToLogin,
  };
});
