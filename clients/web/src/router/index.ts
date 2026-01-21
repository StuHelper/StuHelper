import { createRouter, createWebHashHistory } from 'vue-router'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
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

export default router
