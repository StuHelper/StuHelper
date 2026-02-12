/**
 * 路由配置
 * 教学中心门户 + 评课模块嵌套路由
 */
import { createRouter, createWebHashHistory, type RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { isTokenExpired, clearAuth } from '@/utils/auth'
import i18n from '@/i18n'

declare module 'vue-router' {
  interface RouteMeta {
    requiresAuth?: boolean
    guest?: boolean
    titleKey?: string
    layout?: 'shell' | 'none' | 'admin'
  }
}

function lazyLoad(loader: () => Promise<unknown>) {
  return () =>
    loader().catch(() => {
      return import('@/views/errors/LoadErrorPage.vue')
    })
}

const routes: RouteRecordRaw[] = [
  // 认证路由（无 shell）
  {
    path: '/login',
    name: 'login',
    component: lazyLoad(() => import('@/views/LoginPage.vue')),
    meta: { guest: true, titleKey: 'routes.login', layout: 'none' }
  },
  {
    path: '/auth/callback',
    name: 'auth-callback',
    component: lazyLoad(() => import('@/views/AuthCallbackPage.vue')),
    meta: { guest: true, titleKey: 'routes.authCallback', layout: 'none' }
  },

  // 教学中心门户
  {
    path: '/',
    name: 'teaching-hub',
    component: lazyLoad(() => import('@/views/TeachingHubPage.vue')),
    meta: { titleKey: 'routes.teachingHub' }
  },

  // 评课模块（嵌套路由）
  {
    path: '/review',
    name: 'review',
    component: lazyLoad(() => import('@/views/review/ReviewPage.vue')),
    meta: { titleKey: 'routes.review' },
    children: [
      {
        path: 'courses/:id',
        name: 'review-course-detail',
        component: lazyLoad(() => import('@/views/review/CourseDetailPage.vue')),
        meta: { titleKey: 'routes.courseDetail' }
      }
    ]
  },

  // 教师主页
  {
    path: '/teachers/:id',
    name: 'teacher-profile',
    component: lazyLoad(() => import('@/views/review/TeacherProfilePage.vue')),
    meta: { titleKey: 'routes.teacherProfile' }
  },

  // 子模块占位
  {
    path: '/teacher',
    name: 'teacher-hub',
    component: lazyLoad(() => import('@/views/ComingSoon.vue')),
    meta: { titleKey: 'routes.teacherHub' }
  },
  {
    path: '/spoc',
    name: 'spoc',
    component: lazyLoad(() => import('@/views/ComingSoon.vue')),
    meta: { titleKey: 'routes.spoc' }
  },
  {
    path: '/resource',
    name: 'resource',
    component: lazyLoad(() => import('@/views/ComingSoon.vue')),
    meta: { titleKey: 'routes.resource' }
  },

  // 用户中心
  {
    path: '/user',
    name: 'user-center',
    component: lazyLoad(() => import('@/views/user/UserCenterPage.vue')),
    meta: { requiresAuth: true, titleKey: 'routes.userCenter' }
  },

  // 通知
  {
    path: '/notifications',
    name: 'notifications',
    component: lazyLoad(() => import('@/views/NotificationsPage.vue')),
    meta: { requiresAuth: true, titleKey: 'routes.notifications' }
  },

  // 管理后台
  {
    path: '/admin',
    component: lazyLoad(() => import('@/views/admin/AdminLayout.vue')),
    meta: { requiresAuth: true, layout: 'admin' },
    children: [
      {
        path: '',
        name: 'admin-dashboard',
        component: lazyLoad(() => import('@/views/admin/DashboardPage.vue')),
        meta: { titleKey: 'routes.admin' }
      },
      {
        path: 'reports',
        name: 'admin-reports',
        component: lazyLoad(() => import('@/views/admin/ReportsPage.vue')),
        meta: { titleKey: 'routes.adminReports' }
      },
      {
        path: 'reviews',
        name: 'admin-reviews',
        component: lazyLoad(() => import('@/views/admin/ReviewsManagePage.vue')),
        meta: { titleKey: 'routes.adminReviews' }
      },
      {
        path: 'logs',
        name: 'admin-logs',
        component: lazyLoad(() => import('@/views/admin/LogsPage.vue')),
        meta: { titleKey: 'routes.adminLogs' }
      }
    ]
  },

  // 旧路由重定向
  { path: '/courses/:id', redirect: to => `/review/courses/${to.params.id}` },
  { path: '/campus', redirect: '/spoc' },
  { path: '/community', redirect: '/' },
  { path: '/review/latest', redirect: '/review' },
  { path: '/review/rankings', redirect: '/review' },
  { path: '/review/courses', redirect: '/review' },
  { path: '/review/teachers/:id', redirect: to => `/teachers/${to.params.id}` },
  { path: '/review/post', redirect: '/review' },

  // 404
  {
    path: '/:pathMatch(.*)*',
    name: 'not-found',
    component: lazyLoad(() => import('@/views/errors/NotFoundPage.vue')),
    meta: { titleKey: 'routes.notFound', layout: 'none' }
  }
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
  scrollBehavior(_to, _from, savedPosition) {
    if (savedPosition) return savedPosition
    return { top: 0 }
  }
})

router.beforeEach((to, _from, next) => {
  // Hash 路由兼容：SSO 回调把 code/state 放在真实 query string 中（# 之前），
  // Vue Router 只读 # 之后的部分，所以需要重定向到 hash 路由格式
  if (window.location.pathname === '/auth/callback') {
    const params = new URLSearchParams(window.location.search)
    const code = params.get('code')
    if (code) {
      const state = params.get('state') || ''
      window.location.replace(
        `${window.location.origin}${window.location.pathname.replace('/auth/callback', '')}/#/auth/callback?code=${encodeURIComponent(code)}&state=${encodeURIComponent(state)}`
      )
      return
    }
  }

  const authStore = useAuthStore()
  const isAuthenticated = authStore.isAuthenticated

  if (to.meta.titleKey) {
    const { t } = i18n.global
    document.title = `${t(to.meta.titleKey)} - StuHelper`
  }

  if (to.meta.requiresAuth && !isAuthenticated) {
    next({ name: 'login', query: { redirect: to.fullPath } })
    return
  }

  // 客户端 token 过期预检（不替代服务端校验）
  if (to.meta.requiresAuth && isAuthenticated && isTokenExpired()) {
    clearAuth()
    authStore.user = null
    next({ name: 'login', query: { redirect: to.fullPath } })
    return
  }

  if (to.meta.guest && isAuthenticated) {
    const redirect = to.query.redirect as string
    // 仅允许站内相对路径重定向，防止 Open Redirect
    if (redirect && redirect.startsWith('/') && !redirect.startsWith('//')) {
      next(redirect)
    } else {
      next({ name: 'teaching-hub' })
    }
    return
  }

  next()
})

export default router
