import type { RouteRecordRaw } from 'vue-router';

import { $t } from '#/locales';

const routes: RouteRecordRaw[] = [
  {
    meta: {
      icon: 'lucide:users',
      order: 2,
      title: $t('admin.routes.userSystem.title'),
      authority: ['user:identity:read'],
    },
    name: 'UserSystem',
    path: '/users',
    children: [
      {
        name: 'IdentityReview',
        path: '/users/identity-review',
        component: () => import('#/views/users/identity-review/index.vue'),
        meta: {
          icon: 'lucide:id-card',
          title: $t('admin.routes.userSystem.identityReview'),
          authority: ['user:identity:review'],
        },
      },
      {
        name: 'StudentVerification',
        path: '/users/student-verification',
        component: () => import('#/views/users/student-verification/index.vue'),
        meta: {
          icon: 'lucide:badge-check',
          title: $t('admin.routes.userSystem.studentVerification'),
          authority: ['user:student:review'],
        },
      },
      {
        name: 'SchoolConfig',
        path: '/users/school-config',
        component: () => import('#/views/users/school-config/index.vue'),
        meta: {
          icon: 'lucide:school',
          title: $t('admin.routes.userSystem.schoolConfig'),
          authority: ['user:school:read'],
        },
      },
      {
        name: 'SystemConfig',
        path: '/users/system-config',
        component: () => import('#/views/users/system-config/index.vue'),
        meta: {
          icon: 'lucide:settings',
          title: $t('admin.routes.userSystem.systemConfig'),
          authority: ['user:system:read'],
        },
      },
    ],
  },
];

export default routes;
