<template>
  <div class="min-h-screen relative z-0">
    <header
      class="fixed top-0 left-0 right-0 z-[var(--z-sticky)] transition-all duration-200 ease-smooth bg-bg-glass backdrop-blur-lg backdrop-saturate-150"
      :class="isScrolled && 'bg-bg-glass-heavy backdrop-blur-xl border-b border-white/15 dark:border-white/8 shadow-sm'"
    >
      <div class="max-w-[var(--max-width)] mx-auto h-[var(--navbar-height)] flex items-center gap-4 px-6 max-md:px-4 max-md:gap-3">
        <router-link to="/" class="no-underline shrink-0">
          <span class="font-display text-xl font-extrabold tracking-tight gradient-text">StuHelper</span>
        </router-link>

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

        <LocaleSwitcher />
        <NotificationBell v-if="authStore.isAuthenticated" />

        <div
          v-if="authStore.isAuthenticated"
          ref="userMenuRef"
          class="relative"
        >
          <button
            ref="userMenuButtonRef"
            type="button"
            class="w-8 h-8 rounded-full cursor-pointer overflow-hidden shrink-0 bg-gradient-to-br from-primary to-accent p-0.5"
            aria-haspopup="menu"
            :aria-controls="USER_MENU_ID"
            :aria-expanded="userMenuOpen"
            :aria-label="t('nav.user')"
            @click="toggleUserMenu"
            @keydown="handleMenuButtonKeydown"
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
            :id="USER_MENU_ID"
            role="menu"
            :aria-label="t('nav.user')"
            class="absolute right-0 top-full mt-1.5 w-48 bg-bg-card rounded-lg shadow-md py-1 z-[var(--z-dropdown)] animate-fade-in"
            @keydown="handleUserMenuKeydown"
          >
            <button
              type="button"
              role="menuitem"
              tabindex="-1"
              data-user-menu-item
              class="flex items-center gap-2 w-full px-3 py-2 text-sm text-text-secondary transition-colors duration-fast hover:bg-bg-hover hover:text-text-primary"
              @click="goToUser"
            >
              <User class="size-4" />
              {{ t('nav.profile') }}
            </button>
            <button
              type="button"
              role="menuitem"
              tabindex="-1"
              data-user-menu-item
              class="flex items-center gap-2 w-full px-3 py-2 text-sm transition-colors duration-fast hover:bg-bg-hover"
              :class="verificationStore.identityVerified ? 'text-success' : 'text-text-secondary hover:text-text-primary'"
              @click="goTo('identity-verification')"
            >
              <ShieldCheck class="size-4" />
              {{ t('nav.identityVerification') }}
              <span v-if="verificationStore.identityVerified" class="ml-auto text-[10px] bg-success/10 text-success px-1.5 py-0.5 rounded-full">{{ t('user.verification.identity.verified') }}</span>
            </button>
            <button
              type="button"
              role="menuitem"
              tabindex="-1"
              data-user-menu-item
              class="flex items-center gap-2 w-full px-3 py-2 text-sm transition-colors duration-fast hover:bg-bg-hover"
              :class="verificationStore.studentVerified ? 'text-success' : 'text-text-secondary hover:text-text-primary'"
              @click="goTo('student-verification')"
            >
              <GraduationCap class="size-4" />
              {{ t('nav.studentVerification') }}
              <span v-if="verificationStore.studentVerified" class="ml-auto text-[10px] bg-success/10 text-success px-1.5 py-0.5 rounded-full">{{ t('user.verification.student.verified') }}</span>
            </button>
            <button
              type="button"
              role="menuitem"
              tabindex="-1"
              data-user-menu-item
              class="flex items-center gap-2 w-full px-3 py-2 text-sm transition-colors duration-fast hover:bg-bg-hover"
              :class="verificationStore.qqBound ? 'text-success' : 'text-text-secondary hover:text-text-primary'"
              @click="goTo('qq-binding')"
            >
              <Bot class="size-4" />
              {{ t('nav.qqBinding') }}
              <span v-if="verificationStore.qqBound" class="ml-auto text-[10px] bg-success/10 text-success px-1.5 py-0.5 rounded-full">{{ t('user.verification.qq.bound') }}</span>
            </button>
            <template v-if="showAdminEntry">
              <div class="h-px bg-border mx-2 my-0.5" />
              <button
                type="button"
                role="menuitem"
                tabindex="-1"
                data-user-menu-item
                class="flex items-center gap-2 w-full px-3 py-2 text-sm text-accent font-medium transition-colors duration-fast hover:bg-accent/10"
                @click="goToAdmin"
              >
                <Settings class="size-4" />
                {{ t('nav.adminConsole') }}
              </button>
            </template>
            <div class="h-px bg-border mx-2 my-0.5" />
            <button
              type="button"
              role="menuitem"
              tabindex="-1"
              data-user-menu-item
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

      <div class="hidden max-tablet:block px-4 pb-2">
        <InlineSearch />
      </div>
    </header>

    <main class="pt-[var(--navbar-height)] max-tablet:pt-[var(--mobile-header-height)] min-h-screen">
      <slot />
    </main>

    <CommandPalette />
    <Toast />
    <FloatingModuleNav />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Bot, GraduationCap, LogOut, PenLine, Settings, ShieldCheck, User } from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import { canShowAdminEntry } from '@/utils/adminAccess'
