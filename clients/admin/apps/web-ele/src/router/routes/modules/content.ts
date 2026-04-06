import type { RouteRecordRaw } from 'vue-router';

import { $t } from '#/locales';

const routes: RouteRecordRaw[] = [
  {
    meta: {
      icon: 'lucide:file-text',
      order: 1,
      title: $t('admin.routes.content.title'),
      authority: [
        'admin:reviews:manage',
        'admin:reports:manage',
        'admin:teachers:manage',
        'admin:sensitive_words:manage',
        'admin:logs:view',
      ],
    },
    name: 'ContentManage',
    path: '/content',
    children: [
      {
        name: 'ReviewManage',
        path: '/content/reviews',
        component: () => import('#/views/content/reviews/index.vue'),
        meta: {
          icon: 'lucide:message-square',
          title: $t('admin.routes.content.reviews'),
          authority: ['admin:reviews:manage'],
        },
      },
      {
        name: 'ReportManage',
        path: '/content/reports',
        component: () => import('#/views/content/reports/index.vue'),
        meta: {
          icon: 'lucide:flag',
          title: $t('admin.routes.content.reports'),
          authority: ['admin:reports:manage'],
        },
      },
      {
        name: 'TeacherManage',
        path: '/content/teachers',
        component: () => import('#/views/content/teachers/index.vue'),
        meta: {
          icon: 'lucide:graduation-cap',
          title: $t('admin.routes.content.teachers'),
          authority: ['admin:teachers:manage'],
        },
      },
      {
        name: 'SensitiveWords',
        path: '/content/sensitive-words',
        component: () => import('#/views/content/sensitive-words/index.vue'),
        meta: {
          icon: 'lucide:shield-alert',
          title: $t('admin.routes.content.sensitiveWords'),
          authority: ['admin:sensitive_words:manage'],
        },
      },
      {
        name: 'OperationLogs',
        path: '/content/logs',
        component: () => import('#/views/content/logs/index.vue'),
        meta: {
          icon: 'lucide:scroll-text',
          title: $t('admin.routes.content.logs'),
          authority: ['admin:logs:view'],
        },
      },
    ],
  },
];

export default routes;
