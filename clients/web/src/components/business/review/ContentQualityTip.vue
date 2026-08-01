<template>
  <div
    class="p-3 rounded-sm mb-3 animate-fade-in"
    :class="level === 'error'
      ? 'border border-text-primary'
      : 'border-0'"
  >
    <div class="flex items-center gap-2 mb-2">
      <AlertCircle v-if="level === 'error'" class="w-[18px] h-[18px] text-text-primary" />
      <AlertTriangle v-else class="w-[18px] h-[18px] text-accent" />
      <span
        class="font-medium text-sm"
        :class="level === 'error' ? 'text-text-primary' : 'text-accent'"
      >
        {{ title }}
      </span>
    </div>
    <ul class="m-0 pl-6 text-sm text-text-secondary [&>li]:mb-1">
      <li v-for="(item, index) in issues" :key="index">
        {{ item }}
      </li>
    </ul>
    <button
      type="button"
      v-if="dismissible"
      class="mt-2 px-2 py-1 text-xs bg-transparent rounded-sm text-text-muted cursor-pointer hover:border-text-muted"
      @click="$emit('dismiss')"
    >
      {{ t('review.quality.dismiss') }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { AlertTriangle, AlertCircle } from 'lucide-vue-next'

const { t } = useI18n()

const props = withDefaults(defineProps<{
  issues: string[]
  level?: 'warning' | 'error'
  dismissible?: boolean
}>(), {
  level: 'warning',
  dismissible: true
})

defineEmits<{
  dismiss: []
}>()

const title = computed(() =>
  props.level === 'error' ? t('review.quality.errorTitle') : t('review.quality.warningTitle')
)
</script>
