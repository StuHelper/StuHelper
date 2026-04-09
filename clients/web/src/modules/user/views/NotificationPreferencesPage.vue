<template>
  <div class="max-w-[600px] mx-auto p-6 animate-fade-in">
    <div class="flex items-center gap-3 mb-6 pb-4 border-b border-border-light">
      <router-link to="/notifications" class="text-text-muted hover:text-text-primary transition-colors duration-fast">
        <ArrowLeft :size="20" />
      </router-link>
      <h1 class="font-sans text-xl font-extrabold tracking-tight text-text-primary m-0">
        {{ t('user.notification.preferences') }}
      </h1>
    </div>
    <p class="text-sm text-text-muted mb-4">{{ t('user.notification.preferencesDesc') }}</p>

    <div v-if="loading" class="flex items-center justify-center p-6 text-text-muted text-sm">
      <span class="w-5 h-5 border-2 border-border border-t-text-primary rounded-full animate-spin" />
    </div>
    <div v-else class="flex flex-col gap-1">
      <label
        v-for="item in preferenceItems"
        :key="item.type"
        class="flex items-center justify-between p-3 rounded-sm hover:bg-bg-secondary transition-colors duration-fast cursor-pointer"
      >
        <div>
          <p class="text-sm text-text-primary m-0">{{ item.label }}</p>
        </div>
        <input
          type="checkbox"
          :checked="item.enabled"
          class="w-4 h-4 accent-[var(--color-accent)] cursor-pointer"
          @change="togglePreference(item.type, !item.enabled)"
        >
      </label>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowLeft } from 'lucide-vue-next'
import { api } from '@/api'

const { t } = useI18n()
const loading = ref(true)

interface Preference {
  type: string
  enabled: boolean
}

const preferences = ref<Preference[]>([])

// 用户可配置的通知类型（排除内部类型如 vote/system）
const configurableTypes = [
  'reply',
  'like',
  'review_hidden',
  'review_restored',
  'report_resolved',
  'identity_approved',
  'identity_rejected',
  'student_approved',
  'student_rejected',
] as const

const preferenceItems = computed(() =>
  configurableTypes.map(type => {
    const pref = preferences.value.find(p => p.type === type)
    return {
      type,
      label: t(`user.notification.items.${type}.label`),
      enabled: pref ? pref.enabled : true, // 默认启用
    }
  }),
)

const fetchPreferences = async () => {
  loading.value = true
  try {
    const res = await api.notification.getPreferences()
    preferences.value = (res.data?.data?.preferences || []) as Preference[]
  } catch {
    // 加载失败时保持全部默认启用
  } finally {
    loading.value = false
  }
}

const togglePreference = async (type: string, enabled: boolean) => {
  // 乐观更新
  const idx = preferences.value.findIndex(p => p.type === type)
  if (idx >= 0) {
    preferences.value = preferences.value.map(p => p.type === type ? { ...p, enabled } : p)
  } else {
    preferences.value = [...preferences.value, { type, enabled }]
  }

  try {
    await api.notification.updatePreference(type, enabled)
  } catch {
    // 回滚
    if (idx >= 0) {
      preferences.value = preferences.value.map(p => p.type === type ? { ...p, enabled: !enabled } : p)
    } else {
      preferences.value = preferences.value.filter(p => p.type !== type)
    }
  }
}

onMounted(fetchPreferences)
</script>
