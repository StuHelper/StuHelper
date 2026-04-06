import { defineOverridesPreferences } from '@vben/preferences';

/**
 * StuHelper Admin 配置覆盖
 */
export const overridesPreferences = defineOverridesPreferences({
  app: {
    name: import.meta.env.VITE_APP_TITLE,
    locale: 'zh-CN',
    loginExpiredMode: 'page',
    enableRefreshToken: true,
    defaultHomePath: '/analytics',
  },
  sidebar: {
    collapsed: false,
  },
});
