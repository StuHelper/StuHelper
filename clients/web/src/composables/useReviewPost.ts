/**
 * 共享 composable：跨组件控制"写测评"对话框的显示状态
 * AppShell 顶栏写测评按钮触发打开，ReviewPage 显示 ReviewDialog
 * lastPostedAt 供子页面（如 CourseDetailPage）监听发布事件并刷新数据
 * preselectedCourse 允许从课程详情页直接打开已选好课程的对话框
 */
import { storeToRefs } from 'pinia'
import { defineStore } from 'pinia'
import { useRouter } from 'vue-router'
import type { Course } from '@stuhelper/shared/course'
import { api } from '@/api'
import { getErrorMessage } from '@/api/errors'
import i18n from '@/i18n'
import { useAuthStore } from '@/stores/auth'
import { useToast } from '@/composables/useToast'

const useReviewPostStore = defineStore('reviewPost', {
  state: () => ({
    showPostModal: false,
    lastPostedAt: 0,
    preselectedCourse: null as Course | null,
  }),
  actions: {
    open(course?: Course) {
      this.preselectedCourse = course ?? null
      this.showPostModal = true
    },
    close() {
      this.showPostModal = false
      this.preselectedCourse = null
    },
    markPosted() {
      this.lastPostedAt = Date.now()
    },
  },
})

export type UseReviewPostReturn = ReturnType<typeof useReviewPost>

export function useReviewPost() {
  const router = useRouter()
  const authStore = useAuthStore()
  const toast = useToast()
  const reviewPostStore = useReviewPostStore()
  const { showPostModal, lastPostedAt, preselectedCourse } = storeToRefs(reviewPostStore)

  function openPostModal(course?: Course) {
    reviewPostStore.open(course)
  }

  function closePostModal() {
    reviewPostStore.close()
  }

  function notifyPosted() {
    reviewPostStore.markPosted()
  }

  async function ensureCanPostReview() {
    if (!authStore.isAuthenticated) {
      await router.push({
        name: 'login',
        query: { redirect: router.currentRoute.value.fullPath },
      })
      return false
    }

    try {
      await api.identity.getUserSurface()
      return true
    } catch (error) {
      toast.error(getErrorMessage(error, i18n.global.t('common.loadFailed')))
      return false
    }
  }

  return { showPostModal, lastPostedAt, preselectedCourse, openPostModal, closePostModal, notifyPosted, ensureCanPostReview }
}
