<template>
  <header
    class="fixed top-0 left-0 right-0 z-[var(--z-sticky)] border-b transition-all duration-200 ease-smooth bg-bg-glass/90 backdrop-blur-xl backdrop-saturate-150"
    :class="isScrolled ? 'border-white/15 dark:border-white/8 shadow-sm' : 'border-transparent'"
  >
    <div class="mx-auto flex h-[var(--navbar-height)] max-w-[var(--max-width)] items-center gap-5 px-6 max-md:px-4 max-md:gap-2">
      <router-link
        :to="logoRoute"
        class="group flex min-w-0 shrink-0 items-center gap-2.5 rounded-xl py-1 pr-2 no-underline transition-opacity duration-fast hover:opacity-90"
        :aria-label="t('nav.logo')"
      >
        <span class="grid size-9 place-items-center rounded-lg bg-gradient-to-br from-primary to-accent text-sm font-extrabold text-white shadow-glow-sm">
          S
        </span>
        <span class="flex min-w-0 flex-col leading-none">
          <span class="font-display text-lg font-extrabold tracking-tight text-text-primary">{{ brandTitle }}</span>
          <span class="mt-1 hidden text-[11px] font-medium text-text-muted xl:block">{{ brandTagline }}</span>
        </span>
      </router-link>

      <nav
        class="hidden min-w-0 items-center gap-1 rounded-xl border border-white/25 bg-bg-card/70 p-1 shadow-xs lg:flex dark:border-white/8"
        :aria-label="t('nav.primary')"
      >
        <router-link
          v-for="item in navItems"
          :key="item.to"
          :to="item.to"
          class="relative inline-flex h-11 items-center gap-2 rounded-lg px-3 text-sm font-semibold text-text-secondary no-underline whitespace-nowrap transition-all duration-fast hover:bg-bg-hover hover:text-text-primary"
          :class="isNavActive(item) && 'bg-bg-base text-text-primary shadow-sm'"
        >
          <component
            :is="item.icon"
            :size="16"
            class="shrink-0"
            :class="isNavActive(item) ? 'text-primary' : 'text-text-muted'"
          />
          {{ item.label }}
          <span
            v-if="isNavActive(item)"
            class="absolute inset-x-3 -bottom-1 h-0.5 rounded-full bg-gradient-to-r from-primary to-accent"
            aria-hidden="true"
          />
        </router-link>
      </nav>

      <div v-if="showCourseSearch" class="w-[min(40vw,520px)] min-w-[240px] shrink max-tablet:hidden">
        <InlineSearch />
      </div>

      <button
        v-if="showWriteReview"
        v-ripple
        class="flex h-11 shrink-0 cursor-pointer items-center justify-center gap-2 rounded-xl bg-gradient-to-r from-accent to-primary px-4 text-sm font-semibold text-white whitespace-nowrap shadow-sm transition-all duration-fast hover:-translate-y-px hover:shadow-md press-spring max-sm:w-11 max-sm:px-0"
        :aria-label="t('review.topBar.writeReview')"
        :title="t('review.topBar.writeReview')"
        @click="handleWriteReview"
      >
        <PenLine :size="16" aria-hidden="true" />
        <span class="max-sm:hidden">{{ t('review.topBar.writeReview') }}</span>
      </button>

      <div class="flex-1" />

      <div class="flex items-center gap-2">
        <button
          type="button"
          class="hidden size-11 items-center justify-center rounded-xl border border-transparent text-text-secondary transition-all duration-fast hover:border-white/20 hover:bg-bg-card hover:text-text-primary max-lg:inline-flex"
          :aria-label="t('nav.menu')"
          :title="t('nav.menu')"
          aria-controls="app-mobile-nav"
          :aria-expanded="mobileMenuOpen"
          @click="mobileMenuOpen = !mobileMenuOpen"
        >
          <X v-if="mobileMenuOpen" :size="20" aria-hidden="true" />
          <Menu v-else :size="20" aria-hidden="true" />
        </button>
        <LocaleSwitcher />
        <NotificationBell v-if="showNotificationBell" />
        <AppUserMenu v-if="showAuthenticatedActions" />

        <router-link
          v-else-if="showLoginEntry"
          :to="loginEntryRoute"
          class="inline-flex h-11 items-center justify-center gap-2 rounded-xl bg-text-primary px-4 text-sm font-semibold text-bg-base no-underline whitespace-nowrap shadow-sm transition-all duration-fast hover:-translate-y-px hover:bg-primary hover:text-white hover:shadow-glow-primary max-sm:w-11 max-sm:px-0"
          :aria-label="t('nav.login')"
          :title="t('nav.login')"
        >
          <LogIn class="size-4" aria-hidden="true" />
          <span class="max-sm:hidden">{{ t('nav.login') }}</span>
        </router-link>
      </div>
    </div>

    <div
      v-if="mobileMenuOpen"
      id="app-mobile-nav"
      class="absolute left-3 right-3 lg:hidden"
      :style="{ top: showCourseSearch ? 'calc(var(--navbar-height) + 56px)' : 'calc(var(--navbar-height) + 8px)' }"
    >
      <nav
        class="grid gap-1 rounded-xl border border-white/20 bg-bg-glass-heavy/95 p-2 shadow-lg backdrop-blur-xl dark:border-white/8"
        :aria-label="t('nav.primary')"
      >
        <router-link
          v-for="item in navItems"
          :key="item.to"
          :to="item.to"
          class="flex min-h-11 items-center gap-3 rounded-lg px-3 text-sm font-semibold text-text-secondary no-underline transition-colors duration-fast hover:bg-bg-hover hover:text-text-primary"
          :class="isNavActive(item) && 'bg-bg-card text-text-primary shadow-xs'"
        >
          <span
            class="grid size-8 place-items-center rounded-md"
            :class="isNavActive(item) ? 'bg-primary-alpha text-primary' : 'bg-bg-secondary text-text-muted'"
          >
            <component :is="item.icon" :size="17" aria-hidden="true" />
          </span>
          <span>{{ item.label }}</span>
        </router-link>
      </nav>
    </div>

    <div v-if="showCourseSearch" class="hidden max-tablet:block px-4 pb-2">
      <InlineSearch />
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  GraduationCap,
  Home,
  KeyRound,
  LibraryBig,
  LogIn,
  LockKeyhole,
  Menu,
  Network,
  PenLine,
  ShieldCheck,
  UserRound,
  X,
  type LucideIcon,
} from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import { useReviewPost } from '@/composables/useReviewPost'
import InlineSearch from '@/components/common/InlineSearch.vue'
import NotificationBell from '@/components/common/NotificationBell.vue'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import AppUserMenu from './AppUserMenu.vue'
import { rememberReviewPostCourse } from '@/modules/review/reviewPostNavigation'
import {
  absoluteURLOnPreferredOrigin,
  configuredIdentityOrigin,
  configuredWebOrigin,
} from '@/utils/redirect'

