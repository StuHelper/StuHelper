<template>
  <el-config-provider :locale="elementLocale">
    <ErrorBoundary>
      <AppShell v-if="showShell">
        <router-view v-slot="{ Component }">
          <Transition name="page" mode="out-in">
            <component :is="Component" :key="routeViewKey" />
          </Transition>
        </router-view>
      </AppShell>
      <router-view v-else v-slot="{ Component }">
        <Transition name="page" mode="out-in">
          <component :is="Component" :key="routeViewKey" />
        </Transition>
      </router-view>
    </ErrorBoundary>
  </el-config-provider>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElConfigProvider } from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import en from 'element-plus/es/locale/lang/en'
import { useThemeStore } from '@/stores/theme'
import AppShell from '@/components/layout/AppShell.vue'
import ErrorBoundary from '@/components/common/ErrorBoundary.vue'

const route = useRoute()
const { locale } = useI18n()

useThemeStore()

const elementLocale = computed(() => locale.value === 'zh-CN' ? zhCn : en)
const showShell = computed(() => {
  return route.meta.layout !== 'none'
})
const routeViewKey = computed(() => {
  const name = typeof route.name === 'string' ? route.name : ''
  if (name === 'login' || name === 'auth-callback') {
    return route.fullPath
  }
  if (name.startsWith('user-') && ['user-reviews', 'user-votes', 'user-favorites'].includes(name)) {
    return 'user-center'
  }
  return name || route.fullPath
})
</script>
