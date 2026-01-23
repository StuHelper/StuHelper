<template>
  <nav class="navbar" :class="{ scrolled: isScrolled, 'menu-open': menuOpen }">
    <div class="navbar-inner">
      <!-- Logo -->
      <router-link to="/review" class="logo">
        <span class="logo-icon">📚</span>
        <span class="logo-text">评课社区</span>
      </router-link>

      <!-- Desktop Nav -->
      <div class="nav-links desktop-only">
        <router-link to="/review" class="nav-link" exact-active-class="active">
          首页
        </router-link>
        <router-link to="/review/courses" class="nav-link" active-class="active">
          课程
        </router-link>
        <router-link to="/review/latest" class="nav-link" active-class="active">
          最新
        </router-link>
      </div>

      <!-- Actions -->
      <div class="nav-actions">
        <router-link to="/review/post" class="btn-post desktop-only">
          <span class="btn-icon">✍️</span>
          <span>发测评</span>
        </router-link>

        <!-- User Menu -->
        <div v-if="isAuthenticated" class="user-menu">
          <button class="user-btn" @click="toggleUserMenu">
            <span class="avatar">{{ userInitial }}</span>
          </button>
          <Transition name="dropdown">
            <div v-if="userMenuOpen" class="dropdown-menu">
              <div class="dropdown-header">{{ userName }}</div>
              <button class="dropdown-item" @click="handleLogout">退出登录</button>
            </div>
          </Transition>
        </div>
        <router-link v-else to="/login" class="btn-login desktop-only">
          登录
        </router-link>

        <!-- Mobile Menu Toggle -->
        <button class="menu-toggle mobile-only" @click="toggleMenu">
          <span class="hamburger" :class="{ open: menuOpen }">
            <span></span>
            <span></span>
            <span></span>
          </span>
        </button>
      </div>
    </div>

    <!-- Mobile Menu -->
    <Transition name="slide-down">
      <div v-if="menuOpen" class="mobile-menu mobile-only">
        <router-link to="/review" class="mobile-link" @click="closeMenu">首页</router-link>
        <router-link to="/review/courses" class="mobile-link" @click="closeMenu">课程</router-link>
        <router-link to="/review/latest" class="mobile-link" @click="closeMenu">最新</router-link>
        <router-link to="/review/post" class="mobile-link highlight" @click="closeMenu">发测评</router-link>
        <router-link v-if="!isAuthenticated" to="/login" class="mobile-link" @click="closeMenu">登录</router-link>
      </div>
    </Transition>
  </nav>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const authStore = useAuthStore()

const isScrolled = ref(false)
const menuOpen = ref(false)
const userMenuOpen = ref(false)

const isAuthenticated = computed(() => authStore.isAuthenticated)
const userName = computed(() => authStore.user?.name || '用户')
const userInitial = computed(() => userName.value.charAt(0).toUpperCase())

const toggleMenu = () => {
  menuOpen.value = !menuOpen.value
}

const closeMenu = () => {
  menuOpen.value = false
}

const toggleUserMenu = () => {
  userMenuOpen.value = !userMenuOpen.value
}

const handleLogout = async () => {
  userMenuOpen.value = false
  await authStore.logout()
  router.push('/review')
}

const handleScroll = () => {
  isScrolled.value = window.scrollY > 20
}

onMounted(() => {
  window.addEventListener('scroll', handleScroll)
})

onUnmounted(() => {
  window.removeEventListener('scroll', handleScroll)
})
</script>

<style scoped>
.navbar {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: var(--z-sticky);
  background: transparent;
  transition: all var(--duration-base) var(--ease-out);
}

.navbar.scrolled {
  background: rgba(13, 18, 16, 0.95);
  backdrop-filter: blur(12px);
  border-bottom: 1px solid var(--border);
}

.navbar-inner {
  max-width: var(--max-width);
  margin: 0 auto;
  padding: 0 var(--space-6);
  height: var(--navbar-height);
  display: flex;
  align-items: center;
  justify-content: space-between;
}

/* Logo */
.logo {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  color: var(--text-primary);
  font-family: var(--font-display);
  font-size: var(--text-xl);
  font-weight: 600;
}

