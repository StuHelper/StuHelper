<template>
  <div ref="userMenuRef" class="relative">
    <button
      ref="userMenuButtonRef"
      type="button"
      class="inline-flex size-11 shrink-0 cursor-pointer items-center justify-center rounded-xl border border-transparent bg-transparent transition-all duration-fast hover:border-white/20 hover:bg-bg-card focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
      aria-haspopup="menu"
      :aria-controls="USER_MENU_ID"
      :aria-expanded="userMenuOpen"
      :aria-label="t('nav.user')"
      @click="toggleUserMenu"
      @keydown="handleMenuButtonKeydown"
    >
      <span class="grid size-8 place-items-center overflow-hidden rounded-full bg-gradient-to-br from-primary to-accent p-0.5 shadow-xs">
        <img
          v-if="authStore.user?.avatar"
          :src="authStore.user.avatar"
          :alt="authStore.user?.displayName || t('nav.userAvatar')"
          loading="lazy"
          decoding="async"
          class="size-full rounded-full object-cover"
        />
        <span v-else class="flex size-full items-center justify-center rounded-full bg-bg-base text-sm font-semibold text-text-primary">
          {{ avatarInitial }}
        </span>
      </span>
    </button>

    <span
      v-if="showAdminEntry"
      class="absolute bottom-1 right-1 flex size-4 items-center justify-center rounded-full border-2 border-bg-base bg-accent"
      :title="t('nav.adminBadge')"
    >
      <Settings class="size-2 text-white" />
    </span>

    <div
      v-if="userMenuOpen"
      :id="USER_MENU_ID"
      role="menu"
      :aria-label="t('nav.user')"
      class="absolute right-0 top-full z-[var(--z-dropdown)] mt-2 w-60 rounded-xl border border-white/20 bg-bg-glass-heavy/95 p-1.5 shadow-lg backdrop-blur-xl animate-fade-in dark:border-white/8"
      @keydown="handleUserMenuKeydown"
    >
      <button
        type="button"
        role="menuitem"
        tabindex="-1"
        data-user-menu-item
        class="flex min-h-10 w-full items-center gap-2.5 rounded-lg px-3 text-sm text-text-secondary transition-colors duration-fast hover:bg-bg-hover hover:text-text-primary"
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
        class="flex min-h-10 w-full items-center gap-2.5 rounded-lg px-3 text-sm transition-colors duration-fast hover:bg-bg-hover"
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
        class="flex min-h-10 w-full items-center gap-2.5 rounded-lg px-3 text-sm transition-colors duration-fast hover:bg-bg-hover"
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
        class="flex min-h-10 w-full items-center gap-2.5 rounded-lg px-3 text-sm transition-colors duration-fast hover:bg-bg-hover"
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
          class="flex min-h-10 w-full items-center gap-2.5 rounded-lg px-3 text-sm font-medium text-accent transition-colors duration-fast hover:bg-accent/10"
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
        class="flex min-h-10 w-full items-center gap-2.5 rounded-lg px-3 text-sm text-danger transition-colors duration-fast hover:bg-danger/10"
        @click="handleLogout"
      >
        <LogOut class="size-4" />
        {{ t('nav.logout') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Bot, GraduationCap, LogOut, Settings, ShieldCheck, User } from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import { useVerificationStore } from '@/stores/verification'
import { useToast } from '@/composables/useToast'
import { canShowAdminEntry } from '@/utils/adminAccess'
import { resolveAdminConsoleURL } from '@/utils/adminUrl'

const USER_MENU_ID = 'app-shell-user-menu'
const adminConsoleURL = resolveAdminConsoleURL(import.meta.env.VITE_ADMIN_URL)

const { t } = useI18n()
const router = useRouter()
const authStore = useAuthStore()
const verificationStore = useVerificationStore()
const toast = useToast()

const userMenuOpen = ref(false)
const userMenuRef = ref<HTMLElement | null>(null)
const userMenuButtonRef = ref<HTMLButtonElement | null>(null)
const userMenuItems = ref<HTMLElement[]>([])

const avatarInitial = computed(() => {
  const name = authStore.user?.displayName || authStore.user?.name || '?'
  return name.charAt(0).toUpperCase()
})

const showAdminEntry = computed(() => canShowAdminEntry(authStore.user))
const canFetchVerificationStatus = computed(() =>
  authStore.bootstrapCompleted && authStore.isAuthenticated,
)

function syncUserMenuItems() {
  userMenuItems.value = userMenuRef.value
    ? Array.from(userMenuRef.value.querySelectorAll<HTMLElement>('[data-user-menu-item]'))
    : []
}

function focusUserMenuItem(index: number) {
  const items = userMenuItems.value
  if (items.length === 0) return

  const normalizedIndex = (index + items.length) % items.length
  items[normalizedIndex]?.focus()
}

async function openUserMenu(focusIndex?: number) {
  if (!userMenuOpen.value) {
    userMenuOpen.value = true
    await nextTick()
  }

  syncUserMenuItems()
  if (focusIndex !== undefined) focusUserMenuItem(focusIndex)
}

function closeUserMenu(restoreFocus = false) {
  userMenuOpen.value = false
  userMenuItems.value = []

  if (restoreFocus) {
    void nextTick(() => userMenuButtonRef.value?.focus())
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
  if (event.key === 'ArrowDown') {
    event.preventDefault()
    await openUserMenu(0)
  } else if (event.key === 'ArrowUp') {
    event.preventDefault()
    await openUserMenu(userMenuItems.value.length - 1)
  } else if (event.key === 'Enter' || event.key === ' ') {
    event.preventDefault()
    await toggleUserMenu()
  } else if (event.key === 'Escape') {
    event.preventDefault()
    closeUserMenu(true)
  }
}

function handleUserMenuKeydown(event: KeyboardEvent) {
  const currentIndex = userMenuItems.value.findIndex((item) => item === document.activeElement)
  if (event.key === 'ArrowDown') {
    event.preventDefault()
    focusUserMenuItem(currentIndex < 0 ? 0 : currentIndex + 1)
  } else if (event.key === 'ArrowUp') {
    event.preventDefault()
    focusUserMenuItem(currentIndex < 0 ? userMenuItems.value.length - 1 : currentIndex - 1)
  } else if (event.key === 'Home') {
    event.preventDefault()
    focusUserMenuItem(0)
  } else if (event.key === 'End') {
    event.preventDefault()
    focusUserMenuItem(userMenuItems.value.length - 1)
  } else if (event.key === 'Escape') {
    event.preventDefault()
    closeUserMenu(true)
  } else if (event.key === 'Tab') {
    closeUserMenu()
  }
}

function goToUser() {
  closeUserMenu()
  void router.push('/user/reviews')
}

function goTo(routeName: string) {
  closeUserMenu()
  void router.push({ name: routeName })
}

function goToAdmin() {
  closeUserMenu()
  window.location.assign(adminConsoleURL)
}

async function handleLogout() {
  closeUserMenu()
  const result = await authStore.logout()
  if (result.ok) {
    void router.push('/')
    return
  }

  toast.error(result.reason === 'server'
    ? t('nav.logoutServerError')
    : t('nav.logoutNetworkError'))
}

function onClickOutside(event: MouseEvent) {
  if (userMenuRef.value?.contains(event.target as Node)) return
  closeUserMenu()
}

onMounted(() => {
  document.addEventListener('click', onClickOutside, true)
  if (!canFetchVerificationStatus.value) {
    return
  }
  void verificationStore.fetchStatus().catch((error) => {
    if (import.meta.env.DEV) {
      console.warn('[AppUserMenu] failed to bootstrap verification status', error)
    }
  })
})

onUnmounted(() => {
  document.removeEventListener('click', onClickOutside, true)
})
</script>
