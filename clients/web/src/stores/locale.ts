/**
 * 语言状态管理
 */
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import i18n, {
  DEFAULT_LOCALE,
  LOCALE_STORAGE_KEY,
  SUPPORTED_LOCALES,
  normalizeLocale,
  readStoredLocale,
  type SupportedLocale
} from '@/i18n'
import { updateLocaleMeta } from '@/composables/usePageMeta'
import { safeSetLocalStorageItem } from '@/utils/browserStorage'

export const useLocaleStore = defineStore('locale', () => {
  function getI18nLocale(): SupportedLocale {
    return normalizeLocale(i18n.global.locale.value) ?? DEFAULT_LOCALE
  }

  const initialLocale = readStoredLocale() ?? getI18nLocale()

  const locale = ref<SupportedLocale>(initialLocale)

  // 是否为中文
  const isZhCN = computed(() => locale.value === 'zh-CN')

  // 是否为英文
  const isEnUS = computed(() => locale.value === 'en-US')

  // 根据语言动态更新 meta 标签
  function updateMetaTags(loc: SupportedLocale) {
    const { t } = i18n.global
    updateLocaleMeta(loc, t('common.meta.description'), t('common.meta.ogTitle'))
  }

  // 初始化时同步 i18n、HTML lang 属性和 meta 标签
  i18n.global.locale.value = locale.value
  updateMetaTags(locale.value)

  // 切换语言
  function setLocale(newLocale: SupportedLocale) {
    if (!SUPPORTED_LOCALES.includes(newLocale)) {
      return
    }

    locale.value = newLocale
    safeSetLocalStorageItem(LOCALE_STORAGE_KEY, newLocale)
    i18n.global.locale.value = newLocale
    updateMetaTags(newLocale)
  }

  // 切换语言（在中英文之间切换）
  function toggleLocale() {
    setLocale(locale.value === 'zh-CN' ? 'en-US' : 'zh-CN')
  }

  return {
    locale,
    isZhCN,
    isEnUS,
    setLocale,
    toggleLocale
  }
})
