<template>
  <div class="flex min-h-screen">
    <!-- Mobile overlay -->
    <div
      v-if="mobileOpen"
      class="fixed inset-0 bg-bg-overlay z-[var(--z-modal-backdrop)] md:hidden"
      @click="mobileOpen = false"
    />

    <!-- Sidebar -->
    <aside
      class="fixed top-0 left-0 h-full bg-bg-card border-r border-border flex flex-col z-[var(--z-modal)] transition-all duration-base ease-smooth"
      :class="[
        collapsed ? 'w-16' : 'w-[220px]',
        mobileOpen ? 'translate-x-0' : 'max-md:-translate-x-full'
      ]"
    >
      <!-- Logo area -->
      <div class="flex items-center gap-2.5 h-14 px-4 border-b border-border shrink-0 overflow-hidden">
        <div class="size-8 rounded-lg bg-gradient-to-br from-primary to-accent flex items-center justify-center shrink-0">
          <span class="text-text-inverse text-sm font-bold">S</span>
        </div>
        <span v-if="!collapsed" class="font-sans text-sm font-bold text-text-primary whitespace-nowrap">{{ t('admin.title') }}</span>
      </div>

      <!-- Nav items -->
      <nav class="flex-1 p-2 space-y-0.5 overflow-y-auto">
        <router-link
          v-for="item in visibleNavItems"
          :key="item.name"
          :to="{ name: item.name }"
          class="group relative flex items-center gap-3 px-3 py-2.5 text-text-muted no-underline text-sm rounded-lg transition-all duration-fast hover:text-text-primary hover:bg-bg-hover [&.router-link-active]:text-primary [&.router-link-active]:bg-primary/[0.08] [&.router-link-active]:font-medium"
          :title="collapsed ? item.label : undefined"
          @click="mobileOpen = false"
        >
          <div
            class="absolute left-0 top-1/2 -translate-y-1/2 w-[3px] h-0 rounded-r-full bg-primary transition-all duration-fast group-[&.router-link-active]:h-5"
          />
          <component :is="item.icon" class="size-[18px] shrink-0" />
          <span v-if="!collapsed" class="whitespace-nowrap">{{ item.label }}</span>
        </router-link>
      </nav>

      <!-- Bottom area -->
      <div class="p-2 border-t border-border shrink-0">
        <!-- Collapse toggle (desktop only) -->
        <button
          class="hidden md:flex items-center gap-3 w-full px-3 py-2 text-text-muted text-sm rounded-lg transition-colors duration-fast hover:text-text-primary hover:bg-bg-hover"
          :title="collapsed ? 'Expand' : 'Collapse'"
          @click="collapsed = !collapsed"
        >
          <PanelLeftClose v-if="!collapsed" class="size-[18px] shrink-0" />
          <PanelLeftOpen v-else class="size-[18px] shrink-0" />
          <span v-if="!collapsed" class="whitespace-nowrap">{{ collapsed ? '' : 'Collapse' }}</span>
        </button>
        <!-- Back to site -->
        <router-link
          :to="{ name: 'review' }"
          class="flex items-center gap-3 px-3 py-2 text-text-muted no-underline text-sm rounded-lg transition-colors duration-fast hover:text-text-primary hover:bg-bg-hover"
          :title="collapsed ? t('admin.nav.backToSite') : undefined"
        >
          <ExternalLink class="size-[18px] shrink-0" />
          <span v-if="!collapsed" class="whitespace-nowrap">{{ t('admin.nav.backToSite') }}</span>
        </router-link>
      </div>
    </aside>

    <!-- Main content -->
    <div
      class="flex-1 flex flex-col min-h-screen transition-all duration-base ease-smooth"
      :class="collapsed ? 'md:ml-16' : 'md:ml-[220px]'"
    >
      <!-- Top bar -->
      <header class="sticky top-0 h-14 bg-bg-card border-b border-border flex items-center justify-between px-4 md:px-6 z-[var(--z-sticky)] shrink-0">
        <div class="flex items-center gap-3">
          <!-- Mobile hamburger -->
          <button
            class="md:hidden p-1.5 text-text-muted rounded-lg hover:bg-bg-hover hover:text-text-primary transition-colors duration-fast"
            aria-label="Toggle menu"
            @click="mobileOpen = !mobileOpen"
          >
            <Menu class="size-5" />
          </button>
          <!-- Breadcrumb -->
          <nav class="flex items-center gap-1.5 text-sm" aria-label="Breadcrumb">
            <span class="text-text-muted">{{ t('admin.title') }}</span>
            <ChevronRight v-if="currentPageLabel" class="size-3.5 text-text-muted" />
            <span v-if="currentPageLabel" class="text-text-primary font-medium">{{ currentPageLabel }}</span>
          </nav>
        </div>
        <div class="flex items-center gap-3">
          <router-link
            :to="{ name: 'review' }"
            class="hidden sm:flex items-center gap-1.5 text-sm text-text-muted no-underline hover:text-text-primary transition-colors duration-fast"
          >
            <ExternalLink class="size-3.5" />
            <span>{{ t('admin.nav.backToSite') }}</span>
          </router-link>
          <div v-if="authStore.user" class="relative" ref="userMenuRef">
            <button
              class="flex items-center gap-2 px-2 py-1 rounded-lg transition-colors duration-fast hover:bg-bg-hover"
              @click="userMenuOpen = !userMenuOpen"
            >
              <div class="size-7 rounded-full bg-gradient-to-br from-primary to-accent flex items-center justify-center">
                <span class="text-text-inverse text-xs font-medium">{{ authStore.user.name?.charAt(0)?.toUpperCase() || 'A' }}</span>
              </div>
              <span class="hidden sm:inline text-sm text-text-secondary">{{ authStore.user.name }}</span>
              <ChevronDown class="size-3.5 text-text-muted transition-transform duration-fast" :class="userMenuOpen && 'rotate-180'" />
            </button>
            <div
              v-if="userMenuOpen"
              class="absolute right-0 top-full mt-1 w-40 bg-bg-card border border-border rounded-lg shadow-md py-1 z-[var(--z-dropdown)] animate-fade-in"
            >
              <button
                class="flex items-center gap-2 w-full px-3 py-2 text-sm text-danger transition-colors duration-fast hover:bg-danger/10"
                @click="handleLogout"
              >
                <LogOut class="size-4" />
                {{ t('nav.logout') }}
              </button>
            </div>
          </div>
        </div>
      </header>

      <!-- Page content -->
      <main class="flex-1 p-4 md:p-6 overflow-y-auto">
        <ErrorBoundary>
          <router-view />
        </ErrorBoundary>
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import {
  LayoutDashboard, Flag, MessageSquare, FileText, Users, ShieldAlert,
  PanelLeftClose, PanelLeftOpen, ExternalLink, Menu, ChevronRight,
  ChevronDown, LogOut
} from 'lucide-vue-next'
import ErrorBoundary from '@/components/common/ErrorBoundary.vue'
import { useAuthStore } from '@/stores/auth'
import { useToast } from '@/composables/useToast'
import {
  ADMIN_DASHBOARD_VIEW,
  ADMIN_LOGS_VIEW,
  ADMIN_REPORTS_MANAGE,
  ADMIN_REVIEWS_MANAGE,
  ADMIN_SENSITIVE_WORDS_MANAGE,
  ADMIN_TEACHERS_MANAGE,
  hasAnyCapability,
} from '@stuhelper/shared/constants'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const toast = useToast()

