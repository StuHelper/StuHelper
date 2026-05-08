import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { useAuthStore } from './auth';

const mocks = vi.hoisted(() => ({
  ElNotification: vi.fn(),
  accessStore: null as any,
  getAccountSettingsUrl: vi.fn(() => ''),
  getUserInfoApi: vi.fn(),
  logoutApi: vi.fn(),
  mapMeToUserInfo: vi.fn(),
  redirectToOIDCLogin: vi.fn(),
  resetAllStores: vi.fn(),
  router: null as any,
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

vi.mock('element-plus', () => ({
  ElNotification: mocks.ElNotification,
}));

vi.mock('#/locales', () => ({
  $t: (value: string) => value,
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
    mocks.ElNotification.mockReset();
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
    expect(mocks.ElNotification).toHaveBeenCalledTimes(1);
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

  it('uses full capabilities instead of globalCapabilities during session bootstrap', async () => {
    const me = {
      capabilities: ['user:school:read'],
      globalCapabilities: [],
      canAccessAdmin: true,
    };
    mocks.tryGetMe.mockResolvedValue({ kind: 'authenticated', me });
    mocks.mapMeToUserInfo.mockReturnValue({ realName: 'School Admin' });

    const store = useAuthStore();

    await store.initSession();

    expect(mocks.accessStore.accessCodes).toEqual(['user:school:read']);
  });

  it('uses full capabilities instead of globalCapabilities when refreshing user info', async () => {
    mocks.getUserInfoApi.mockResolvedValue({
      userInfo: { realName: 'School Admin' },
      me: {
        capabilities: ['user:school:read'],
        globalCapabilities: [],
      },
    });

    const store = useAuthStore();

    await store.fetchUserInfo();

    expect(mocks.accessStore.accessCodes).toEqual(['user:school:read']);
  });

  it('defaults blank student verification school filters to the first allowed scoped school', async () => {
    const me = {
      capabilities: ['user:student:review'],
      globalCapabilities: [],
      capabilityGrants: [
        {
          name: 'user:student:review',
          global: false,
          scopeSchoolIDs: ['1001', '1002'],
        },
      ],
      canAccessAdmin: true,
    };
    mocks.tryGetMe.mockResolvedValue({ kind: 'authenticated', me });
    mocks.mapMeToUserInfo.mockReturnValue({ realName: 'School Admin' });

    const store = useAuthStore();

    await store.initSession();

    expect(store.resolveScopedSchoolId('user:student:review', '')).toBe('1001');
    expect(store.resolveScopedSchoolId('user:student:review', '1002')).toBe(
      '1002',
    );
  });

  it('keeps blank school filters for global student verification admins', async () => {
    mocks.getUserInfoApi.mockResolvedValue({
      userInfo: { realName: 'Platform Admin' },
      me: {
        capabilities: ['user:student:review'],
        globalCapabilities: ['user:student:review'],
        capabilityGrants: [
          {
            name: 'user:student:review',
            global: true,
          },
        ],
      },
    });

    const store = useAuthStore();

    await store.fetchUserInfo();

    expect(store.resolveScopedSchoolId('user:student:review', '')).toBe('');
  });
});
