<template>
  <CourseThemeProvider>
    <div
      class="max-w-[900px] mx-auto px-4 py-6"
      :class="[
        isPanelMode ? '!max-w-none !p-0' : '',
        contentReady ? 'animate-fade-in' : 'opacity-0'
      ]"
    >
      <!-- Loading -->
      <div v-if="loading" class="flex flex-col gap-6" role="status" aria-busy="true" :aria-label="t('common.actions.loading')">
        <div class="h-[180px] rounded-xl bg-bg-elevated animate-shimmer bg-[length:200%_100%] bg-[linear-gradient(90deg,transparent_25%,var(--color-bg-secondary)_50%,transparent_75%)]" />
        <div class="h-[44px] rounded-xl bg-bg-elevated animate-shimmer bg-[length:200%_100%] bg-[linear-gradient(90deg,transparent_25%,var(--color-bg-secondary)_50%,transparent_75%)]" />
        <div class="h-[300px] rounded-xl bg-bg-elevated animate-shimmer bg-[length:200%_100%] bg-[linear-gradient(90deg,transparent_25%,var(--color-bg-secondary)_50%,transparent_75%)]" />
      </div>

      <!-- Error -->
      <div v-else-if="error" class="text-center py-12">
        <p class="mb-4 text-text-muted">{{ t('common.loadFailed') }}</p>
        <button
          class="py-2 px-6 text-sm font-medium text-white bg-primary rounded-full transition-opacity duration-fast hover:opacity-90"
          @click="fetchAll()"
        >
          {{ t('common.actions.retry') }}
        </button>
      </div>

      <template v-else-if="course">
        <!-- ========== 1. Course Header ========== -->
        <header class="flex items-center gap-3 flex-wrap mb-6">
          <h4 class="text-xl font-bold m-0 text-text-primary">
            {{ course.name }}
          </h4>
          <span
            v-if="course.departmentName"
            class="inline-flex items-center text-xs font-medium px-2.5 py-0.5 rounded-full bg-primary/10 text-primary"
          >
            {{ course.departmentName }}
          </span>
          <span
            v-if="total > 0"
            class="inline-flex items-center text-xs font-medium px-2.5 py-0.5 rounded-full bg-primary/10 text-primary"
          >
            {{ total }} {{ t('review.course.reviewUnit') }}
          </span>
          <div class="flex items-center gap-2 ml-auto">
            <FavoriteButton :course-i-d="courseID" />
            <button
              class="py-1.5 px-5 text-sm font-medium text-white bg-accent rounded-full transition-opacity duration-fast hover:opacity-90"
              @click="goToPostPage"
            >
              {{ t('review.hub.postReview') }}
            </button>
          </div>
        </header>

        <!-- ========== 2. Rating Section ========== -->
        <section class="mb-6">
          <h3 class="text-sm font-semibold uppercase tracking-wider text-text-muted mb-4">
            {{ t('review.detail.ratingTitle') }} &gt;
          </h3>

          <!-- No data at all -->
          <div
            v-if="!ratingStats || !ratingStats.byTerm || ratingStats.byTerm.length === 0"
            class="bg-bg-card rounded-xl shadow-card p-8 text-center"
          >
            <p class="text-text-muted mb-3">{{ t('review.detail.noData') }}</p>
            <button
              class="py-2 px-6 text-sm font-medium text-white bg-accent rounded-full transition-opacity duration-fast hover:opacity-90"
              @click="goToPostPage"
            >
              {{ t('review.detail.writeFirst') }}
            </button>
          </div>

          <!-- Semester rating cards grid -->
          <div
            v-else
            class="grid gap-4"
            style="grid-template-columns: repeat(auto-fill, minmax(260px, 1fr))"
          >
            <div
              v-for="(term, idx) in ratingStats.byTerm"
              :key="term.termID ?? term.termName"
              class="glass-card rounded-xl shadow-card p-4 stagger-item"
              :style="{ animationDelay: `${idx * 80}ms` }"
            >
              <!-- Semester header -->
              <div class="flex items-center gap-2 mb-3">
                <span class="text-sm font-bold text-text-primary">
                  {{ term.termName }}
                </span>
                <span
                  v-if="termReviewCount(term) > 0"
                  class="inline-flex items-center text-[0.65rem] font-medium px-2 py-0.5 h-[18px] rounded-full bg-primary/10 text-primary"
                >
                  {{ termReviewCount(term) }} {{ t('review.course.reviewUnit') }}
                </span>
                <span
                  v-if="termReviewCount(term) > 0 && termReviewCount(term) < 3"
                  class="inline-flex items-center text-[0.65rem] font-medium px-2 py-0.5 h-[18px] rounded-full bg-warning/15 text-warning"
                >
                  {{ t('review.detail.insufficientData') }}
                </span>
              </div>

              <!-- Insufficient data message -->
              <div v-if="termReviewCount(term) > 0 && termReviewCount(term) < 3" class="mb-2">
                <p class="text-xs m-0 text-text-muted">
                  {{ t('review.detail.insufficientHint') }}
                </p>
              </div>

              <!-- Rating bars (4 dimensions) -->
              <div v-if="term.dimensions && term.dimensions.length > 0" class="flex flex-col gap-2.5">
                <div v-for="dim in term.dimensions" :key="dim.key" class="flex items-center gap-2">
                  <span class="text-xs w-16 shrink-0 text-right text-text-secondary">
                    {{ dimensionLabel(dim.key) }}
                  </span>
                  <div class="flex-1 h-2 bg-bg-secondary rounded-full overflow-hidden">
                    <div
                      class="h-full rounded-full transition-all duration-slow"
                      :style="{
                        width: `${(dim.avgRating / 5) * 100}%`,
                        backgroundColor: ratingBarColor(dim.avgRating)
                      }"
                    />
                  </div>
                  <span class="text-xs font-mono w-8 text-right" :style="{ color: ratingBarColor(dim.avgRating) }">
                    {{ dim.avgRating.toFixed(1) }}
                  </span>
                </div>
              </div>

              <!-- No dimensions -->
              <div v-else class="text-center py-2">
                <p class="text-xs text-text-muted">{{ t('review.stats.noData') }}</p>
              </div>
            </div>
          </div>
        </section>

        <!-- ========== 2.5. Rating Trend ========== -->
        <section v-if="ratingTrend.length >= 2" class="mb-6">
          <h3 class="text-sm font-semibold uppercase tracking-wider text-text-muted mb-3">
            {{ t('review.detail.trendTitle') }}
          </h3>
          <div class="glass-card rounded-xl shadow-card p-4">
            <div class="flex items-end gap-1 h-[120px]">
              <div
                v-for="(point, idx) in ratingTrend"
                :key="idx"
                class="flex-1 flex flex-col items-center gap-1"
              >
                <span class="text-xs font-mono font-medium text-primary">
                  {{ point.avgRating.toFixed(1) }}
                </span>
                <div
                  class="w-full rounded-t-md bg-primary/20 transition-all duration-base"
                  :style="{ height: `${Math.max(8, (point.avgRating / 5) * 100)}%` }"
                >
                  <div
                    class="w-full h-full rounded-t-md bg-gradient-to-t from-primary/60 to-primary"
                  />
                </div>
                <span class="text-[10px] text-text-muted truncate max-w-full text-center">
                  {{ point.termName }}
                </span>
              </div>
            </div>
          </div>
        </section>

        <!-- ========== 3. Teacher Filter Chips ========== -->
        <section v-if="uniqueTeachers.length > 0" class="mb-6">
          <h3 class="text-sm font-semibold uppercase tracking-wider text-text-muted mb-3">
            {{ t('review.detail.teacherTitle') }} &gt;
          </h3>
          <div class="flex flex-wrap gap-2">
            <button
              v-for="teacher in teacherChips"
              :key="teacher"
              class="rounded-full px-3 py-1 text-[0.8125rem] font-semibold transition-all duration-base"
              :class="selectedTeacher === teacher
                ? 'bg-primary text-white shadow-glow-primary scale-105'
                : 'bg-bg-elevated text-text-secondary hover:scale-[1.02] hover:shadow-sm'"
              @click="selectTeacher(teacher)"
            >
              {{ teacher === '' ? t('review.filter.all') : teacher }}
            </button>
          </div>
        </section>

        <!-- ========== 4. Reviews List ========== -->
        <section>
          <div class="flex flex-col gap-4 mb-4">
            <div
              v-for="r in filteredReviews"
              :key="r.id"
              class="relative bg-bg-card rounded-xl shadow-card p-5 transition-all duration-slow hover:-translate-y-0.5 hover:shadow-lg"
              :class="reviewCardBorderClass(r)"
            >
              <!-- Admin toolbar -->
              <div v-if="canManageReviews" class="absolute top-3 right-3 flex items-center gap-1 z-10">
                <button
                  v-if="r.status !== 'hidden'"
                  class="p-1.5 rounded-lg text-text-muted transition-colors duration-fast hover:text-text-primary hover:bg-bg-elevated"
                  :title="t('review.admin.hide')"
                  @click="openModeration(r)"
                >
                  <EyeOff :size="16" />
                </button>
                <template v-else>
                  <button
                    class="p-1.5 rounded-lg text-text-muted transition-colors duration-fast hover:text-text-primary hover:bg-bg-elevated"
                    :title="t('review.admin.restore')"
                    @click="handleRestore(r)"
                  >
                    <Eye :size="16" />
                  </button>
                  <button
                    class="p-1.5 rounded-lg text-text-muted transition-colors duration-fast hover:text-text-primary hover:bg-bg-elevated"
                    :title="t('review.admin.edit')"
                    @click="openEdit(r)"
                  >
                    <Pencil :size="16" />
                  </button>
                </template>
              </div>

              <!-- Title row -->
              <div class="flex items-center gap-2 mb-1" :class="canManageReviews ? 'pr-20' : ''">
                <span class="font-bold text-base text-text-primary">
                  {{ r.title || course.name }}
                </span>
              </div>

              <!-- Teacher + Semester + Date -->
              <div class="flex items-center gap-1.5 mb-2 text-xs text-text-muted">
                <span v-if="r.teacherName" class="font-medium text-secondary">
                  {{ r.teacherName }}
                </span>
                <span v-if="r.teacherName && r.termName">&middot;</span>
                <span v-if="r.termName">{{ r.termName }}</span>
                <span>&middot;</span>
                <span>{{ formatRelativeTime(r.createdAt, locale, t) }}</span>
              </div>

              <!-- Content -->
              <div
                class="text-sm leading-relaxed break-words whitespace-pre-line mb-2 text-text-secondary"
                v-text="r.content"
              />

              <!-- Grade -->
              <div v-if="r.grade" class="mb-2">
                <span class="inline-flex items-center text-xs font-medium px-2.5 py-0.5 rounded-full bg-success/15 text-success">
                  {{ t('review.post.gradeLabel') }}: {{ r.grade }}
                </span>
              </div>

              <!-- Controversial badge -->
              <ControversialBadge :dislike-count="r.dislikeCount" />

              <!-- Divider -->
              <hr class="border-0 border-t border-border-light my-3" />

              <!-- Bottom: Emoji ratings row -->
              <div class="flex flex-wrap items-center gap-4 mb-3">
                <div
                  v-for="(value, key) in r.ratings"
                  :key="key"
                  class="flex items-center gap-1.5"
                >
                  <EmojiRating :value="value" :show-value="false" size="sm" />
                  <span class="text-xs text-text-muted">
                    {{ dimensionLabel(String(key)) }}
                  </span>
                </div>
              </div>

              <!-- Like / Dislike buttons -->
              <div class="flex items-center gap-2">
                <button
                  :data-testid="`review-like-${r.id}`"
                  class="inline-flex items-center gap-1.5 py-1.5 px-3 rounded-full text-sm text-text-muted transition-colors duration-fast hover:text-vote-up hover:bg-vote-up/10"
                  :class="reviewVotes[r.id] === 'like' ? '!text-vote-up-active !bg-vote-up/12' : ''"
                  @click="handleVote(r, 'like')"
                >
                  <ThumbsUp :size="16" />
                  <span class="text-xs font-mono">{{ displayLikeCount(r) }}</span>
                </button>
                <button
                  :data-testid="`review-dislike-${r.id}`"
                  class="inline-flex items-center gap-1.5 py-1.5 px-3 rounded-full text-sm text-text-muted transition-colors duration-fast hover:text-vote-down hover:bg-vote-down/10"
                  :class="reviewVotes[r.id] === 'dislike' ? '!text-vote-down-active !bg-vote-down/12' : ''"
                  @click="handleVote(r, 'dislike')"
                >
                  <ThumbsDown :size="16" />
                  <span class="text-xs font-mono">{{ displayDislikeCount(r) }}</span>
                </button>
                <button
                  class="inline-flex items-center gap-1.5 py-1.5 px-3 rounded-full text-sm text-text-muted transition-colors duration-fast hover:text-text-primary hover:bg-bg-elevated"
                  @click="toggleExpand(r.id)"
                >
                  <MessageCircle :size="16" />
                  <span class="text-xs font-mono">{{ replyCountMap[r.id] ?? r.replyCount }}</span>
                </button>
              </div>

              <!-- Expanded reply area -->
              <div v-if="expandedReviewID === r.id" class="mt-4 pt-4 border-t border-border-light animate-fade-in">
                <div v-if="repliesLoading" class="flex justify-center py-4">
                  <div class="w-5 h-5 border-2 border-border border-t-primary rounded-full animate-spin" />
                </div>
                <div v-else-if="repliesError" class="text-center text-sm py-4 text-danger">
                  {{ t('review.review.replyLoadFailed') }}
                  <button class="text-sm cursor-pointer underline ml-2 text-primary" @click="loadReplies(r.id)">
                    {{ t('common.actions.retry') }}
                  </button>
                </div>
                <div v-else-if="replies.length > 0" class="mb-2">
                  <ReplyCard
                    v-for="reply in replies"
                    :key="reply.id"
                    :reply="reply"
                    @delete="handleDeleteReply"
                  />
                </div>
                <div v-else class="text-center text-sm py-4 text-text-muted">
                  {{ t('review.review.noReplies') }}
                </div>

                <ReplyForm
                  ref="replyFormRef"
                  :submitting="replySubmitting"
                  @submit="(content: string) => handleReplySubmit(r.id, content)"
                  @cancel="expandedReviewID = null"
                />
              </div>
            </div>
          </div>

          <!-- Empty state -->
          <div
            v-if="!reviewsLoading && reviews.length === 0"
            class="bg-bg-card rounded-xl shadow-card p-8 text-center"
          >
            <p class="text-text-muted">{{ t('review.course.noReviews') }}</p>
            <button
              class="mt-2 py-2 px-6 text-sm font-medium text-white bg-accent rounded-full transition-opacity duration-fast hover:opacity-90"
              @click="goToPostPage"
            >
              {{ t('review.detail.writeFirst') }}
            </button>
          </div>

          <!-- Load more -->
          <div v-if="hasMore" class="flex justify-center py-6">
            <button
              class="py-2 px-6 text-sm font-medium text-white bg-primary rounded-full opacity-90 transition-opacity duration-fast hover:opacity-100 disabled:opacity-50"
              :disabled="reviewsLoading"
              @click="loadMoreReviews"
            >
              {{ reviewsLoading ? t('common.actions.loading') : t('common.actions.loadMore') }}
            </button>
          </div>
        </section>
      </template>

      <!-- Moderation Dialog -->
      <ModerationDialog
        :visible="showModerationDialog"
        :review-i-d="moderatingReviewID"
        @confirm="handleModerate"
        @close="showModerationDialog = false"
      />

      <!-- Admin Edit Dialog -->
      <AdminEditDialog
        :visible="showEditDialog"
        :review="editingReview!"
        @confirm="handleAdminEdit"
        @close="showEditDialog = false"
      />
    </div>
  </CourseThemeProvider>
