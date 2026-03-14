<template>
  <el-container class="h-screen">
    <!-- Sidebar -->
    <el-aside :width="collapsed ? '64px' : '220px'" class="transition-all duration-300 border-r border-gray-200">
      <div class="h-full flex flex-col bg-white">
        <!-- Logo -->
        <div class="h-14 flex items-center gap-2.5 px-4 border-b border-gray-100 shrink-0 overflow-hidden">
          <div class="w-8 h-8 rounded-lg bg-gradient-to-br from-blue-500 to-indigo-600 flex items-center justify-center shrink-0">
            <span class="text-white text-sm font-bold">S</span>
          </div>
          <span v-if="!collapsed" class="text-sm font-bold text-gray-900 whitespace-nowrap">StuHelper Admin</span>
        </div>

        <!-- Menu -->
        <el-scrollbar class="flex-1">
          <el-menu
            :default-active="activeMenu"
            :collapse="collapsed"
            :collapse-transition="false"
            router
            class="!border-r-0"
          >
            <template v-for="route in menuRoutes" :key="route.path">
              <!-- Has children -->
              <el-sub-menu v-if="route.children && route.children.length > 1" :index="route.path">
                <template #title>
                  <el-icon><component :is="route.meta?.icon" /></el-icon>
                  <span>{{ route.meta?.title }}</span>
                </template>
                <el-menu-item
                  v-for="child in route.children"
                  :key="child.path"
                  :index="child.path"
                >
                  {{ child.meta?.title }}
                </el-menu-item>
              </el-sub-menu>
              <!-- Single item -->
              <el-menu-item v-else :index="route.children?.[0]?.path || route.path">
                <el-icon><component :is="route.meta?.icon" /></el-icon>
                <template #title>{{ route.meta?.title }}</template>
              </el-menu-item>
            </template>
          </el-menu>
        </el-scrollbar>

        <!-- Bottom -->
        <div class="p-2 border-t border-gray-100 shrink-0 space-y-1">
          <button
            class="hidden md:flex items-center gap-3 w-full px-3 py-2 text-gray-500 text-sm rounded-lg hover:text-gray-700 hover:bg-gray-50 transition-colors"
            @click="collapsed = !collapsed"
          >
            <el-icon :size="18">
              <Fold v-if="!collapsed" />
              <Expand v-else />
            </el-icon>
            <span v-if="!collapsed" class="whitespace-nowrap">收起</span>
          </button>
          <a
            href="/"
            target="_blank"
            class="flex items-center gap-3 px-3 py-2 text-gray-500 text-sm rounded-lg hover:text-gray-700 hover:bg-gray-50 transition-colors no-underline"
          >
            <el-icon :size="18"><Link /></el-icon>
            <span v-if="!collapsed" class="whitespace-nowrap">返回前台</span>
          </a>
        </div>
      </div>
    </el-aside>

    <el-container>
      <!-- Header -->
      <el-header class="!h-14 flex items-center justify-between border-b border-gray-200 bg-white px-4">
        <div class="flex items-center gap-3">
          <!-- Mobile toggle -->
          <el-button :icon="Expand" text class="md:!hidden" @click="collapsed = !collapsed" />
          <!-- Breadcrumb -->
          <el-breadcrumb separator="/">
            <el-breadcrumb-item>管理后台</el-breadcrumb-item>
            <el-breadcrumb-item v-if="currentTitle">{{ currentTitle }}</el-breadcrumb-item>
          </el-breadcrumb>
        </div>

        <div class="flex items-center gap-3">
          <el-dropdown trigger="click" @command="handleCommand">
            <div class="flex items-center gap-2 cursor-pointer px-2 py-1 rounded-lg hover:bg-gray-50 transition-colors">
              <el-avatar :size="28" class="bg-gradient-to-br from-blue-500 to-indigo-600">
                {{ authStore.user?.name?.charAt(0)?.toUpperCase() || 'A' }}
              </el-avatar>
              <span class="hidden sm:inline text-sm text-gray-600">{{ authStore.user?.displayName || authStore.user?.name }}</span>
              <el-icon class="text-gray-400"><ArrowDown /></el-icon>
            </div>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="logout" divided>
                  <el-icon class="text-red-500"><SwitchButton /></el-icon>
                  <span class="text-red-500">退出登录</span>
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>

      <!-- Content -->
      <el-main class="!p-4 md:!p-6 bg-gray-50">
        <router-view v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <keep-alive :include="cachedViews">
              <component :is="Component" />
            </keep-alive>
          </transition>
        </router-view>
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import { Fold, Expand, Link, ArrowDown, SwitchButton } from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'
import { menuRoutes } from '@/router'

const authStore = useAuthStore()
const route = useRoute()
const collapsed = ref(false)
const cachedViews = ref<string[]>([])

const activeMenu = computed(() => route.path)
const currentTitle = computed(() => {
  const matched = route.matched
  const last = matched[matched.length - 1]
  return (last?.meta?.title as string) || ''
})

function handleCommand(command: string) {
  if (command === 'logout') {
    authStore.logout()
  }
}
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