interface NavItem {
  to: string
  label: string
  icon: LucideIcon
  exact?: boolean
}

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const { ensureCanPostReview } = useReviewPost()

const isScrolled = ref(false)
const mobileMenuOpen = ref(false)
let scrollTicking = false

const isIdentityPortalHost = computed(() => {
  if (typeof window === 'undefined') return false
  return configuredIdentityOrigin() === window.location.origin
})
const logoRoute = computed(() => (isIdentityPortalHost.value ? '/identity' : '/'))
const brandTitle = computed(() => (isIdentityPortalHost.value ? t('nav.identityBrand') : 'StuHelper'))
const brandTagline = computed(() => (isIdentityPortalHost.value ? t('nav.identityTagline') : t('nav.tagline')))
const showCourseSearch = computed(() => !isIdentityPortalHost.value && route.path === '/courses')
const showWriteReview = computed(() =>
  !isIdentityPortalHost.value &&
  (route.path.startsWith('/courses') ||
    route.path.startsWith('/teachers')),
)
const showAuthenticatedActions = computed(() =>
  authStore.bootstrapCompleted && authStore.isAuthenticated,
)
const showNotificationBell = computed(() =>
  showAuthenticatedActions.value && !isIdentityPortalHost.value,
)
const showLoginEntry = computed(() => !authStore.isAuthenticated && route.name !== 'login')
const loginEntryRoute = computed(() => ({
  path: '/login',
  query: {
    redirect: typeof window === 'undefined'
      ? route.fullPath
      : absoluteURLOnPreferredOrigin(
        route.fullPath,
        isIdentityPortalHost.value ? window.location.origin : configuredWebOrigin(),
      ),
  },
}))

const navItems = computed<NavItem[]>(() =>
  isIdentityPortalHost.value
    ? [
        { to: '/identity', label: t('routes.identityHome'), icon: UserRound },
        { to: '/account/profile', label: t('routes.accountProfile'), icon: UserRound },
        { to: '/account/security', label: t('routes.accountSecurity'), icon: LockKeyhole },
        { to: '/connect', label: t('routes.identityConnect'), icon: Network },
        { to: '/user/authorized-apps', label: t('routes.userAuthorizedApps'), icon: ShieldCheck },
        { to: '/developers/apps', label: t('routes.openPlatformDeveloperApps'), icon: KeyRound },
      ]
    : [
        { to: '/', label: t('nav.home'), icon: Home, exact: true },
        { to: '/courses', label: t('nav.courses'), icon: LibraryBig },
        { to: '/teachers', label: t('nav.teacher'), icon: GraduationCap },
      ],
)

function isNavActive(item: NavItem) {
  if (item.exact) {
    return route.path === item.to
  }
  return route.path === item.to || route.path.startsWith(`${item.to}/`)
}

async function handleWriteReview() {
  if (!(await ensureCanPostReview())) {
    return
  }

  const courseID = typeof route.params.id === 'string' ? Number(route.params.id) : NaN
  rememberReviewPostCourse(courseID)
  await router.push({ name: 'course-review-post' })
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
  isScrolled.value = window.scrollY > 10
  window.addEventListener('scroll', handleScroll, { passive: true })
})

onUnmounted(() => {
  window.removeEventListener('scroll', handleScroll)
})

watch(() => route.fullPath, () => {
  mobileMenuOpen.value = false
})
</script>
