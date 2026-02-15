<template>
  <div class="min-h-screen relative z-0">
    <header
      class="fixed top-0 left-0 right-0 z-[var(--z-sticky)] transition-all duration-200 ease-smooth bg-bg-glass backdrop-blur-lg backdrop-saturate-150"
      :class="isScrolled && 'bg-bg-glass-heavy backdrop-blur-xl border-b border-white/15 dark:border-white/8 shadow-sm'"
    >
      <!-- 第一行：Logo + 搜索(宽屏) + 写测评 + 通知 + 头像 -->
      <div class="max-w-[var(--max-width)] mx-auto h-[var(--navbar-height)] flex items-center gap-4 px-6 max-md:px-4 max-md:gap-3">
        <router-link to="/" class="no-underline shrink-0">
          <span class="font-display text-xl font-extrabold tracking-tight gradient-text">StuHelper</span>
        </router-link>

        <!-- 宽屏搜索框（contents 让 div 对 flex 布局透明） -->
        <div class="contents max-tablet:hidden">
          <InlineSearch />
        </div>

        <button
          v-if="isReviewRoute"
          class="flex items-center gap-1.5 bg-gradient-to-r from-accent to-primary text-white rounded-full px-4 py-1.5 text-sm font-semibold whitespace-nowrap cursor-pointer shadow-sm hover:shadow-md hover:-translate-y-px transition-all duration-fast shrink-0"
          @click="openPostModal()"
        >
          <PenLine :size="14" />
          <span>{{ t('review.topBar.writeReview') }}</span>
        </button>

        <div class="flex-1" />

        <!-- 右侧：语言切换 + 通知铃铛 + 头像/登录 -->
        <LocaleSwitcher />
        <NotificationBell v-if="authStore.isAuthenticated" />

        <div
          v-if="authStore.isAuthenticated"
          class="w-8 h-8 rounded-full cursor-pointer overflow-hidden shrink-0 bg-gradient-to-br from-primary to-accent p-0.5"
          @click="goToUser"
        >
          <img
            v-if="authStore.user?.avatar"
            :src="authStore.user.avatar"
            :alt="authStore.user?.displayName || t('nav.userAvatar')"
            loading="lazy"
            decoding="async"
            class="w-full h-full rounded-full object-cover"
          />
          <div v-else class="w-full h-full rounded-full bg-bg-card flex items-center justify-center text-sm font-semibold text-text-primary">
            {{ avatarInitial }}
          </div>
        </div>

        <router-link
          v-else
          to="/login"
          class="py-1.5 px-4 bg-gradient-to-br from-primary to-accent text-white rounded-full text-sm font-medium no-underline whitespace-nowrap hover:opacity-90 transition-opacity duration-fast"
        >
          {{ t('nav.login') }}
        </router-link>
      </div>

      <!-- 第二行：窄屏搜索框 -->
      <div class="hidden max-tablet:block px-4 pb-2">
        <InlineSearch />
      </div>
    </header>

    <main class="pt-[var(--navbar-height)] max-tablet:pt-[calc(var(--navbar-height)+44px)] min-h-screen">
      <slot />
    </main>

    <CommandPalette />
    <Toast />
    <FloatingModuleNav />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { PenLine } from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import { useReviewPost } from '@/composables/useReviewPost'
import FloatingModuleNav from './FloatingModuleNav.vue'
import InlineSearch from '@/components/common/InlineSearch.vue'
import NotificationBell from '@/components/common/NotificationBell.vue'
import CommandPalette from '@/components/common/CommandPalette.vue'
import Toast from '@/components/common/Toast.vue'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'

const { t } = useI18n()
const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const { openPostModal } = useReviewPost()

const isReviewRoute = computed(() => route.path.startsWith('/review'))

const isScrolled = ref(false)
// M-43: 滚动节流，避免高频触发
let scrollTicking = false

const avatarInitial = computed(() => {
  const name = authStore.user?.displayName || authStore.user?.name || '?'
  return name.charAt(0).toUpperCase()
})

function goToUser() {
  router.push('/user')
}

function handleScroll() {
  if (scrollTicking) return
  scrollTicking = true
  requestAnimationFrame(() => {
    isScrolled.value = window.scrollY > 10
    scrollTicking = false
  })
}

onMounted(() => {
  // L-44: 初始化时同步滚动状态，防止页面刷新后状态不一致
  isScrolled.value = window.scrollY > 10
  // L-16: passive 已设置
  window.addEventListener('scroll', handleScroll, { passive: true })
})

onUnmounted(() => {
  window.removeEventListener('scroll', handleScroll)
})
</script>