.logo:hover {
  color: var(--accent);
}

.logo-icon {
  font-size: 1.5em;
}

/* Nav Links */
.nav-links {
  display: flex;
  gap: var(--space-8);
}

.nav-link {
  color: var(--text-secondary);
  font-size: var(--text-base);
  padding: var(--space-2) 0;
  position: relative;
}

.nav-link::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 0;
  width: 0;
  height: 2px;
  background: var(--accent);
  transition: width var(--duration-base) var(--ease-out);
}

.nav-link:hover,
.nav-link.active {
  color: var(--text-primary);
}

.nav-link.active::after,
.nav-link:hover::after {
  width: 100%;
}

/* Actions */
.nav-actions {
  display: flex;
  align-items: center;
  gap: var(--space-4);
}

.btn-post {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-5);
  background: var(--accent);
  color: var(--bg-primary);
  font-weight: 500;
  border-radius: var(--radius-full);
  transition: all var(--duration-fast) var(--ease-out);
}

.btn-post:hover {
  background: var(--accent-light);
  transform: translateY(-1px);
  box-shadow: var(--shadow-glow-sm);
}

.btn-login {
  color: var(--text-secondary);
  padding: var(--space-2) var(--space-4);
}

.btn-login:hover {
  color: var(--accent);
}

/* User Menu */
.user-menu {
  position: relative;
}

.user-btn {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: var(--primary-light);
  display: flex;
  align-items: center;
  justify-content: center;
}

.avatar {
  color: var(--text-primary);
  font-weight: 600;
  font-size: var(--text-sm);
}

.dropdown-menu {
  position: absolute;
  top: calc(100% + 8px);
  right: 0;
  min-width: 160px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  overflow: hidden;
  box-shadow: var(--shadow-lg);
}

.dropdown-header {
  padding: var(--space-3) var(--space-4);
  color: var(--text-secondary);
  font-size: var(--text-sm);
  border-bottom: 1px solid var(--border);
}

.dropdown-item {
  width: 100%;
  padding: var(--space-3) var(--space-4);
  color: var(--text-primary);
  text-align: left;
  transition: background var(--duration-fast);
}

.dropdown-item:hover {
  background: var(--bg-hover);
}

/* Mobile Menu Toggle */
.menu-toggle {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.hamburger {
  width: 20px;
  height: 14px;
  position: relative;
}

.hamburger span {
  position: absolute;
  left: 0;
  width: 100%;
  height: 2px;
  background: var(--text-primary);
  transition: all var(--duration-fast) var(--ease-out);
}

.hamburger span:nth-child(1) { top: 0; }
.hamburger span:nth-child(2) { top: 6px; }
.hamburger span:nth-child(3) { top: 12px; }

.hamburger.open span:nth-child(1) {
  top: 6px;
  transform: rotate(45deg);
}

.hamburger.open span:nth-child(2) {
  opacity: 0;
}

.hamburger.open span:nth-child(3) {
  top: 6px;
  transform: rotate(-45deg);
}

/* Mobile Menu */
.mobile-menu {
  background: var(--bg-secondary);
  border-top: 1px solid var(--border);
  padding: var(--space-4);
}

.mobile-link {
  display: block;
  padding: var(--space-4);
  color: var(--text-primary);
  font-size: var(--text-lg);
  border-radius: var(--radius-md);
}

.mobile-link:hover,
.mobile-link.highlight {
  background: var(--bg-card);
}

.mobile-link.highlight {
  color: var(--accent);
}

/* Responsive */
.desktop-only {
  display: flex;
}

.mobile-only {
  display: none;
}

@media (max-width: 768px) {
  .desktop-only {
    display: none;
  }

  .mobile-only {
    display: flex;
  }

  .navbar-inner {
    padding: 0 var(--space-4);
  }
}

/* Transitions */
.dropdown-enter-active,
.dropdown-leave-active {
  transition: all var(--duration-fast) var(--ease-out);
}

.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}

.slide-down-enter-active,
.slide-down-leave-active {
  transition: all var(--duration-base) var(--ease-out);
}

.slide-down-enter-from,
.slide-down-leave-to {
  opacity: 0;
  transform: translateY(-10px);
}
</style>
