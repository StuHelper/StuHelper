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
          v-ripple
          class="flex items-center gap-1.5 bg-gradient-to-r from-accent to-primary text-white rounded-full px-4 py-1.5 text-sm font-semibold whitespace-nowrap cursor-pointer shadow-sm hover:shadow-md hover:-translate-y-px transition-all duration-fast shrink-0 press-spring"
          @click="handleWriteReview"
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
          class="relative"
          ref="userMenuRef"
        >
          <button
            class="w-8 h-8 rounded-full cursor-pointer overflow-hidden shrink-0 bg-gradient-to-br from-primary to-accent p-0.5"
            @click="userMenuOpen = !userMenuOpen"
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
          </button>
          <span
            v-if="showAdminEntry"
            class="absolute -bottom-0.5 -right-0.5 w-3.5 h-3.5 bg-accent rounded-full border-2 border-bg-card flex items-center justify-center"
            :title="t('nav.adminBadge')"
          >
            <Settings class="size-2 text-white" />
          </span>
          <div
            v-if="userMenuOpen"
            class="absolute right-0 top-full mt-1.5 w-48 bg-bg-card rounded-lg shadow-md py-1 z-[var(--z-dropdown)] animate-fade-in"
          >
            <button
              class="flex items-center gap-2 w-full px-3 py-2 text-sm text-text-secondary transition-colors duration-fast hover:bg-bg-hover hover:text-text-primary"
              @click="goToUser"
            >
              <User class="size-4" />
              {{ t('nav.profile') }}
            </button>
            <button
              class="flex items-center gap-2 w-full px-3 py-2 text-sm transition-colors duration-fast hover:bg-bg-hover"
              :class="verificationStore.identityVerified ? 'text-success' : 'text-text-secondary hover:text-text-primary'"
              @click="goTo('identity-verification')"
            >
              <ShieldCheck class="size-4" />
              {{ t('nav.identityVerification') }}
              <span v-if="verificationStore.identityVerified" class="ml-auto text-[10px] bg-success/10 text-success px-1.5 py-0.5 rounded-full">{{ t('user.verification.identity.verified') }}</span>
            </button>
            <button
              class="flex items-center gap-2 w-full px-3 py-2 text-sm transition-colors duration-fast hover:bg-bg-hover"
              :class="verificationStore.studentVerified ? 'text-success' : 'text-text-secondary hover:text-text-primary'"
              @click="goTo('student-verification')"
            >
              <GraduationCap class="size-4" />
              {{ t('nav.studentVerification') }}
              <span v-if="verificationStore.studentVerified" class="ml-auto text-[10px] bg-success/10 text-success px-1.5 py-0.5 rounded-full">{{ t('user.verification.student.verified') }}</span>
            </button>
            <template v-if="showAdminEntry">
              <div class="h-px bg-border mx-2 my-0.5" />
              <a
                href="/admin/"
                class="flex items-center gap-2 w-full px-3 py-2 text-sm text-accent font-medium transition-colors duration-fast hover:bg-accent/10 no-underline"
              >
                <Settings class="size-4" />
                {{ t('nav.adminConsole') }}
              </a>
            </template>
            <div class="h-px bg-border mx-2 my-0.5" />
            <button
              class="flex items-center gap-2 w-full px-3 py-2 text-sm text-danger transition-colors duration-fast hover:bg-danger/10"
              @click="handleLogout"
            >
              <LogOut class="size-4" />
              {{ t('nav.logout') }}
            </button>
          </div>
        </div>

        <router-link
          v-else
          to="/login"
          class="py-1.5 px-4 bg-gradient-to-br from-primary to-accent text-white rounded-full text-sm font-medium no-underline whitespace-nowrap hover:opacity-90 transition-all duration-fast press-spring hover:shadow-glow-primary"
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
import { PenLine, LogOut, User, ShieldCheck, GraduationCap, Settings } from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import { useVerificationStore } from '@/stores/verification'
import { useReviewPost } from '@/composables/useReviewPost'
import { useToast } from '@/composables/useToast'
import { canAccessAdmin } from '@stuhelper/shared/constants'
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
const verificationStore = useVerificationStore()
const toast = useToast()
const { openPostModal } = useReviewPost()

const isReviewRoute = computed(() =>
  route.path.startsWith('/review') ||
  route.path.startsWith('/courses') ||
  route.path.startsWith('/teachers')
)

const isScrolled = ref(false)
const userMenuOpen = ref(false)
const userMenuRef = ref<HTMLElement | null>(null)
// 滚动节流，避免高频触发
let scrollTicking = false

const avatarInitial = computed(() => {
  const name = authStore.user?.displayName || authStore.user?.name || '?'
  return name.charAt(0).toUpperCase()
})

const showAdminEntry = computed(() =>
  canAccessAdmin(authStore.globalCapabilities)
)

function goToUser() {
  userMenuOpen.value = false
  router.push('/user/reviews')
}

function goTo(routeName: string) {
  userMenuOpen.value = false
  router.push({ name: routeName })
}

function handleWriteReview() {
  const courseID = typeof route.params.id === 'string' ? Number(route.params.id) : NaN
  if (route.path.startsWith('/courses/') && Number.isFinite(courseID) && courseID > 0) {
    router.push({ name: 'course-review-post', params: { id: courseID } })
    return
  }
  openPostModal()
}

async function handleLogout() {
  userMenuOpen.value = false
  const result = await authStore.logout()
  if (result.ok) {
    router.push('/')
  } else {
    toast.error(result.reason === 'server'
      ? t('nav.logoutServerError')
      : t('nav.logoutNetworkError'))
  }
}

function onClickOutside(e: MouseEvent) {
  if (userMenuRef.value && !userMenuRef.value.contains(e.target as Node)) {
    userMenuOpen.value = false
  }
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
  // 初始化时同步滚动状态，防止页面刷新后状态不一致
  isScrolled.value = window.scrollY > 10
  // passive 已设置
  window.addEventListener('scroll', handleScroll, { passive: true })
  document.addEventListener('click', onClickOutside, true)
  // Bootstrap verification status for authenticated users
  if (authStore.isAuthenticated) {
    void verificationStore.fetchStatus().catch((error) => {
      if (import.meta.env.DEV) {
        console.warn('[AppShell] failed to bootstrap verification status', error)
      }
    })
  }
})

onUnmounted(() => {
  window.removeEventListener('scroll', handleScroll)
  document.removeEventListener('click', onClickOutside, true)
})
</script>