import { useVerificationStore } from '@/stores/verification'
import { useReviewPost } from '@/composables/useReviewPost'
import { useToast } from '@/composables/useToast'
import FloatingModuleNav from './FloatingModuleNav.vue'
import InlineSearch from '@/components/common/InlineSearch.vue'
import NotificationBell from '@/components/common/NotificationBell.vue'
import CommandPalette from '@/components/common/CommandPalette.vue'
import Toast from '@/components/common/Toast.vue'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'

const USER_MENU_ID = 'app-shell-user-menu'

const { t } = useI18n()
const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const verificationStore = useVerificationStore()
const toast = useToast()
const { ensureCanPostReview, openPostModal } = useReviewPost()

const isReviewRoute = computed(() =>
  route.path.startsWith('/review') ||
  route.path.startsWith('/courses') ||
  route.path.startsWith('/teachers'),
)

const isScrolled = ref(false)
const userMenuOpen = ref(false)
const userMenuRef = ref<HTMLElement | null>(null)
const userMenuButtonRef = ref<HTMLButtonElement | null>(null)
const userMenuItems = ref<HTMLElement[]>([])
let scrollTicking = false

const avatarInitial = computed(() => {
  const name = authStore.user?.displayName || authStore.user?.name || '?'
  return name.charAt(0).toUpperCase()
})

const showAdminEntry = computed(() => canShowAdminEntry(authStore.user))

function syncUserMenuItems() {
  userMenuItems.value = userMenuRef.value
    ? Array.from(userMenuRef.value.querySelectorAll<HTMLElement>('[data-user-menu-item]'))
    : []
}

function focusUserMenuItem(index: number) {
  const items = userMenuItems.value
  if (items.length === 0) {
    return
  }

  const normalizedIndex = (index + items.length) % items.length
  items[normalizedIndex]?.focus()
}

async function openUserMenu(focusIndex?: number) {
  if (!userMenuOpen.value) {
    userMenuOpen.value = true
    await nextTick()
  }

  syncUserMenuItems()

  if (focusIndex !== undefined) {
    focusUserMenuItem(focusIndex)
  }
}

function closeUserMenu(restoreFocus = false) {
  userMenuOpen.value = false
  userMenuItems.value = []

  if (restoreFocus) {
    void nextTick(() => {
      userMenuButtonRef.value?.focus()
    })
  }
}

async function toggleUserMenu() {
  if (userMenuOpen.value) {
    closeUserMenu()
    return
  }

  await openUserMenu()
}

async function handleMenuButtonKeydown(event: KeyboardEvent) {
  switch (event.key) {
    case 'ArrowDown': {
      event.preventDefault()
      await openUserMenu(0)
      break
    }
    case 'ArrowUp': {
      event.preventDefault()
      await openUserMenu(userMenuItems.value.length - 1)
      break
    }
    case 'Enter':
    case ' ': {
      event.preventDefault()
      await toggleUserMenu()
      break
    }
    case 'Escape': {
      event.preventDefault()
      closeUserMenu(true)
      break
    }
  }
}

function handleUserMenuKeydown(event: KeyboardEvent) {
  const items = userMenuItems.value
  if (items.length === 0) {
    return
  }

  const currentIndex = items.findIndex((item) => item === document.activeElement)

  switch (event.key) {
    case 'ArrowDown': {
      event.preventDefault()
      focusUserMenuItem(currentIndex < 0 ? 0 : currentIndex + 1)
      break
    }
    case 'ArrowUp': {
      event.preventDefault()
      focusUserMenuItem(currentIndex < 0 ? items.length - 1 : currentIndex - 1)
      break
    }
    case 'Home': {
      event.preventDefault()
      focusUserMenuItem(0)
      break
    }
    case 'End': {
      event.preventDefault()
      focusUserMenuItem(items.length - 1)
      break
    }
    case 'Escape': {
      event.preventDefault()
      closeUserMenu(true)
      break
    }
    case 'Tab': {
      closeUserMenu()
      break
    }
  }
}

function goToUser() {
  closeUserMenu()
  router.push('/user/reviews')
}

function goTo(routeName: string) {
  closeUserMenu()
  router.push({ name: routeName })
}

function goToAdmin() {
  closeUserMenu()
  window.location.assign('/admin/')
}

async function handleWriteReview() {
  if (!(await ensureCanPostReview())) {
    return
  }

  const courseID = typeof route.params.id === 'string' ? Number(route.params.id) : NaN
  if (route.path.startsWith('/courses/') && Number.isFinite(courseID) && courseID > 0) {
    await router.push({ name: 'course-review-post', params: { id: courseID } })
    return
  }
  openPostModal()
}

async function handleLogout() {
  closeUserMenu()
  const result = await authStore.logout()
  if (result.ok) {
    router.push('/')
  } else {
    toast.error(result.reason === 'server'
      ? t('nav.logoutServerError')
      : t('nav.logoutNetworkError'))
  }
}

function onClickOutside(event: MouseEvent) {
  if (userMenuRef.value && !userMenuRef.value.contains(event.target as Node)) {
    closeUserMenu()
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
  isScrolled.value = window.scrollY > 10
  window.addEventListener('scroll', handleScroll, { passive: true })
  document.addEventListener('click', onClickOutside, true)
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

<style scoped>
.has-new .bell-btn svg {
  animation: bellShake 0.5s ease;
}

@keyframes bellShake {
  0%, 100% { transform: rotate(0); }
  25% { transform: rotate(15deg); }
  75% { transform: rotate(-15deg); }
}

.dropdown-enter-active {
  transition: all var(--duration-slow) var(--ease-spring);
}
.dropdown-leave-active {
  transition: all var(--duration-base) var(--ease-out);
}

.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: translateY(-8px) scale(0.95);
}
</style>
