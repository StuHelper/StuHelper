<template>
  <div class="relative">
    <!-- 窄屏侧栏：固定宽度抽屉（放在 grid 外面，不影响布局） -->
    <div class="hidden max-tablet:block">
      <!-- 竖排标签按钮：展开/收起共用位置和样式 -->
      <button
        class="fixed left-0 top-1/2 -translate-y-1/2 z-40 w-6 py-2.5 flex flex-col items-center justify-center gap-px bg-gradient-to-b from-pink-400 to-pink-500 dark:from-blue-500 dark:to-indigo-500 rounded-r-lg shadow-md cursor-pointer transition-all duration-200 ease-out hover:shadow-lg hover:brightness-110 border-none"
        :class="sidebarOpen && 'translate-x-[260px]'"
        @click="sidebarOpen = !sidebarOpen"
      >
        <span v-for="ch in t('review.courseList')" :key="ch" class="text-[10px] font-semibold text-white leading-[1.3]">{{ ch }}</span>
      </button>

      <!-- 抽屉面板 -->
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
          class="fixed left-0 top-[var(--navbar-height)] bottom-0 z-30 w-[260px] bg-bg-base border-r border-border shadow-lg overflow-y-auto p-3"
        >
          <DepartmentSidebar />
        </div>
      </transition>
    </div>

    <div
      class="pl-4 pr-6 py-6 grid grid-cols-[clamp(200px,20vw,260px)_1fr] gap-4 min-h-[calc(100vh-var(--navbar-height))] max-tablet:grid-cols-1 max-tablet:pl-0 max-tablet:pr-4"
    >
      <!-- 院系侧栏：宽屏正常展示 -->
      <div
        class="sticky top-[calc(var(--navbar-height)+1.5rem)] self-start max-h-[calc(100vh-var(--navbar-height)-3rem)] overflow-y-auto max-tablet:hidden"
      >
        <DepartmentSidebar />
      </div>

      <!-- 右侧内容 -->
      <main class="min-w-0 max-tablet:px-4">
        <router-view v-if="hasChildRoute" />
        <ReviewFeed v-else :key="feedKey" />
      </main>
    </div>
    <ReviewDialog :visible="showPostModal" @close="closePostModal" @posted="handlePosted" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import DepartmentSidebar from '@/components/review/DepartmentSidebar.vue'
import ReviewFeed from '@/components/review/ReviewFeed.vue'
import ReviewDialog from '@/components/review/ReviewDialog.vue'
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

// 登录后自动恢复草稿：检测 draft_pending 标记并打开弹窗
onMounted(() => {
  if (sessionStorage.getItem('draft_pending')) {
    sessionStorage.removeItem('draft_pending')
    openPostModal()
  }
})

// 选中课程（路由变化）后自动收起抽屉
watch(() => route.params.id, () => {
  sidebarOpen.value = false
})
</script>
