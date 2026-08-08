import type { RouteRecordRaw } from 'vue-router';

import { $t } from '#/locales';

const routes: RouteRecordRaw[] = [
  {
    meta: {
      icon: 'lucide:users',
      order: 2,
      title: $t('admin.routes.userSystem.title'),
      authority: [
        'student:manual_review:read',
        'student:manual_review:decide',
        'student:verification_config:read',
        'student:verification_config:update',
        'student:credential:read',
        'student:credential:revoke',
        'student:subject_conflict:read',
        'student:subject_conflict:resolve',
        'student:roster:read',
        'student:roster:activate',
        'campus_connector:health:read',
        'user:system:read',
        'admission:session:read',
        'admission:session:manage',
        'admission:policy:update',
        'member_blacklist:read',
        'member_blacklist:manage',
      ],
    },
    name: 'UserSystem',
    path: '/users',
    children: [
      {
        name: 'StudentVerification',
        path: '/users/student-verification',
        component: () => import('#/views/users/student-verification/index.vue'),
        meta: {
          icon: 'lucide:badge-check',
          title: $t('admin.routes.userSystem.studentVerification'),
          authority: [
            'student:manual_review:read',
            'student:manual_review:decide',
          ],
        },
      },
      {
        name: 'SchoolConfig',
        path: '/users/school-config',
        component: () => import('#/views/users/school-config/index.vue'),
        meta: {
          icon: 'lucide:school',
          title: $t('admin.routes.userSystem.schoolConfig'),
          authority: [
            'student:verification_config:read',
            'student:verification_config:update',
            'student:roster:read',
            'student:roster:activate',
            'campus_connector:health:read',
          ],
        },
      },
      {
        name: 'StudentCredentialGovernance',
        path: '/users/student-credentials',
        component: () =>
          import('#/views/users/credential-governance/index.vue'),
        meta: {
          icon: 'lucide:shield-alert',
          title: $t('admin.routes.userSystem.studentCredentials'),
          authority: [
            'student:credential:read',
            'student:credential:revoke',
            'student:subject_conflict:read',
            'student:subject_conflict:resolve',
          ],
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
