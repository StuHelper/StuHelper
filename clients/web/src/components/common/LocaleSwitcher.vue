<template>
  <button
    class="w-8 h-8 rounded-full flex items-center justify-center text-xs font-semibold cursor-pointer shrink-0 transition-all duration-fast text-text-secondary hover:text-text-primary hover:bg-bg-hover"
    :aria-label="isZhCN ? t('common.locale.switchToEnglish') : t('common.locale.switchToChinese')"
    :title="isZhCN ? t('common.locale.switchToEnglish') : t('common.locale.switchToChinese')"
    @click="toggleLocale"
  >
    {{ isZhCN ? t('common.locale.englishShort') : t('common.locale.chineseShort') }}
  </button>
</template>

<script setup lang="ts">
import { useLocaleStore } from '@/stores/locale'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'

const localeStore = useLocaleStore()
const { isZhCN } = storeToRefs(localeStore)
const { toggleLocale } = localeStore
const { t } = useI18n()
</script>
