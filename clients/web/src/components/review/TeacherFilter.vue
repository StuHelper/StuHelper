<template>
  <div v-if="teachers.length > 0" class="flex items-center gap-2 flex-wrap">
    <span class="text-xs text-text-muted font-medium shrink-0">{{ t('review.filter.teacher') }}</span>
    <button
      class="py-1 px-3 text-xs rounded-full border transition-all duration-fast ease-smooth cursor-pointer"
      :class="!modelValue
        ? 'bg-primary text-white border-primary'
        : 'bg-transparent text-text-secondary border-border hover:border-primary hover:text-primary'"
      @click="emit('update:modelValue', '')"
    >
      {{ t('review.filter.all') }}
    </button>
    <button
      v-for="teacher in teachers"
      :key="teacher"
      class="py-1 px-3 text-xs rounded-full border transition-all duration-fast ease-smooth cursor-pointer"
      :class="modelValue === teacher
        ? 'bg-primary text-white border-primary'
        : 'bg-transparent text-text-secondary border-border hover:border-primary hover:text-primary'"
      @click="emit('update:modelValue', teacher)"
    >
      {{ teacher }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineProps<{
  teachers: string[]
  modelValue: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const { t } = useI18n()
</script>