</template>

<script setup lang="ts">
import { ref, computed, onUnmounted, watch, nextTick, reactive } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  ThumbsUp,
  ThumbsDown,
  MessageCircle,
  EyeOff,
  Eye,
  Pencil,
} from 'lucide-vue-next'

import CourseThemeProvider from '@/modules/course/theme/CourseThemeProvider.vue'
import EmojiRating from '@/components/business/review/EmojiRating.vue'
import ControversialBadge from '@/components/business/review/ControversialBadge.vue'
import FavoriteButton from '@/components/business/review/FavoriteButton.vue'
import ReplyCard from '@/components/business/review/ReplyCard.vue'
import ReplyForm from '@/components/business/review/ReplyForm.vue'
import ModerationDialog from '@/components/business/review/ModerationDialog.vue'
import AdminEditDialog from '@/components/business/review/AdminEditDialog.vue'
import {
  applyOptimisticVote,
  createReviewVoteState,
  getDisplayVoteCount,
  type VoteType,
} from '@/components/business/review/reviewVoteState'

import { api } from '@/api'
import { ADMIN_REVIEWS_MANAGE, hasCapability } from '@stuhelper/shared/constants'
import { useAuthStore } from '@/stores/auth'
import { useToast } from '@/composables/useToast'
import { formatRelativeTime } from '@/utils/date'
import { useReviewPost } from '@/composables/useReviewPost'
import { getRatingColor } from '@/modules/course/theme'

