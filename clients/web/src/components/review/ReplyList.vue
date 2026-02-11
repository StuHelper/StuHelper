<template>
  <div class="reply-list">
    <div class="reply-header">
      <button class="toggle-btn" @click="toggleExpand">
        <span>{{ t('review.reply.count', { count: replies.length }) }}</span>
        <svg
          class="chevron"
          :class="{ expanded }"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
        >
          <path d="M19 9l-7 7-7-7" stroke-width="2" />
        </svg>
      </button>
      <button
        v-if="!showForm"
        class="reply-btn"
        @click="showForm = true"
      >
        {{ t('common.actions.reply') }}
      </button>
    </div>

    <transition name="collapse">
      <div v-if="expanded" class="reply-content">
        <div v-if="error" class="error-msg">{{ error }}</div>
        <ReplyForm
          v-if="showForm"
          :submitting="submitting"
          @submit="handleSubmit"
          @cancel="showForm = false"
        />

        <div v-if="loading" class="loading">
          <span class="spinner" />
        </div>

        <template v-else-if="replies.length > 0">
          <ReplyCard
            v-for="reply in replies"
            :key="reply.id"
            :reply="reply"
            @delete="handleDelete"
          />
        </template>

        <EmptyState
          v-else
          :title="t('review.reply.emptyTitle')"
          :description="t('review.reply.emptyDesc')"
        />
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Reply } from '@/types/reply'
import { getReplies, postReply, deleteReply } from '@/api/review'
import ReplyCard from './ReplyCard.vue'
import ReplyForm from './ReplyForm.vue'
import EmptyState from '@/components/common/EmptyState.vue'

const { t } = useI18n()

const props = defineProps<{
  reviewID: number
}>()

const expanded = ref(false)
const showForm = ref(false)
const loading = ref(false)
const submitting = ref(false)
const replies = ref<Reply[]>([])
const error = ref('')

const toggleExpand = () => {
  expanded.value = !expanded.value
  if (expanded.value && replies.value.length === 0) {
    fetchReplies()
  }
}

const fetchReplies = async () => {
  loading.value = true
  try {
    const res = await getReplies(props.reviewID)
    replies.value = res.data?.list || []
  } catch {
    // 回复列表加载失败，UI 显示空状态
  } finally {
    loading.value = false
  }
}

const handleSubmit = async (content: string) => {
  submitting.value = true
  error.value = ''
  try {
    const res = await postReply({
      reviewID: props.reviewID,
      content
    })
    if (res.data) {
      replies.value.unshift(res.data)
    }
    showForm.value = false
  } catch {
    error.value = t('review.reply.sendFailed')
  } finally {
    submitting.value = false
  }
}

const handleDelete = async (id: number) => {
  error.value = ''
  try {
    await deleteReply(id)
    replies.value = replies.value.filter(r => r.id !== id)
  } catch {
    error.value = t('review.reply.deleteFailed')
  }
}
</script>

<style scoped>
.reply-list {
  margin-top: var(--space-3);
  border-top: 1px solid var(--border);
  padding-top: var(--space-3);
}

.reply-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.toggle-btn {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  background: none;
  border: none;
  color: var(--text-muted);
  font-size: var(--text-sm);
  cursor: pointer;
  padding: var(--space-1) 0;
}

.toggle-btn:hover {
  color: var(--text-secondary);
}

.chevron {
  width: 16px;
  height: 16px;
  transition: transform var(--duration-base) ease;
}

.chevron.expanded {
  transform: rotate(180deg);
}

.reply-btn {
  font-size: var(--text-sm);
  color: var(--brand-primary);
  background: none;
  border: none;
  cursor: pointer;
  padding: var(--space-1) var(--space-2);
}

.reply-btn:hover {
  text-decoration: underline;
}

.reply-content {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  margin-top: var(--space-3);
}

.loading {
  display: flex;
  justify-content: center;
  padding: var(--space-4);
}

.spinner {
  width: 24px;
  height: 24px;
  border: 2px solid var(--border);
  border-top-color: var(--brand-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.collapse-enter-active,
.collapse-leave-active {
  transition: all var(--duration-base) ease;
  overflow: hidden;
}

.collapse-enter-from,
.collapse-leave-to {
  opacity: 0;
  max-height: 0;
}

.collapse-enter-to,
.collapse-leave-from {
  opacity: 1;
  max-height: 1000px;
}
</style>
