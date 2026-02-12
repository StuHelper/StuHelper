<template>
  <div class="bg-bg-card border border-border rounded-xl shadow-card p-5 flex flex-col gap-4">
    <input
      v-model="title"
      class="w-full p-3 bg-bg-secondary border border-border rounded-lg text-sm text-text-primary font-sans transition-[border-color,box-shadow] duration-fast focus:outline-none focus:border-primary focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] focus:bg-bg-card"
      :placeholder="t('review.post.title')"
      :aria-label="t('review.post.titleLabel')"
      :maxlength="TITLE_MAX"
    />
    <span class="block text-right text-xs text-text-muted -mt-2">
      {{ t('review.validation.charCount', { current: title.length, max: TITLE_MAX }) }}
    </span>

    <RatingGroup v-model="ratings" />

    <textarea
      ref="textareaRef"
      v-model="content"
      class="w-full p-3 bg-bg-secondary border border-border rounded-lg text-sm text-text-primary font-sans resize-vertical min-h-[100px] transition-[border-color,box-shadow] duration-fast focus:outline-none focus:border-primary focus:shadow-[0_0_0_3px_rgba(59,130,246,0.1)] focus:bg-bg-card"
      :placeholder="t('review.post.contentPlaceholder')"
      :aria-label="t('review.post.contentLabel')"
      :aria-describedby="contentError ? 'review-form-content-error' : undefined"
      :maxlength="CONTENT_MAX"
      rows="4"
    />
    <span
      class="block text-right text-xs text-text-muted -mt-2"
      :class="{ '!text-danger': content.length < CONTENT_MIN && content.length > 0 }"
    >
      {{ t('review.validation.charCount', { current: content.length, max: CONTENT_MAX }) }}
    </span>
    <span v-if="contentError" id="review-form-content-error" class="block text-xs text-danger -mt-2">{{ contentError }}</span>

    <div class="flex justify-end">
      <button
        class="py-2 px-5 text-sm font-medium text-white bg-gradient-to-br from-primary to-accent rounded-full cursor-pointer transition-all duration-fast hover:not-disabled:opacity-90 hover:not-disabled:-translate-y-px disabled:opacity-50 disabled:cursor-not-allowed"
        :disabled="!canSubmit || submitting"
        @click="handleSubmit"
      >
        {{ submitting ? t('common.actions.loading') : t('review.post.submit') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { postReview } from '@/api/review'
import { useToast } from '@/composables/useToast'
import RatingGroup from './RatingGroup.vue'
import type { ReviewRatings } from '@/types/review'

const props = defineProps<{
  courseID: number
}>()

const emit = defineEmits<{ posted: [] }>()

const { t } = useI18n()
const toast = useToast()

const TITLE_MAX = 100
const CONTENT_MIN = 10
const CONTENT_MAX = 5000

const title = ref('')
const content = ref('')
const ratings = ref<ReviewRatings>({})
const submitting = ref(false)
const textareaRef = ref<HTMLTextAreaElement | null>(null)

const contentError = computed(() => {
  const len = content.value.trim().length
  if (len > 0 && len < CONTENT_MIN) {
    return t('review.validation.contentTooShort', { min: CONTENT_MIN })
  }
  return ''
})

const canSubmit = computed(() =>
  content.value.trim().length >= CONTENT_MIN &&
  content.value.length <= CONTENT_MAX &&
  title.value.length <= TITLE_MAX &&
  Object.keys(ratings.value).length > 0 &&
  Object.values(ratings.value).every((v) => v >= 1 && v <= 5)
)

function reset() {
  title.value = ''
  content.value = ''
  ratings.value = {}
}

async function handleSubmit() {
  if (!canSubmit.value) return
  submitting.value = true
  try {
    await postReview({
      courseID: props.courseID,
      title: title.value.trim() || undefined,
      content: content.value.trim(),
      ratings: ratings.value
    })
    toast.success(t('review.post.success'))
    reset()
    emit('posted')
  } catch {
    toast.error(t('review.post.failed'))
  } finally {
    submitting.value = false
  }
}
</script>