import type { Course, CourseRatingStatsResponse, TeacherStats, TermRatingStats } from '@/types/course'
import type { Review } from '@/types/review'
import { toReviews } from '@/types/review'
import type { Reply } from '@/types/reply'

const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()
const toast = useToast()
const authStore = useAuthStore()
const { lastPostedAt } = useReviewPost()

const courseID = computed(() => Number(route.params.id))

const isPanelMode = computed(() => {
  return route.matched.some(r => r.name === 'review')
})

// ── Auth & Capabilities ──
const canManageReviews = computed(() =>
  hasCapability(authStore.globalCapabilities, ADMIN_REVIEWS_MANAGE),
)

// ── Page state ──
const loading = ref(false)
const error = ref(false)
const contentReady = ref(false)
const course = ref<Course | null>(null)
const ratingStats = ref<CourseRatingStatsResponse | null>(null)
const courseTeachers = ref<TeacherStats[]>([])
const ratingTrend = ref<{ termName: string; avgRating: number }[]>([])
// ── Reviews ──
const reviews = ref<Review[]>([])
const reviewsLoading = ref(false)
const page = ref(1)
const limit = 20
const total = ref(0)
const hasMore = computed(() => reviews.value.length < total.value)

// ── Teacher filter ──
const selectedTeacher = ref('')
const uniqueTeachers = computed(() => {
  return courseTeachers.value
    .map(t => t.teacherName)
    .filter(Boolean)
    .sort()
})
const teacherChips = computed(() => ['', ...uniqueTeachers.value])

