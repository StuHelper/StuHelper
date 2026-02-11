<template>
  <div class="app-shell">
    <GradientBar />

    <header class="navbar" :class="{ scrolled: isScrolled }">
      <div class="navbar-inner">
        <router-link to="/" class="logo">
          <span class="logo-text gradient-text">StuHelper</span>
        </router-link>

        <ModuleTabs v-if="!isMobile" />

        <div class="navbar-spacer" />

        <button class="search-pill" @click="commandPalette.open()">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="11" cy="11" r="8" />
            <path d="m21 21-4.3-4.3" />
          </svg>
          <span class="search-pill-text">{{ t('nav.searchPlaceholder') }}</span>
          <kbd class="search-kbd">{{ isMac ? '⌘' : 'Ctrl' }}K</kbd>
        </button>

        <NotificationBell v-if="authStore.isAuthenticated" />

        <div v-if="authStore.isAuthenticated" class="avatar-btn" @click="goToUser">
          <img
            v-if="authStore.user?.avatar"
            :src="authStore.user.avatar"
            :alt="authStore.user.displayName"
            class="avatar-img"
          />
          <div v-else class="avatar-placeholder">
            {{ avatarInitial }}
          </div>
        </div>

        <router-link v-else to="/login" class="login-btn">
          {{ t('nav.login') }}
        </router-link>
      </div>
    </header>

    <main class="main-content">
      <slot />
    </main>

    <CommandPalette />
    <Toast />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useCommandPalette } from '@/composables/useCommandPalette'
import GradientBar from './GradientBar.vue'
import ModuleTabs from './ModuleTabs.vue'
import NotificationBell from '@/components/common/NotificationBell.vue'
import CommandPalette from '@/components/common/CommandPalette.vue'
import Toast from '@/components/common/Toast.vue'

const { t } = useI18n()
const router = useRouter()
const authStore = useAuthStore()
const commandPalette = useCommandPalette()

const isScrolled = ref(false)
const isMobile = ref(false)
const isMac = navigator.platform.toUpperCase().includes('MAC')

const avatarInitial = computed(() => {
  const name = authStore.user?.displayName || authStore.user?.name || '?'
  return name.charAt(0).toUpperCase()
})

function goToUser() {
  router.push('/user')
}

function handleScroll() {
  isScrolled.value = window.scrollY > 10
}

function handleResize() {
  isMobile.value = window.innerWidth < 768
}

onMounted(() => {
  window.addEventListener('scroll', handleScroll, { passive: true })
  window.addEventListener('resize', handleResize)
  handleResize()
})

onUnmounted(() => {
  window.removeEventListener('scroll', handleScroll)
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped>
.app-shell {
  min-height: 100vh;
  position: relative;
  z-index: var(--z-base);
}

.navbar {
  position: fixed;
  top: var(--gradient-bar-height);
  left: 0;
  right: 0;
  height: var(--navbar-height);
  z-index: var(--z-sticky);
  transition: background var(--duration-base) var(--ease-smooth),
    border-color var(--duration-base) var(--ease-smooth),
    backdrop-filter var(--duration-base) var(--ease-smooth);
}

.navbar.scrolled {
  background: var(--bg-glass-heavy);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  border-bottom: 1px solid var(--border);
}

.navbar-inner {
  max-width: var(--max-width);
  margin: 0 auto;
  height: 100%;
  display: flex;
  align-items: center;
  gap: var(--space-4);
  padding: 0 var(--space-6);
}

.logo {
  text-decoration: none;
  flex-shrink: 0;
}

.logo-text {
  font-family: var(--font-display);
  font-size: var(--text-xl);
  font-weight: var(--weight-extrabold);
  letter-spacing: var(--tracking-tight);
}

.navbar-spacer {
  flex: 1;
}

.search-pill {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-1-5) var(--space-3);
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: var(--radius-full);
  color: var(--text-muted);
  font-size: var(--text-sm);
  cursor: pointer;
  transition: border-color var(--duration-fast) var(--ease-smooth),
    box-shadow var(--duration-fast) var(--ease-smooth);
  min-width: 200px;
}

.search-pill:hover {
  border-color: var(--brand-primary);
  box-shadow: var(--shadow-glow-sm);
}

.search-pill-text {
  flex: 1;
  text-align: left;
}

.search-kbd {
  font-family: var(--font-sans);
  font-size: var(--text-xs);
  padding: 2px 6px;
  background: var(--bg-tertiary);
  border-radius: var(--radius-xs);
  color: var(--text-muted);
  border: none;
}

.avatar-btn {
  width: 32px;
  height: 32px;
  border-radius: var(--radius-full);
  cursor: pointer;
  overflow: hidden;
  flex-shrink: 0;
  background: var(--gradient-brand);
  padding: 2px;
}

.avatar-img {
  width: 100%;
  height: 100%;
  border-radius: var(--radius-full);
  object-fit: cover;
}

.avatar-placeholder {
  width: 100%;
  height: 100%;
  border-radius: var(--radius-full);
  background: var(--bg-card);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--text-sm);
  font-weight: var(--weight-semibold);
  color: var(--text-primary);
}

.login-btn {
  padding: var(--space-1-5) var(--space-4);
  background: var(--gradient-brand);
  color: white;
  border-radius: var(--radius-full);
  font-size: var(--text-sm);
  font-weight: var(--weight-medium);
  text-decoration: none;
  transition: opacity var(--duration-fast) var(--ease-smooth);
  white-space: nowrap;
}

.login-btn:hover {
  opacity: 0.9;
  color: white;
}

.main-content {
  padding-top: calc(var(--gradient-bar-height) + var(--navbar-height));
  min-height: 100vh;
}

@media (max-width: 767px) {
  .navbar-inner {
    padding: 0 var(--space-4);
    gap: var(--space-3);
  }

  .search-pill {
    min-width: auto;
    padding: var(--space-1-5) var(--space-2);
  }

  .search-pill-text,
  .search-kbd {
    display: none;
  }
}
</style>
