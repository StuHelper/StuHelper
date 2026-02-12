<template>
  <div class="mt-3 border-t border-border pt-3">
    <div class="flex items-center justify-between">
      <button
        class="flex items-center gap-1 bg-transparent border-none text-text-muted text-sm cursor-pointer py-1 hover:text-text-secondary"
        :aria-expanded="expanded"
        @click="toggleExpand"
      >
        <span>{{ t('review.reply.count', { count: displayCount }) }}</span>
        <ChevronDown
          :size="16"
          class="transition-transform duration-base ease-linear"
          :class="{ 'rotate-180': expanded }"
        />
      </button>
      <button
        v-if="!showForm"
        class="text-sm text-primary bg-transparent border-none cursor-pointer py-1 px-2 hover:underline"
        @click="showForm = true"
      >
        {{ t('common.actions.reply') }}
      </button>
    </div>

    <transition
      @enter="onEnter"
      @after-enter="onAfterEnter"
      @leave="onLeave"
      @after-leave="onAfterLeave"
    >
      <div v-if="expanded" class="flex flex-col gap-3 mt-3">
        <ReplyForm
          v-if="showForm"
          :submitting="submitting"
          @submit="handleSubmit"
          @cancel="showForm = false"
        />

        <div v-if="loading" class="flex justify-center py-4">
          <span class="w-6 h-6 border-2 border-border border-t-primary rounded-full animate-spin" />
        </div>

        <div v-else-if="error" class="flex items-center gap-3 p-3 text-danger text-sm">
          {{ error }}
          <button
            class="py-1 px-3 bg-transparent border border-border rounded-sm text-text-secondary text-sm cursor-pointer shrink-0 hover:border-text-primary hover:text-text-primary"
            @click="fetchReplies"
          >
            {{ t('common.actions.retry') }}
          </button>
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
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ChevronDown } from 'lucide-vue-next'
import type { Reply } from '@/types/reply'
import { getReplies, postReply, deleteReply } from '@/api/review'
import ReplyCard from './ReplyCard.vue'
import ReplyForm from './ReplyForm.vue'
import EmptyState from '@/components/common/EmptyState.vue'

const { t } = useI18n()

const props = defineProps<{
  reviewID: string
  replyCount?: number
}>()

const expanded = ref(false)
const showForm = ref(false)
const loading = ref(false)
const submitting = ref(false)
const replies = ref<Reply[]>([])
const error = ref('')

// 展开前使用父组件传入的 replyCount，展开后使用实际加载的数据长度
const displayCount = computed(() =>
  expanded.value ? replies.value.length : (props.replyCount ?? 0)
)

const toggleExpand = () => {
  expanded.value = !expanded.value
  if (expanded.value && replies.value.length === 0 && !loading.value) {
    fetchReplies()
  }
}

const fetchReplies = async () => {
  loading.value = true
  error.value = ''
  try {
    const res = await getReplies(props.reviewID)
    replies.value = res.data?.list || []
  } catch {
    error.value = t('review.reply.loadFailed')
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

// collapse transition hooks — 动态计算高度替代 max-height hack
const onEnter = (el: Element) => {
  const htmlEl = el as HTMLElement
  htmlEl.style.overflow = 'hidden'
  htmlEl.style.height = '0'
  htmlEl.offsetHeight // force reflow
  htmlEl.style.transition = 'height var(--duration-base) ease, opacity var(--duration-base) ease'
  htmlEl.style.height = htmlEl.scrollHeight + 'px'
  htmlEl.style.opacity = '1'
}

const onAfterEnter = (el: Element) => {
  const htmlEl = el as HTMLElement
  htmlEl.style.height = ''
  htmlEl.style.overflow = ''
  htmlEl.style.transition = ''
}

const onLeave = (el: Element) => {
  const htmlEl = el as HTMLElement
  htmlEl.style.overflow = 'hidden'
  htmlEl.style.height = htmlEl.scrollHeight + 'px'
  htmlEl.offsetHeight // force reflow
  htmlEl.style.transition = 'height var(--duration-base) ease, opacity var(--duration-base) ease'
  htmlEl.style.height = '0'
  htmlEl.style.opacity = '0'
}

const onAfterLeave = (el: Element) => {
  const htmlEl = el as HTMLElement
  htmlEl.style.height = ''
  htmlEl.style.overflow = ''
  htmlEl.style.transition = ''
  htmlEl.style.opacity = ''
}

const handleDelete = async (id: string) => {
  error.value = ''
  try {
    await deleteReply(id)
    replies.value = replies.value.filter(r => r.id !== id)
  } catch {
    error.value = t('review.reply.deleteFailed')
  }
}
</script>
