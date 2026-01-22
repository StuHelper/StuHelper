import { createRouter, createWebHashHistory } from 'vue-router'
import { userManager } from '@/utils/auth'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    // 认证路由
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginPage.vue'),
      meta: { guest: true }
    },
    {
      path: '/auth/callback',
      name: 'auth-callback',
      component: () => import('@/views/AuthCallbackPage.vue'),
      meta: { guest: true }
    },
    {
      path: '/',
      name: 'home',
      component: () => import('@/views/HomePage.vue')
    },
    // 评课社区路由
    {
      path: '/review',
      name: 'review',
      component: () => import('@/views/review/IndexPage.vue')
    },
    {
      path: '/review/courses',
      name: 'courses',
      component: () => import('@/views/review/CourseListPage.vue')
    },
    {
      path: '/review/courses/:id',
      name: 'course-detail',
      component: () => import('@/views/review/CourseDetailPage.vue')
    },
    {
      path: '/review/latest',
      name: 'latest-reviews',
      component: () => import('@/views/review/LatestReviewsPage.vue')
    },
    {
      path: '/review/post',
      name: 'post-review',
      component: () => import('@/views/review/PostReviewPage.vue')
    },
    // 资料共享路由（开发中）
    {
      path: '/resource',
      name: 'resource',
      component: () => import('@/views/ComingSoon.vue')
    },
    // SPOC路由（开发中）
    {
      path: '/spoc',
      name: 'spoc',
      component: () => import('@/views/ComingSoon.vue')
    }
  ]
})

// 路由守卫
router.beforeEach((to, from, next) => {
  const isAuthenticated = userManager.isAuthenticated()

  // 需要登录的页面
  if (to.meta.requiresAuth && !isAuthenticated) {
    next({ name: 'login', query: { redirect: to.fullPath } })
    return
  }

  // 已登录用户访问登录页面，重定向到首页
  if (to.meta.guest && isAuthenticated) {
    next({ name: 'home' })
    return
  }

  next()
})

export default router
