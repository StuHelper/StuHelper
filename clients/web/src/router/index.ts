/**
 * 路由配置
 * 统一认证检查、404 处理、懒加载错误处理
 */
import { createRouter, createWebHashHistory, type RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import i18n from '@/i18n'

// 路由元信息类型
declare module 'vue-router' {
  interface RouteMeta {
    requiresAuth?: boolean
    guest?: boolean
    titleKey?: string // i18n key for route title
  }
}

// 懒加载错误处理
function lazyLoad(loader: () => Promise<unknown>) {
  return () =>
    loader().catch(() => {
      return import('@/views/errors/LoadErrorPage.vue')
    })
}

// 路由配置
const routes: RouteRecordRaw[] = [
  // 认证路由
  {
    path: '/login',
    name: 'login',
    component: lazyLoad(() => import('@/views/LoginPage.vue')),
    meta: { guest: true, titleKey: 'routes.login' }
  },
  {
    path: '/auth/callback',
    name: 'auth-callback',
    component: lazyLoad(() => import('@/views/AuthCallbackPage.vue')),
    meta: { guest: true, titleKey: 'routes.authCallback' }
  },

  // 首页
  {
    path: '/',
    name: 'home',
    component: lazyLoad(() => import('@/views/HomePage.vue')),
    meta: { titleKey: 'routes.home' }
  },

  // 评课社区
  {
    path: '/review',
    name: 'review',
    component: lazyLoad(() => import('@/views/review/IndexPage.vue')),
    meta: { titleKey: 'routes.review' }
  },
  {
    path: '/review/courses',
    name: 'courses',
    component: lazyLoad(() => import('@/views/review/CourseListPage.vue')),
    meta: { titleKey: 'routes.courses' }
  },
  {
    path: '/review/courses/:id',
    name: 'course-detail',
    component: lazyLoad(() => import('@/views/review/CourseDetailPage.vue')),
    meta: { titleKey: 'routes.courseDetail' }
  },
  {
    path: '/review/latest',
    name: 'latest-reviews',
    component: lazyLoad(() => import('@/views/review/LatestReviewsPage.vue')),
    meta: { titleKey: 'routes.latestReviews' }
  },
  {
    path: '/review/post',
    name: 'post-review',
    component: lazyLoad(() => import('@/views/review/PostReviewPage.vue')),
    meta: { requiresAuth: true, titleKey: 'routes.postReview' }
  },

  // 开发中功能
  {
    path: '/resource',
    name: 'resource',
    component: lazyLoad(() => import('@/views/ComingSoon.vue')),
    meta: { titleKey: 'routes.resource' }
  },
  {
    path: '/spoc',
    name: 'spoc',
    component: lazyLoad(() => import('@/views/ComingSoon.vue')),
    meta: { titleKey: 'routes.spoc' }
  },

  // 404 页面
  {
    path: '/:pathMatch(.*)*',
    name: 'not-found',
    component: lazyLoad(() => import('@/views/errors/NotFoundPage.vue')),
    meta: { titleKey: 'routes.notFound' }
  }
]

// 创建路由
const router = createRouter({
  history: createWebHashHistory(),
  routes,
  scrollBehavior(to, from, savedPosition) {
    if (savedPosition) return savedPosition
    return { top: 0 }
  }
})

// 路由守卫
router.beforeEach((to, from, next) => {
  const authStore = useAuthStore()
  const isAuthenticated = authStore.isAuthenticated

  // 设置页面标题（使用 i18n 翻译）
  if (to.meta.titleKey) {
    const { t } = i18n.global
    document.title = `${t(to.meta.titleKey)} - StuHelper`
  }

  // 需要登录的页面
  if (to.meta.requiresAuth && !isAuthenticated) {
    next({
      name: 'login',
      query: { redirect: to.fullPath }
    })
    return
  }

  // 已登录用户访问登录页
  if (to.meta.guest && isAuthenticated) {
    const redirect = to.query.redirect as string
    next(redirect || { name: 'home' })
    return
  }

  next()
})

export default router