const filteredReviews = computed(() => {
  if (!selectedTeacher.value) return reviews.value
  return reviews.value.filter(r => r.teacherName === selectedTeacher.value)
})

function selectTeacher(name: string) {
  selectedTeacher.value = name
}

// ── Voting state (per-review) ──
const reviewVotes = reactive<Record<string, VoteType | null>>({})
const likeOffsets = reactive<Record<string, number>>({})
const dislikeOffsets = reactive<Record<string, number>>({})
const votingReviews = ref(new Set<string>())

function displayLikeCount(r: Review): number {
  return getDisplayVoteCount(r.likeCount, likeOffsets[r.id] ?? 0)
}

function displayDislikeCount(r: Review): number {
  return getDisplayVoteCount(r.dislikeCount, dislikeOffsets[r.id] ?? 0)
}

async function handleVote(r: Review, type: VoteType) {
  if (votingReviews.value.has(r.id)) return
  votingReviews.value.add(r.id)

  const id = r.id
  const previousState = createReviewVoteState({
    userVote: reviewVotes[id] ?? null,
    likeOffset: likeOffsets[id] ?? 0,
    dislikeOffset: dislikeOffsets[id] ?? 0,
  })
  const nextState = applyOptimisticVote(previousState, type)

  reviewVotes[id] = nextState.userVote
  likeOffsets[id] = nextState.likeOffset
  dislikeOffsets[id] = nextState.dislikeOffset

  try {
    await api.review.voteReview(id, { voteType: type })
  } catch {
    reviewVotes[id] = previousState.userVote
    likeOffsets[id] = previousState.likeOffset
    dislikeOffsets[id] = previousState.dislikeOffset
    toast.error(t('review.review.voteFailed'))
  } finally {
    votingReviews.value.delete(id)
  }
}

