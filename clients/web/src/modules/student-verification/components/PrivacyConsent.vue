<template>
  <section class="rounded-xl border border-border bg-bg-elevated/60 p-4">
    <button
      type="button"
      class="flex min-h-11 w-full items-center justify-between gap-3 border-0 bg-transparent p-0 text-left text-text-primary"
      :aria-expanded="expanded"
      @click="expanded = !expanded"
    >
      <span>
        <span class="block text-sm font-bold">{{ notice.title }}</span>
        <span class="mt-1 block text-xs font-normal text-text-muted">{{ notice.summary }}</span>
      </span>
      <ChevronDown
        class="size-4 shrink-0 transition-transform"
        :class="expanded ? 'rotate-180' : ''"
        aria-hidden="true"
      />
    </button>

    <div v-if="expanded" class="mt-3 border-t border-border pt-3 text-xs leading-5 text-text-secondary">
      <p class="m-0 font-semibold text-text-primary">
        {{ t('user.verification.student.platform.privacy.dataUsed') }}
      </p>
      <ul class="mb-0 mt-2 list-disc space-y-1 pl-5">
        <li v-for="category in notice.dataCategories" :key="category">{{ category }}</li>
      </ul>
      <p class="mb-0 mt-3">{{ notice.retentionSummary }}</p>
      <a
        v-if="notice.rightsUrl"
        class="mt-2 inline-flex min-h-11 items-center font-semibold text-primary underline-offset-4 hover:underline"
        :href="notice.rightsUrl"
        target="_blank"
        rel="noopener noreferrer"
      >
        {{ t('user.verification.student.platform.privacy.learnMore') }}
      </a>
    </div>

    <label class="mt-4 flex cursor-pointer items-start gap-3 border-t border-border pt-4 text-sm leading-6 text-text-secondary">
      <input
        :checked="modelValue"
        data-verification-consent
        name="verification_consent"
        type="checkbox"
        class="mt-1 size-4 shrink-0 accent-primary"
        @change="updateValue"
      />
      <span>{{ t('user.verification.student.platform.privacy.consent') }}</span>
    </label>
  </section>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ChevronDown } from 'lucide-vue-next'
import type { VerificationSchool } from '@stuhelper/shared/api'

type PrivacyNotice = NonNullable<VerificationSchool['methods'][number]['privacyNotice']>

defineProps<{
  modelValue: boolean
  notice: PrivacyNotice
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

const { t } = useI18n()
const expanded = ref(false)

function updateValue(event: Event): void {
  const target = event.target
  if (target instanceof HTMLInputElement) emit('update:modelValue', target.checked)
}
</script>
