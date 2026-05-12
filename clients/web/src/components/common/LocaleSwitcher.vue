<template>
  <button
    class="group relative inline-flex size-11 shrink-0 cursor-pointer items-center justify-center rounded-xl border border-transparent text-text-secondary transition-all duration-fast hover:border-white/20 hover:bg-bg-card hover:text-text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
    :aria-label="isZhCN ? t('common.locale.switchToEnglish') : t('common.locale.switchToChinese')"
    :title="isZhCN ? t('common.locale.switchToEnglish') : t('common.locale.switchToChinese')"
    @click="toggleLocale"
  >
    <Languages class="size-[18px]" aria-hidden="true" />
    <span
      class="absolute -right-0.5 -bottom-0.5 min-w-4 rounded-md bg-bg-base px-1 text-[10px] font-bold leading-4 text-primary shadow-xs ring-1 ring-white/40"
      aria-hidden="true"
    >
      {{ isZhCN ? t('common.locale.englishShort') : t('common.locale.chineseShort') }}
    </span>
  </button>
</template>

<script setup lang="ts">
import { useLocaleStore } from '@/stores/locale'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import { Languages } from 'lucide-vue-next'

const localeStore = useLocaleStore()
const { isZhCN } = storeToRefs(localeStore)
const { toggleLocale } = localeStore
const { t } = useI18n()
</script>
