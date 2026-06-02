import type { RouteRecordRaw } from 'vue-router';

import { $t } from '#/locales';

const routes: RouteRecordRaw[] = [
  {
    meta: {
      icon: 'lucide:users',
      order: 2,
      title: $t('admin.routes.userSystem.title'),
      authority: [
        'user:identity:read',
        'user:identity:review',
        'user:student:review',
        'user:school:read',
        'user:system:read',
        'admission:session:read',
        'admission:session:manage',
        'admission:freshman:review',
        'admission:policy:update',
        'member_blacklist:read',
        'member_blacklist:manage',
      ],
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
        name: 'AdmissionSessions',
        path: '/users/admission-sessions',
        component: () => import('#/views/users/admission-sessions/index.vue'),
        meta: {
          icon: 'lucide:list-checks',
          title: $t('admin.routes.userSystem.admissionSessions'),
          authority: ['admission:session:read'],
        },
      },
      {
        name: 'FreshmanVerification',
        path: '/users/freshman-verification',
        component: () =>
          import('#/views/users/freshman-verification/index.vue'),
        meta: {
          icon: 'lucide:file-check-2',
          title: $t('admin.routes.userSystem.freshmanVerification'),
          authority: ['admission:freshman:review'],
        },
      },
      {
        name: 'AdmissionPolicy',
        path: '/users/admission-policy',
        component: () => import('#/views/users/admission-policy/index.vue'),
        meta: {
          icon: 'lucide:shield-check',
          title: $t('admin.routes.userSystem.admissionPolicy'),
          authority: ['admission:policy:update'],
        },
      },
      {
        name: 'MemberBlacklist',
        path: '/users/member-blacklist',
        component: () => import('#/views/users/member-blacklist/index.vue'),
        meta: {
          icon: 'lucide:user-x',
          title: $t('admin.routes.userSystem.memberBlacklist'),
          authority: ['member_blacklist:read', 'member_blacklist:manage'],
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
