<template>
  <el-config-provider :locale="elementLocale">
    <ErrorBoundary>
      <AppShell v-if="showShell">
        <router-view v-slot="{ Component }">
          <Transition name="page" mode="out-in">
            <component :is="Component" :key="transitionKey" />
          </Transition>
        </router-view>
      </AppShell>

      <router-view v-else v-slot="{ Component }">
        <Transition name="page" mode="out-in">
          <component :is="Component" :key="transitionKey" />
        </Transition>
      </router-view>
    </ErrorBoundary>
  </el-config-provider>
</template>

<script setup lang="ts">
import '@/styles/tailwind.css'
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

// L-17: 提取为 computed，消除模板中重复的 matched[0] 访问
const transitionKey = computed(() => route.matched[0]?.path || route.path)

// 初始化主题
useThemeStore()

const elementLocale = computed(() => {
  return locale.value === 'zh-CN' ? zhCn : en
})

const showShell = computed(() => {
  const layout = route.meta.layout
  return layout !== 'none' && layout !== 'admin'
})
</script>

<style>
.page-enter-active {
  animation: fade-in-up var(--duration-base) var(--ease-out);
}

.page-leave-active {
  animation: fade-in var(--duration-fast) var(--ease-out) reverse;
}
</style>