// ── Reply system ──
const expandedReviewID = ref<string | null>(null)
const replies = ref<Reply[]>([])
const repliesLoading = ref(false)
const repliesError = ref(false)
const replySubmitting = ref(false)
const replyFormRef = ref<InstanceType<typeof ReplyForm> | null>(null)
const replyCountMap = reactive<Record<string, number>>({})
let repliesRequestSeq = 0

async function toggleExpand(reviewId: string) {
  if (expandedReviewID.value === reviewId) {
    expandedReviewID.value = null
    return
  }
  expandedReviewID.value = reviewId
  await loadReplies(reviewId)
}

async function loadReplies(reviewId: string) {
  const requestSeq = ++repliesRequestSeq
  repliesLoading.value = true
  repliesError.value = false
  try {
    const res = await api.reply.getReplies(reviewId)
    if (requestSeq !== repliesRequestSeq || expandedReviewID.value !== reviewId) return
    replies.value = res.data?.data?.list || []
    replyCountMap[reviewId] = res.data?.data?.total || 0
  } catch {
    if (requestSeq !== repliesRequestSeq || expandedReviewID.value !== reviewId) return
    replies.value = []
    repliesError.value = true
  } finally {
    if (requestSeq === repliesRequestSeq) {
      repliesLoading.value = false
    }
  }
}

