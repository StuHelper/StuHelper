<template>
  <button
    class="w-8 h-8 rounded-full flex items-center justify-center text-xs font-semibold cursor-pointer shrink-0 transition-all duration-fast text-text-secondary hover:text-text-primary hover:bg-bg-hover"
    :aria-label="isZhCN ? 'Switch to English' : '切换到中文'"
    :title="isZhCN ? 'Switch to English' : '切换到中文'"
    @click="toggleLocale"
  >
    {{ isZhCN ? 'EN' : '中' }}
  </button>
</template>

<script setup lang="ts">
import { useLocaleStore } from '@/stores/locale'
import { storeToRefs } from 'pinia'

const localeStore = useLocaleStore()
const { isZhCN } = storeToRefs(localeStore)
const { toggleLocale } = localeStore
</script>
