<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { MagnifyingGlassIcon, XMarkIcon } from '@heroicons/vue/24/outline'
import { useDebounceFn } from '@vueuse/core'

const { t } = useI18n()

interface Props {
  modelValue: string
  placeholder?: string
  debounce?: number
}

const props = withDefaults(defineProps<Props>(), {
  placeholder: undefined,
  debounce: 300
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const localValue = ref(props.modelValue)

const debouncedEmit = useDebounceFn((value: string) => {
  emit('update:modelValue', value)
}, props.debounce)

watch(localValue, (newValue) => {
  debouncedEmit(newValue)
})

watch(() => props.modelValue, (newValue) => {
  if (localValue.value !== newValue) {
    localValue.value = newValue
  }
})

const handleClear = () => {
  localValue.value = ''
  emit('update:modelValue', '')
}
</script>

<template>
  <div class="relative">
    <div class="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
      <MagnifyingGlassIcon class="h-5 w-5 text-gray-400 dark:text-gray-500" aria-hidden="true" />
    </div>
    <input
      v-model="localValue"
      type="search"
      :placeholder="placeholder ?? t('common.search.placeholder')"
      class="w-full pl-12 pr-12 py-3 rounded-xl bg-white/10 backdrop-blur-md border border-white/20 text-gray-900 placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-primary/50 transition-all dark:bg-gray-800/50 dark:border-gray-700/50 dark:text-white dark:placeholder-gray-400"
      :aria-label="t('common.search.label')"
    />
    <button
      v-if="localValue"
      type="button"
      @click="handleClear"
      class="absolute inset-y-0 right-0 pr-4 flex items-center text-gray-400 hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300 transition-colors"
      :aria-label="t('common.search.clear')"
    >
      <XMarkIcon class="h-5 w-5" aria-hidden="true" />
    </button>
  </div>
</template>

<style scoped>
input[type="search"]::-webkit-search-cancel-button {
  -webkit-appearance: none;
}
</style>
