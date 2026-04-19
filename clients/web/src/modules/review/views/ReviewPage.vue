<template>
  <CourseThemeProvider>
    <div class="relative">
      <!-- Narrow-screen sidebar: fixed-width drawer (outside grid, doesn't affect layout) -->
      <div class="hidden max-tablet:block">
        <!-- Vertical tab button: shared position and style for expand/collapse -->
        <button
          class="fixed left-0 top-1/2 -translate-y-1/2 z-40 w-6 py-2.5 flex flex-col items-center justify-center gap-px rounded-r-lg shadow-lg cursor-pointer transition-all duration-200 ease-out hover:shadow-xl hover:brightness-110 border-none bg-primary text-white"
          :class="sidebarOpen && 'translate-x-[260px]'"
          @click="sidebarOpen = !sidebarOpen"
        >
          <span
            v-for="ch in t('review.courseListLabel')"
            :key="ch"
            class="text-[10px] font-semibold text-text-muted leading-[1.3]"
          >{{ ch }}</span>
        </button>

        <!-- Drawer panel -->
        <transition
          enter-active-class="transition-transform duration-200 ease-out"
          enter-from-class="-translate-x-full"
          enter-to-class="translate-x-0"
          leave-active-class="transition-transform duration-200 ease-out"
          leave-from-class="translate-x-0"
          leave-to-class="-translate-x-full"
        >
          <div
            v-if="sidebarOpen"
            class="fixed left-0 top-[var(--navbar-height)] bottom-0 z-30 w-[260px] bg-bg-card border-r border-border-light shadow-lg overflow-y-auto p-3"
          >
            <DepartmentSidebar />
          </div>
        </transition>
      </div>

      <div
        class="bg-bg-base pl-4 pr-6 py-6 grid grid-cols-[clamp(200px,20vw,260px)_1fr] gap-4 min-h-[calc(100vh-var(--navbar-height))] max-tablet:grid-cols-1 max-tablet:pl-0 max-tablet:pr-4"
      >
        <!-- Department sidebar: wide screen normal display -->
        <div
          class="sticky top-[calc(var(--navbar-height)+1.5rem)] self-start max-h-[calc(100vh-var(--navbar-height)-3rem)] overflow-y-auto max-tablet:hidden"
        >
          <DepartmentSidebar />
        </div>

        <!-- Right content area -->
        <main class="min-w-0 max-tablet:px-4">
          <router-view v-if="hasChildRoute" />
          <ReviewFeed v-else :key="feedKey" />
        </main>
      </div>
      <ReviewDialog :visible="showPostModal" @close="closePostModal" @posted="handlePosted" />
    </div>
  </CourseThemeProvider>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import CourseThemeProvider from '@/modules/course/theme/CourseThemeProvider.vue'
import DepartmentSidebar from '@/components/business/review/DepartmentSidebar.vue'
import ReviewFeed from '@/components/business/review/ReviewFeed.vue'
import ReviewDialog from '@/components/business/review/ReviewDialog.vue'
import { useReviewPost } from '@/composables/useReviewPost'

const { t } = useI18n()
const route = useRoute()
const sidebarOpen = ref(false)
const { showPostModal, closePostModal, notifyPosted, openPostModal } = useReviewPost()
const feedKey = ref(0)

const hasChildRoute = computed(() => {
  return route.matched.length > 1 && route.name !== 'review'
})

function handlePosted() {
  closePostModal()
  notifyPosted()
  feedKey.value++
}

// 登录回流后检测 draft_pending 标记，自动恢复发帖弹窗
onMounted(() => {
  if (sessionStorage.getItem('draft_pending')) {
    sessionStorage.removeItem('draft_pending')
    openPostModal()
  }
})

// 路由切到具体课程后自动关闭侧边抽屉
watch(() => route.params.id, () => {
  sidebarOpen.value = false
})
</script>
