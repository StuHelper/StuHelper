import type { UserInfo } from '@vben/types';

import type { AdminLoginRedirectOptions, AuthApi } from '#/api/core/auth';

import { ref } from 'vue';
import { useRouter } from 'vue-router';

import { preferences } from '@vben/preferences';
import { resetAllStores, useAccessStore, useUserStore } from '@vben/stores';

import { defineStore } from 'pinia';

import {
  getAccountSettingsUrl,
  logoutApi,
  redirectToOIDCLogin,
  tryGetMe,
} from '#/api/core/auth';
import { getUserInfoApi, mapMeToUserInfo } from '#/api/core/user';
import { $t } from '#/locales';
import { adminLogger } from '#/utils/admin-logger';

import { showAuthNotification } from './auth-notification';

export const SCHOOL_SCOPE_REQUIRED_ERROR = 'school scope required';

class SessionBootstrapError extends Error {}

function hasGlobalCapabilityGrant(
  grants: AuthApi.MeResult['capabilityGrants'],
  capabilityName: string,
) {
  return grants.some((grant) => grant.name === capabilityName && grant.global);
}

function getScopedSchoolIdsForCapability(
  grants: AuthApi.MeResult['capabilityGrants'],
  capabilityName: string,
) {
  const schoolIds = new Set<string>();

  for (const grant of grants) {
    if (grant.global || grant.name !== capabilityName) {
      continue;
    }

    for (const schoolId of grant.scopeSchoolIDs ?? []) {
      const trimmed = schoolId.trim();
      if (trimmed !== '') {
        schoolIds.add(trimmed);
      }
    }
  }

  return [...schoolIds];
}

export const useAuthStore = defineStore('auth', () => {
  const accessStore = useAccessStore();
  const userStore = useUserStore();
  const router = useRouter();

  const loginLoading = ref(false);
  const sessionForbidden = ref(false);
  const capabilityGrants = ref<AuthApi.MeResult['capabilityGrants']>([]);
  /** 后端返回的身份提供方账户设置页地址。 */
  const accountSettingsUrl = ref('');

  function syncCapabilityGrants(me: AuthApi.MeResult) {
    capabilityGrants.value = me.capabilityGrants ?? [];
  }

  function clearCapabilityGrants() {
    capabilityGrants.value = [];
  }

  function resolveScopedSchoolId(
    capabilityName: string,
    requestedSchoolId: string,
  ): string {
    const trimmed = requestedSchoolId.trim();
    if (trimmed !== '') {
      return trimmed;
    }

    if (hasGlobalCapabilityGrant(capabilityGrants.value, capabilityName)) {
      return '';
    }

    const scopedSchoolIds = getScopedSchoolIdsForCapability(
      capabilityGrants.value,
      capabilityName,
    );
    if (scopedSchoolIds.length === 1) {
      return scopedSchoolIds[0] ?? '';
    }
    if (scopedSchoolIds.length > 1) {
      throw new Error(SCHOOL_SCOPE_REQUIRED_ERROR);
    }
    return '';
  }

  /**
   * OIDC 登录：调用后端获取授权 URL，然后浏览器跳转到外部身份提供方
   */
  async function redirectToLogin(
    redirectPath?: string,
    options?: AdminLoginRedirectOptions,
  ) {
    await redirectToOIDCLogin(redirectPath, options);
  }

  function clearLocalSessionState() {
    sessionForbidden.value = false;
    resetAllStores();
    accessStore.setLoginExpired(false);
    clearCapabilityGrants();
    accountSettingsUrl.value = '';
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
        clearCapabilityGrants();
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
        showAuthNotification({
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

      // StuHelper 使用基于 Cookie 的 OIDC 会话。
      // Vben 默认 auth 流程仍要求存在一个 truthy token，因此这里用
      // `cookie-session` 作为占位值，仅用于通过其“已登录”守卫判断。
      const userInfo = mapMeToUserInfo(me);

      userStore.setUserInfo(userInfo);
      accessStore.setAccessToken('cookie-session');
      accessStore.setAccessCodes(me.capabilities);
      syncCapabilityGrants(me);
      accountSettingsUrl.value = getAccountSettingsUrl(me);

      if (accessStore.loginExpired) {
        accessStore.setLoginExpired(false);
      }

      if (userInfo.realName) {
        showAuthNotification({
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
      adminLogger.warn('initSession failed unexpectedly', error);
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
      showAuthNotification({
        message: $t(result.message),
        title: $t('common.logout'),
        type: 'error',
      });
      throw new Error($t(result.message));
    }

    clearLocalSessionState();

    if (redirect) {
      await redirectToLogin(router.currentRoute.value.fullPath);
    }
  }

  async function switchAccount(
    redirectPath: string = preferences.app.defaultHomePath,
  ) {
    try {
      const result = await logoutApi();
      if (result.kind === 'error') {
        adminLogger.warn(
          'admin switch account local logout returned an error before forced re-auth',
          result.message,
        );
      }
    } catch (error) {
      adminLogger.warn(
        'admin switch account local logout failed before forced re-auth',
        error,
      );
    }

    clearLocalSessionState();
    await redirectToLogin(redirectPath, { forceReauth: true });
  }

  async function fetchUserInfo() {
    const { userInfo, me } = await getUserInfoApi();
    userStore.setUserInfo(userInfo);
    accessStore.setAccessCodes(me.capabilities);
    syncCapabilityGrants(me);
    accountSettingsUrl.value = getAccountSettingsUrl(me);
    return userInfo;
  }

  function $reset() {
    loginLoading.value = false;
    sessionForbidden.value = false;
    clearCapabilityGrants();
    accountSettingsUrl.value = '';
  }

  return {
    $reset,
    accountSettingsUrl,
    authLogin,
    capabilityGrants,
    fetchUserInfo,
    initSession,
    loginLoading,
    logout,
    resolveScopedSchoolId,
    sessionForbidden,
    switchAccount,
    redirectToLogin,
  };
});