const collapsed = ref(false)
const mobileOpen = ref(false)
const userMenuOpen = ref(false)
const userMenuRef = ref<HTMLElement | null>(null)

async function handleLogout() {
  userMenuOpen.value = false
  const result = await authStore.logout()
  if (result.ok) {
    if (result.ssoLogoutURL) {
      window.location.href = result.ssoLogoutURL
    } else {
      router.push('/')
    }
  } else {
    toast.error(result.reason === 'server'
      ? '服务器异常，请稍后重试'
      : '登出失败，请检查网络后重试')
  }
}

function onClickOutside(e: MouseEvent) {
  if (userMenuRef.value && !userMenuRef.value.contains(e.target as Node)) {
    userMenuOpen.value = false
  }
}

onMounted(() => document.addEventListener('click', onClickOutside, true))
onUnmounted(() => document.removeEventListener('click', onClickOutside, true))

const navItems = computed(() => [
  { name: 'admin-dashboard', label: t('admin.nav.dashboard'), icon: LayoutDashboard, requiredCapabilities: [ADMIN_DASHBOARD_VIEW] },
  { name: 'admin-reports', label: t('admin.nav.reports'), icon: Flag, requiredCapabilities: [ADMIN_REPORTS_MANAGE] },
  { name: 'admin-reviews', label: t('admin.nav.reviews'), icon: MessageSquare, requiredCapabilities: [ADMIN_REVIEWS_MANAGE] },
  { name: 'admin-teachers', label: t('admin.nav.teachers'), icon: Users, requiredCapabilities: [ADMIN_TEACHERS_MANAGE] },
  { name: 'admin-sensitive-words', label: t('admin.nav.sensitiveWords'), icon: ShieldAlert, requiredCapabilities: [ADMIN_SENSITIVE_WORDS_MANAGE] },
  { name: 'admin-logs', label: t('admin.nav.logs'), icon: FileText, requiredCapabilities: [ADMIN_LOGS_VIEW] },
])

const visibleNavItems = computed(() =>
  navItems.value.filter((item) =>
    hasAnyCapability(authStore.globalCapabilities, item.requiredCapabilities),
  ),
)

const currentPageLabel = computed(() => {
  const current = navItems.value.find(item => item.name === route.name)
  return current?.label || ''
})
</script>
