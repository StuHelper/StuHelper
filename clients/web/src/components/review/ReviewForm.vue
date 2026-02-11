<template>
  <div class="inline-form" :class="{ expanded: isExpanded }">
    <div v-if="!isExpanded" class="inline-form__collapsed" @click="expand">
      <div class="inline-form__avatar">
        <div class="avatar-circle" />
      </div>
      <span class="inline-form__placeholder">
        {{ t('review.course.shareExperience') }}
      </span>
    </div>

    <div v-else class="inline-form__expanded">
      <input
        v-model="title"
        class="inline-form__title"
        :placeholder="t('review.post.title')"
      />

      <RatingGroup v-model="ratings" />

      <textarea
        ref="textareaRef"
        v-model="content"
        class="inline-form__textarea"
        :placeholder="t('review.post.contentPlaceholder')"
        rows="4"
      />

      <div class="inline-form__footer">
        <button class="btn-cancel" @click="collapse">
          {{ t('common.actions.cancel') }}
        </button>
        <button
          class="btn-submit"
          :disabled="!canSubmit || submitting"
          @click="handleSubmit"
        >
          {{ submitting ? t('common.actions.loading') : t('review.post.submit') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, nextTick } from 'vue'
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

const isExpanded = ref(false)
const title = ref('')
const content = ref('')
const ratings = ref<ReviewRatings>({})
const submitting = ref(false)
const textareaRef = ref<HTMLTextAreaElement | null>(null)

const canSubmit = computed(() =>
  content.value.trim().length > 0 && Object.keys(ratings.value).length > 0
)

function expand() {
  isExpanded.value = true
  nextTick(() => textareaRef.value?.focus())
}

function collapse() {
  isExpanded.value = false
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
    collapse()
    emit('posted')
  } catch {
    toast.error(t('review.post.failed'))
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.inline-form {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  transition: border-color var(--duration-base) var(--ease-smooth);
}

.inline-form.expanded {
  border-color: var(--brand-primary);
  box-shadow: var(--shadow-glow-sm);
}

.inline-form__collapsed {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-4);
  cursor: pointer;
}

.avatar-circle {
  width: 32px;
  height: 32px;
  border-radius: var(--radius-full);
  background: var(--bg-tertiary);
  flex-shrink: 0;
}

.inline-form__placeholder {
  color: var(--text-muted);
  font-size: var(--text-sm);
}

.inline-form__expanded {
  padding: var(--space-5);
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  animation: fadeInUp var(--duration-base) var(--ease-out);
}

.inline-form__title {
  width: 100%;
  padding: var(--space-3);
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  color: var(--text-primary);
  font-family: var(--font-sans);
}

.inline-form__title:focus {
  outline: none;
  border-color: var(--brand-primary);
}

.inline-form__textarea {
  width: 100%;
  padding: var(--space-3);
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  color: var(--text-primary);
  font-family: var(--font-sans);
  resize: vertical;
  min-height: 100px;
}

.inline-form__textarea:focus {
  outline: none;
  border-color: var(--brand-primary);
}

.inline-form__footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-3);
}

.btn-cancel {
  padding: var(--space-2) var(--space-4);
  font-size: var(--text-sm);
  color: var(--text-secondary);
  border: 1px solid var(--border);
  border-radius: var(--radius-full);
  cursor: pointer;
}

.btn-submit {
  padding: var(--space-2) var(--space-5);
  font-size: var(--text-sm);
  font-weight: var(--weight-medium);
  color: white;
  background: var(--gradient-brand);
  border-radius: var(--radius-full);
  cursor: pointer;
}

.btn-submit:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