async function handleReplySubmit(reviewId: string, content: string) {
  replySubmitting.value = true
  try {
    const res = await api.reply.createReply(reviewId, { content })
    if (res.data?.data) {
      replies.value = [...replies.value, res.data.data]
      replyCountMap[reviewId] = (replyCountMap[reviewId] ?? 0) + 1
      replyFormRef.value?.clear()
      toast.success(t('review.review.replySuccess'))
    }
  } catch {
    toast.error(t('review.review.replyFailed'))
  } finally {
    replySubmitting.value = false
  }
}

async function handleDeleteReply(id: string) {
  if (!expandedReviewID.value) return
  try {
    await api.reply.deleteReply(id)
    replies.value = replies.value.filter(r => r.id !== id)
    replyCountMap[expandedReviewID.value] = Math.max(0, (replyCountMap[expandedReviewID.value] ?? 0) - 1)
  } catch {
    toast.error(t('review.review.deleteFailed'))
  }
}

// ── Admin tools ──
const showModerationDialog = ref(false)
const showEditDialog = ref(false)
const moderatingReviewID = ref('')
const editingReview = ref<Review | null>(null)

function openModeration(r: Review) {
  moderatingReviewID.value = r.id
  showModerationDialog.value = true
}

function openEdit(r: Review) {
  editingReview.value = r
  showEditDialog.value = true
}

async function handleModerate(reason: string) {
  showModerationDialog.value = false
  try {
    await api.admin.updateReview(moderatingReviewID.value, { action: 'hide', reason })
    toast.success(t('review.admin.moderateSuccess'))
    refreshReviews()
  } catch {
    toast.error(t('review.admin.actionFailed'))
  }
}

async function handleRestore(r: Review) {
  try {
    await api.admin.updateReview(r.id, { action: 'restore' })
    toast.success(t('review.admin.restoreSuccess'))
    refreshReviews()
  } catch {
    toast.error(t('review.admin.actionFailed'))
  }
}

async function handleAdminEdit(payload: { title: string; content: string; reason: string }) {
  if (!editingReview.value) return
  showEditDialog.value = false
  try {
    await api.admin.editReview(editingReview.value.id, payload)
    toast.success(t('review.admin.editSuccess'))
    refreshReviews()
  } catch {
    toast.error(t('review.admin.actionFailed'))
  }
}

// ── Rating helpers ──
const DIMENSION_LABELS: Record<string, string> = {
  recommendation: 'review.detail.recommend',
  content_quality: 'review.detail.contentQuality',
  workload: 'review.detail.workload',
  assessment: 'review.detail.exam',
}

function dimensionLabel(key: string): string {
  const labelKey = DIMENSION_LABELS[key]
  if (labelKey) return t(labelKey)

  const translated = t(`review.ratingEmoji.${key}`)
  return translated === `review.ratingEmoji.${key}` ? key : translated
}

function ratingBarColor(avg: number): string {
  return getRatingColor(Math.round(avg))
}

