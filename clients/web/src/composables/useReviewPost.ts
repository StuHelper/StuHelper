/**
 * 共享 composable：跨组件控制"写测评"对话框的显示状态
 * AppShell 顶栏写测评按钮触发打开，ReviewPage 显示 ReviewDialog
 * lastPostedAt 供子页面（如 CourseDetailPage）监听发布事件并刷新数据
 * preselectedCourse 允许从课程详情页直接打开已选好课程的对话框
 */
import { ref } from 'vue'
import type { Course } from '@/types/course'

const showPostModal = ref(false)
const lastPostedAt = ref(0)
const preselectedCourse = ref<Course | null>(null)

export function useReviewPost() {
  function openPostModal(course?: Course) {
    preselectedCourse.value = course ?? null
    showPostModal.value = true
  }

  function closePostModal() {
    showPostModal.value = false
  }

  function notifyPosted() {
    lastPostedAt.value = Date.now()
  }

  return { showPostModal, lastPostedAt, preselectedCourse, openPostModal, closePostModal, notifyPosted }
}
