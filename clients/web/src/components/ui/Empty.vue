<script setup lang="ts">
import { InboxIcon } from '@heroicons/vue/24/outline'

interface Props {
  title: string
  description?: string
  actionText?: string
}

withDefaults(defineProps<Props>(), {
  description: '',
  actionText: ''
})

const emit = defineEmits<{
  action: []
}>()
</script>

<template>
  <div class="flex flex-col items-center justify-center py-16 px-4" role="status">
    <InboxIcon class="w-24 h-24 mb-6 text-gray-400 dark:text-gray-500" aria-hidden="true" />
    <h3 class="text-xl font-semibold text-gray-600 dark:text-gray-300 mb-2">{{ title }}</h3>
    <p v-if="description" class="text-gray-500 dark:text-gray-400 text-center mb-6 max-w-md">{{ description }}</p>
    <button
      v-if="actionText"
      type="button"
      @click="emit('action')"
      class="px-6 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
    >
      {{ actionText }}
    </button>
  </div>
</template>
