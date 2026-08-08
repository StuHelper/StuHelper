// @vitest-environment happy-dom

import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => ({
  accessStore: null as any,
  getAccountSettingsUrl: vi.fn(() => ''),
  getUserInfoApi: vi.fn(),
  logoutApi: vi.fn(),
  mapMeToUserInfo: vi.fn(),
  adminLogger: {
    warn: vi.fn(),
  },
  redirectToOIDCLogin: vi.fn(),
  resetAllStores: vi.fn(),
  router: null as any,
  showAuthNotification: vi.fn(),
  tryGetMe: vi.fn(),
  userStore: null as any,
}));

vi.mock('#/api/core/auth', () => ({
  getAccountSettingsUrl: mocks.getAccountSettingsUrl,
  logoutApi: mocks.logoutApi,
  redirectToOIDCLogin: mocks.redirectToOIDCLogin,
  tryGetMe: mocks.tryGetMe,
}));

vi.mock('#/api/core/user', () => ({
  getUserInfoApi: mocks.getUserInfoApi,
  mapMeToUserInfo: mocks.mapMeToUserInfo,
}));

vi.mock('./auth-notification', () => ({
  showAuthNotification: mocks.showAuthNotification,
}));

vi.mock('#/locales', () => ({
  $t: (value: string) => value,
}));

vi.mock('#/utils/admin-logger', () => ({
  adminLogger: mocks.adminLogger,
}));

vi.mock('vue-router', () => ({
  useRouter: () => mocks.router,
}));

vi.mock('@vben/preferences', () => ({
  preferences: {
    app: {
      defaultHomePath: '/dashboard',
    },
  },
}));

vi.mock('@vben/stores', () => ({
  resetAllStores: mocks.resetAllStores,
  useAccessStore: () => mocks.accessStore,
  useUserStore: () => mocks.userStore,
}));

const { SCHOOL_SCOPE_REQUIRED_ERROR, useAuthStore } = await import('./auth');

function createAccessStoreState() {
  const state: any = {
    accessCodes: ['legacy-code'],
    accessMenus: ['legacy-menu'],
    accessRoutes: ['legacy-route'],
    accessToken: 'legacy-token',
    isAccessChecked: true,
    loginExpired: false,
    setAccessCodes(value: string[]) {
      this.accessCodes = value;
    },
    setAccessMenus(value: unknown[]) {
      this.accessMenus = value;
    },
    setAccessRoutes(value: unknown[]) {
      this.accessRoutes = value;
    },
    setAccessToken(value: null | string) {
      this.accessToken = value;
    },
    setIsAccessChecked(value: boolean) {
      this.isAccessChecked = value;
    },
    setLoginExpired(value: boolean) {
      this.loginExpired = value;
    },
  };
  return state;
}

function createUserStoreState() {
  const state: any = {
    userInfo: { homePath: '/dashboard' },
    setUserInfo(value: unknown) {
      this.userInfo = value;
    },
  };
  return state;
}

