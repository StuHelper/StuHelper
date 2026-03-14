import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import {
  Odometer,
  Flag,
  ChatDotRound,
  User,
  Key,
  School,
  Setting,
  Document,
  UserFilled,
  Warning,
} from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'
import BasicLayout from '@/layouts/BasicLayout.vue'

// Route definitions for the admin panel
const adminChildren: RouteRecordRaw[] = [
  // --- Existing pages (migrated) ---
  {
    path: '',
    name: 'admin-dashboard',
    component: () => import('@/views/dashboard/index.vue'),
    meta: { title: '仪表盘', icon: Odometer },
  },
  {
    path: 'reviews',
    name: 'admin-reviews',
    component: () => import('@/views/review-manage/index.vue'),
    meta: { title: '评课管理', icon: ChatDotRound },
  },
  {
    path: 'reports',
    name: 'admin-reports',
    component: () => import('@/views/report-manage/index.vue'),
    meta: { title: '举报管理', icon: Flag },
  },
  {
    path: 'teachers',
    name: 'admin-teachers',
    component: () => import('@/views/teacher-manage/index.vue'),
    meta: { title: '教师管理', icon: UserFilled },
  },
  {
    path: 'sensitive-words',
    name: 'admin-sensitive-words',
    component: () => import('@/views/sensitive-words/index.vue'),
    meta: { title: '敏感词管理', icon: Warning },
  },
  {
    path: 'logs',
    name: 'admin-logs',
    component: () => import('@/views/logs/index.vue'),
    meta: { title: '操作日志', icon: Document },
  },
  // --- New user system pages ---
  {
    path: 'identities',
    name: 'admin-identities',
    component: () => import('@/views/user-system/IdentityReview.vue'),
    meta: { title: '实名认证审核', icon: User },
  },
  {
    path: 'student-verifications',
    name: 'admin-student-verifications',
    component: () => import('@/views/user-system/StudentVerificationReview.vue'),
    meta: { title: '学生认证审核', icon: User },
  },
  {
    path: 'roles',
    name: 'admin-roles',
    component: () => import('@/views/rbac/RoleManage.vue'),
    meta: { title: '角色管理', icon: Key },
  },
  {
    path: 'school-configs',
    name: 'admin-school-configs',
    component: () => import('@/views/school-config/index.vue'),
    meta: { title: '学校配置', icon: School },
  },
  {
    path: 'system-configs',
    name: 'admin-system-configs',
    component: () => import('@/views/system-config/index.vue'),
    meta: { title: '系统配置', icon: Setting },
  },
]

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    component: BasicLayout,
    redirect: '/dashboard',
    children: adminChildren.map((child) => ({
      ...child,
      // Normalize paths: remove leading '/' if any
      path: child.path || '',
    })),
  },
  {
    path: '/login',
    name: 'login',
    component: () => import('@/views/login/index.vue'),
    meta: { title: '登录' },
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'not-found',
    component: () => import('@/views/error/404.vue'),
    meta: { title: '404' },
  },
]

// Build menu structure with groups
interface MenuGroup {
  path: string
  meta?: { title?: string; icon?: unknown }
  children?: RouteRecordRaw[]
}

export const menuRoutes: MenuGroup[] = [
  {
    path: '/dashboard',
    meta: { title: '仪表盘', icon: Odometer },
    children: [adminChildren[0]].map(addFullPath),
  },
  {
    path: '/content',
    meta: { title: '内容管理', icon: ChatDotRound },
    children: adminChildren.slice(1, 6).map(addFullPath),
  },
  {
    path: '/user-system',
    meta: { title: '用户管理', icon: User },
    children: adminChildren.slice(6, 8).map(addFullPath),
  },
  {
    path: '/permission',
    meta: { title: '权限管理', icon: Key },
    children: [adminChildren[8]].map(addFullPath),
  },
  {
    path: '/school-config',
    meta: { title: '学校配置', icon: School },
    children: [adminChildren[9]].map(addFullPath),
  },
  {
    path: '/system',
    meta: { title: '系统设置', icon: Setting },
    children: [adminChildren[10]].map(addFullPath),
  },
]

function addFullPath(route: RouteRecordRaw): RouteRecordRaw {
  return {
    ...route,
    path: route.path ? `/${route.path}` : '/',
  }
}

const router = createRouter({
  history: createWebHistory('/admin'),
  routes,
  scrollBehavior: () => ({ top: 0 }),
})

// Auth guard
router.beforeEach(async (to) => {
  const authStore = useAuthStore()

  // Bootstrap session on first load
  if (!authStore.initialized) {
    await authStore.bootstrap()
  }

  // Login page: skip auth check
  if (to.name === 'login') {
    if (authStore.isAuthenticated && authStore.isAdmin) {
      return { path: '/' }
    }
    return true
  }

  // Check authentication
  if (!authStore.isAuthenticated) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }

  // Check admin permission
  if (!authStore.isAdmin) {
    return { name: 'login' }
  }

  // Update page title
  const title = to.meta?.title
  if (typeof title === 'string') {
    document.title = `${title} - StuHelper Admin`
  }

  return true
})

export default router
