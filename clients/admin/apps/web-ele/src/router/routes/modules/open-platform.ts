import type { RouteRecordRaw } from 'vue-router';

import { $t } from '#/locales';

const routes: RouteRecordRaw[] = [
  {
    meta: {
      authority: ['open_platform:manage'],
      icon: 'lucide:key-round',
      order: 3,
      title: $t('admin.routes.openPlatform.title'),
    },
    name: 'OpenPlatform',
    path: '/open-platform',
    children: [
      {
        name: 'OpenPlatformApps',
        path: '/open-platform/apps',
        component: () => import('#/views/open-platform/apps/index.vue'),
        meta: {
          authority: ['open_platform:manage'],
          icon: 'lucide:app-window',
          title: $t('admin.routes.openPlatform.apps'),
        },
      },
      {
        name: 'OpenPlatformAuditEvents',
        path: '/open-platform/audit-events',
        component: () => import('#/views/open-platform/audit-events/index.vue'),
        meta: {
          authority: ['open_platform:manage'],
          icon: 'lucide:scroll-text',
          title: $t('admin.routes.openPlatform.auditEvents'),
        },
      },
      {
        name: 'OpenPlatformConsents',
        path: '/open-platform/consents',
        component: () => import('#/views/open-platform/consents/index.vue'),
        meta: {
          authority: ['open_platform:manage'],
          icon: 'lucide:user-check',
          title: $t('admin.routes.openPlatform.consents'),
        },
      },
      {
        name: 'OpenPlatformTokenProbeEvidence',
        path: '/open-platform/token-probe-evidence',
        component: () =>
          import('#/views/open-platform/token-probe-evidence/index.vue'),
        meta: {
          authority: ['open_platform:manage'],
          icon: 'lucide:shield-check',
          title: $t('admin.routes.openPlatform.tokenProbeEvidence'),
        },
      },
      {
        name: 'OpenPlatformDisclosureReport',
        path: '/open-platform/disclosure-report',
        component: () =>
          import('#/views/open-platform/disclosure-report/index.vue'),
        meta: {
          authority: ['open_platform:manage'],
          icon: 'lucide:activity',
          title: $t('admin.routes.openPlatform.disclosureReport'),
        },
      },
    ],
  },
];

export default routes;