describe('useAuthStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    mocks.accessStore = createAccessStoreState();
    mocks.userStore = createUserStoreState();
    mocks.router = {
      currentRoute: { value: { fullPath: '/admin/reviews' } },
      push: vi.fn(),
      replace: vi.fn(),
    };

    mocks.tryGetMe.mockReset();
    mocks.redirectToOIDCLogin.mockReset();
    mocks.logoutApi.mockReset();
    mocks.getAccountSettingsUrl.mockReset();
    mocks.getAccountSettingsUrl.mockReturnValue('');
    mocks.getUserInfoApi.mockReset();
    mocks.mapMeToUserInfo.mockReset();
    mocks.showAuthNotification.mockReset();
    mocks.adminLogger.warn.mockReset();
    mocks.resetAllStores.mockReset();
  });

  it('propagates retryable session probe failures instead of treating them as unauthenticated', async () => {
    mocks.tryGetMe.mockResolvedValue({
      kind: 'retryable_error',
      message: 'admin.result.requestFailed',
    });

    const store = useAuthStore();

    await expect(store.initSession()).rejects.toThrow(
      'admin.result.requestFailed',
    );
    expect(mocks.accessStore.accessToken).toBeNull();
    expect(mocks.accessStore.accessCodes).toEqual([]);
    expect(mocks.accessStore.accessMenus).toEqual([]);
    expect(mocks.accessStore.accessRoutes).toEqual([]);
    expect(mocks.accessStore.isAccessChecked).toBe(false);
    expect(mocks.userStore.userInfo).toBeNull();
    expect(mocks.showAuthNotification).toHaveBeenCalledTimes(1);
  });

  it('does not clear local session or redirect when server logout fails', async () => {
    mocks.logoutApi.mockResolvedValue({
      kind: 'error',
      message: 'admin.result.requestFailed',
    });

    const store = useAuthStore();

    await expect(store.logout()).rejects.toThrow('admin.result.requestFailed');
    expect(mocks.resetAllStores).not.toHaveBeenCalled();
    expect(mocks.redirectToOIDCLogin).not.toHaveBeenCalled();
    expect(mocks.accessStore.accessToken).toBe('legacy-token');
    expect(mocks.userStore.userInfo).toEqual({ homePath: '/dashboard' });
  });

  it('switches accounts with forced upstream reauthentication', async () => {
    mocks.logoutApi.mockResolvedValue({ kind: 'ok' });

    const store = useAuthStore();
    store.sessionForbidden = true;

    await store.switchAccount('/dashboard');

    expect(mocks.logoutApi).toHaveBeenCalledTimes(1);
    expect(mocks.resetAllStores).toHaveBeenCalledTimes(1);
    expect(mocks.accessStore.loginExpired).toBe(false);
    expect(store.sessionForbidden).toBe(false);
    expect(mocks.redirectToOIDCLogin).toHaveBeenCalledWith('/dashboard', {
      forceReauth: true,
    });
  });

  it('continues forced account switching when local logout returns an error', async () => {
    mocks.logoutApi.mockResolvedValue({
      kind: 'error',
      message: 'admin.result.requestFailed',
    });

    const store = useAuthStore();

    await store.switchAccount('/dashboard');

    expect(mocks.adminLogger.warn).toHaveBeenCalledWith(
      'admin switch account local logout returned an error before forced re-auth',
      'admin.result.requestFailed',
    );
    expect(mocks.resetAllStores).toHaveBeenCalledTimes(1);
    expect(mocks.redirectToOIDCLogin).toHaveBeenCalledWith('/dashboard', {
      forceReauth: true,
    });
  });

  it('continues forced account switching when local logout rejects', async () => {
    const logoutError = new Error('network failed');
    mocks.logoutApi.mockRejectedValue(logoutError);

    const store = useAuthStore();

    await store.switchAccount('/dashboard');

    expect(mocks.adminLogger.warn).toHaveBeenCalledWith(
      'admin switch account local logout failed before forced re-auth',
      logoutError,
    );
    expect(mocks.resetAllStores).toHaveBeenCalledTimes(1);
    expect(mocks.redirectToOIDCLogin).toHaveBeenCalledWith('/dashboard', {
      forceReauth: true,
    });
  });

  it('uses full capabilities instead of globalCapabilities during session bootstrap', async () => {
    const me = {
      capabilities: ['student:verification_config:read'],
      globalCapabilities: [],
      canAccessAdmin: true,
    };
    mocks.tryGetMe.mockResolvedValue({ kind: 'authenticated', me });
    mocks.mapMeToUserInfo.mockReturnValue({ realName: 'School Admin' });

    const store = useAuthStore();

    await store.initSession();

    expect(mocks.accessStore.accessCodes).toEqual([
      'student:verification_config:read',
    ]);
    expect(mocks.showAuthNotification).not.toHaveBeenCalled();
  });

  it('uses full capabilities instead of globalCapabilities when refreshing user info', async () => {
    mocks.getUserInfoApi.mockResolvedValue({
      userInfo: { realName: 'School Admin' },
      me: {
        capabilities: ['student:verification_config:read'],
        globalCapabilities: [],
      },
    });

    const store = useAuthStore();

    await store.fetchUserInfo();

    expect(mocks.accessStore.accessCodes).toEqual([
      'student:verification_config:read',
    ]);
  });

  it('defaults blank student verification school filters when only one scoped school is allowed', async () => {
    const me = {
      capabilities: ['student:manual_review:decide'],
      globalCapabilities: [],
      capabilityGrants: [
        {
          name: 'student:manual_review:decide',
          global: false,
          scopeSchoolIDs: ['4111010001'],
        },
      ],
      canAccessAdmin: true,
    };
    mocks.tryGetMe.mockResolvedValue({ kind: 'authenticated', me });
    mocks.mapMeToUserInfo.mockReturnValue({ realName: 'School Admin' });

    const store = useAuthStore();

    await store.initSession();

    expect(
      store.resolveScopedSchoolId('student:manual_review:decide', ''),
    ).toBe('4111010001');
    expect(
      store.resolveScopedSchoolId('student:manual_review:decide', '4111010002'),
    ).toBe('4111010002');
  });

  it('requires explicit school filters for multi-school scoped admins', async () => {
    const me = {
      capabilities: ['student:manual_review:decide'],
      globalCapabilities: [],
      capabilityGrants: [
        {
          name: 'student:manual_review:decide',
          global: false,
          scopeSchoolIDs: ['4111010001', '4111010002'],
        },
      ],
      canAccessAdmin: true,
    };
    mocks.tryGetMe.mockResolvedValue({ kind: 'authenticated', me });
    mocks.mapMeToUserInfo.mockReturnValue({ realName: 'School Admin' });

    const store = useAuthStore();

    await store.initSession();

    expect(() =>
      store.resolveScopedSchoolId('student:manual_review:decide', ''),
    ).toThrow(SCHOOL_SCOPE_REQUIRED_ERROR);
    expect(
      store.resolveScopedSchoolId('student:manual_review:decide', '4111010002'),
    ).toBe('4111010002');
  });

  it('keeps blank school filters for global student verification admins', async () => {
    mocks.getUserInfoApi.mockResolvedValue({
      userInfo: { realName: 'Platform Admin' },
      me: {
        capabilities: ['student:manual_review:decide'],
        globalCapabilities: ['student:manual_review:decide'],
        capabilityGrants: [
          {
            name: 'student:manual_review:decide',
            global: true,
          },
        ],
      },
    });

    const store = useAuthStore();

    await store.fetchUserInfo();

    expect(
      store.resolveScopedSchoolId('student:manual_review:decide', ''),
    ).toBe('');
  });
});
