import type { RouteRecordRaw } from 'vue-router';

import { $t } from '#/locales';

const routes: RouteRecordRaw[] = [
  {
    meta: {
      authority: ['iam:grants:manage'],
      icon: 'lucide:shield-keyhole',
      order: 3,
      title: $t('admin.routes.authorization.title'),
    },
    name: 'Authorization',
    path: '/authorization',
    children: [
      {
        name: 'AuthorizationGrants',
        path: '/authorization/grants',
        component: () => import('#/views/authorization/grants/index.vue'),
        meta: {
          authority: ['iam:grants:manage'],
          icon: 'lucide:network',
          title: $t('admin.routes.authorization.grants'),
        },
      },
    ],
  },
];

export default routes;
