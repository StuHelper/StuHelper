<template>
  <el-config-provider :locale="elementLocale">
    <AppShell v-if="showShell">
      <router-view v-slot="{ Component, route }">
        <Transition name="page" mode="out-in">
          <component :is="Component" :key="route.path" />
        </Transition>
      </router-view>
    </AppShell>

    <router-view v-else v-slot="{ Component, route }">
      <Transition name="page" mode="out-in">
        <component :is="Component" :key="route.path" />
      </Transition>
    </router-view>
  </el-config-provider>
</template>

<script setup lang="ts">
import '@/styles/main.css'
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElConfigProvider } from 'element-plus'
import zhCn from 'element-plus/es/locale/lang/zh-cn'
import en from 'element-plus/es/locale/lang/en'
import { useThemeStore } from '@/stores/theme'
import AppShell from '@/components/layout/AppShell.vue'

const route = useRoute()
const { locale } = useI18n()

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
  animation: fadeInUp var(--duration-base) var(--ease-out);
}

.page-leave-active {
  animation: fadeIn var(--duration-fast) var(--ease-out) reverse;
}
</style>
