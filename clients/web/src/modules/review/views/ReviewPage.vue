<template>
  <CourseThemeProvider>
    <div class="relative">
      <!-- Narrow-screen sidebar: fixed-width drawer (outside grid, doesn't affect layout) -->
      <div class="hidden max-tablet:block">
        <!-- Vertical tab button: shared position and style for expand/collapse -->
        <button
          type="button"
          class="fixed left-0 top-1/2 -translate-y-1/2 z-40 size-11 flex items-center justify-center rounded-r-xl shadow-lg cursor-pointer transition-all duration-200 ease-out hover:shadow-xl hover:brightness-110 border-none bg-primary text-white"
          :class="sidebarOpen && 'translate-x-[260px]'"
          :aria-label="t('review.courseListLabel')"
          :aria-expanded="sidebarOpen"
          @click="sidebarOpen = !sidebarOpen"
        >
          <PanelLeftClose v-if="sidebarOpen" :size="20" aria-hidden="true" />
          <PanelLeftOpen v-else :size="20" aria-hidden="true" />
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
            class="fixed left-0 top-[var(--mobile-header-height)] bottom-0 z-30 w-[260px] bg-bg-card border-r border-border-light shadow-lg overflow-y-auto p-3"
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
          <ReviewFeed v-else />
        </main>
      </div>
    </div>
  </CourseThemeProvider>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { PanelLeftClose, PanelLeftOpen } from 'lucide-vue-next'
import CourseThemeProvider from '@/modules/course/theme/CourseThemeProvider.vue'
import DepartmentSidebar from '@/components/business/review/DepartmentSidebar.vue'
import ReviewFeed from '@/components/business/review/ReviewFeed.vue'

const { t } = useI18n()
const route = useRoute()
const sidebarOpen = ref(false)

const hasChildRoute = computed(() => {
  return route.matched.length > 1 && route.name !== 'review'
})

// 路由切到具体课程后自动关闭侧边抽屉
watch(() => route.params.id, () => {
  sidebarOpen.value = false
})
</script>