function termReviewCount(term: TermRatingStats): number {
  if (!term.dimensions || term.dimensions.length === 0) return 0
  return term.dimensions[0].ratingCount ?? 0
}

// ── Review card styles (low rating left border) ──
function reviewCardBorderClass(r: Review): string {
  const avg = avgRatingForReview(r)
  if (avg <= 1) return 'border-l-4 !border-l-danger shadow-[0_0_12px_var(--color-danger)/40]'
  if (avg <= 2) return 'border-l-4 !border-l-warning shadow-[0_0_12px_var(--color-warning)/40]'
  return ''
}

function avgRatingForReview(r: Review): number {
  const vals = Object.values(r.ratings || {})
  if (vals.length === 0) return 3
  return vals.reduce((a, b) => a + b, 0) / vals.length
}

// ── Navigation ──
function goToPostPage() {
  router.push({ name: 'course-review-post', params: { id: courseID.value } })
}

// ── Data fetching ──
let loadVersion = 0

const fetchReviews = async (append = false, expectedVersion?: number) => {
  reviewsLoading.value = true
  try {
    const res = await api.review.getReviews(courseID.value, {
      page: page.value,
      pageSize: limit,
      sort: 'time',
    })
    if (expectedVersion !== undefined && expectedVersion !== loadVersion) return
    const list = toReviews(res.data?.data?.list || [])
    reviews.value = append ? [...reviews.value, ...list] : list
    total.value = res.data?.data?.total || 0
  } catch (error) {
    if (expectedVersion === undefined || expectedVersion === loadVersion) {
      toast.error(error instanceof Error ? error.message : t('review.course.loadFailed'))
    }
  } finally {
    if (expectedVersion === undefined || expectedVersion === loadVersion) {
      reviewsLoading.value = false
    }
  }
}

const loadMoreReviews = () => {
  if (reviewsLoading.value || !hasMore.value) return
  page.value++
  fetchReviews(true)
}

function refreshReviews() {
  page.value = 1
  const version = ++loadVersion
  fetchReviews(false, version)
  fetchRatingStats()
}

const fetchRatingStats = async () => {
  try {
    const res = await api.rating.getCourseStats(courseID.value)
    ratingStats.value = res.data?.data ?? null
  } catch (error) {
    toast.error(error instanceof Error ? error.message : t('common.loadFailed'))
  }
}

const fetchAll = async () => {
  const id = courseID.value
  if (isNaN(id) || id <= 0) {
    router.replace({ name: 'course-hub' })
    return
  }

  const version = ++loadVersion
  course.value = null
  ratingStats.value = null
  courseTeachers.value = []
  reviews.value = []
  total.value = 0
  page.value = 1
  selectedTeacher.value = ''
  contentReady.value = false
  loading.value = true

  try {
    const [courseRes, statsRes, reviewsRes, teachersRes, trendRes] = await Promise.all([
      api.course.getCourse(id).catch(() => null),
      api.rating.getCourseStats(id).catch(() => null),
      api.review.getReviews(id, { page: 1, pageSize: limit, sort: 'time' }).catch(() => null),
      api.rating.getCourseTeachers(id).catch(() => null),
      api.rating.getRatingTrend(id).catch(() => null),
    ])

    if (version !== loadVersion) return

    course.value = courseRes?.data?.data ?? null
    error.value = !courseRes
    ratingStats.value = statsRes?.data?.data ?? null
    reviews.value = toReviews(reviewsRes?.data?.data?.list || [])
    total.value = reviewsRes?.data?.data?.total || 0
    courseTeachers.value = teachersRes?.data?.data || []
    ratingTrend.value = trendRes?.data?.data?.trend || []
  } finally {
    if (version === loadVersion) {
      loading.value = false
      await nextTick()
      contentReady.value = true
    }
  }
}

// Watch for post events
let lastPostedAtSnapshot = lastPostedAt.value
watch(lastPostedAt, (val) => {
  if (val > lastPostedAtSnapshot) {
    lastPostedAtSnapshot = val
    refreshReviews()
  }
})

onUnmounted(() => {
  ++loadVersion
})

// Main data load
watch(courseID, async (newID, oldID) => {
  if (oldID !== undefined && (newID === oldID || isNaN(newID) || newID <= 0)) return
  await fetchAll()
}, { immediate: true })
</script>
