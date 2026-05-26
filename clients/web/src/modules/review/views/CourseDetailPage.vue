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
        <div
          v-if="partialLoadError"
          role="alert"
          class="mb-4 rounded-xl border border-warning/30 bg-warning/10 px-4 py-3 text-sm text-warning"
        >
          {{ t('common.loadFailed') }}
        </div>
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
        <section v-if="hasRatingTerms || showRatingUnavailable" class="mb-6">
          <h3 class="text-sm font-semibold uppercase tracking-wider text-text-muted mb-4">
            {{ t('review.detail.ratingTitle') }} &gt;
          </h3>

          <!-- No data at all -->
          <div
            v-if="showRatingUnavailable"
            class="bg-bg-card rounded-xl shadow-card p-8 text-center"
          >
            <p class="text-text-muted">{{ t('review.stats.noData') }}</p>
          </div>

          <!-- Semester rating cards grid -->
          <div
            v-else
            class="grid gap-4"
            style="grid-template-columns: repeat(auto-fill, minmax(260px, 1fr))"
          >
            <div
              v-for="(term, idx) in ratingTerms"
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
                    {{ dimensionLabel(dim.key, dim.name) }}
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
                  <span
                    class="h-2.5 w-2.5 rounded-full shrink-0"
                    :style="{ backgroundColor: ratingBarColor(dim.avgRating) }"
                    aria-hidden="true"
                  />
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
                v-if="r.status === 'hidden' && !canManageReviews"
                class="mb-2 flex items-start gap-2 rounded-lg border border-warning/20 bg-warning/5 px-3 py-4"
              >
                <ShieldAlert :size="18" class="mt-0.5 shrink-0 text-warning" />
                <div>
                  <span class="text-sm font-medium text-text-secondary">{{ t('review.card.contentHidden') }}</span>
                  <p v-if="r.moderationReason" class="mt-1 text-xs text-text-muted" v-text="r.moderationReason" />
                </div>
              </div>
              <div
                v-else-if="!isAuthenticated"
                class="mb-2"
              >
                <LockedReviewContent
                  :preview-line="r.content"
                  :message="t('review.card.loginToView')"
                  :action-label="t('review.card.loginBtn')"
                  @action="handleLogin"
                />
              </div>
              <div
                v-else-if="isReviewContentLocked"
                class="mb-2"
              >
                <LockedReviewContent
                  :preview-line="r.content"
                  :message="t('review.card.verifyToView')"
                  :action-label="t('review.card.goVerify')"
                  @action="goVerify"
                />
              </div>
              <div
                v-else
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
                  <EmojiRating :value="value" size="sm" />
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
                  :aria-label="t('review.vote.like')"
                  :aria-pressed="reviewVotes[r.id] === 'like'"
                  :title="t('review.vote.like')"
                  @click="handleVote(r, 'like')"
                >
                  <ThumbsUp :size="16" />
                  <span class="text-xs font-mono">{{ displayLikeCount(r) }}</span>
                </button>
                <button
                  :data-testid="`review-dislike-${r.id}`"
                  class="inline-flex items-center gap-1.5 py-1.5 px-3 rounded-full text-sm text-text-muted transition-colors duration-fast hover:text-vote-down hover:bg-vote-down/10"
                  :class="reviewVotes[r.id] === 'dislike' ? '!text-vote-down-active !bg-vote-down/12' : ''"
                  :aria-label="t('review.vote.dislike')"
                  :aria-pressed="reviewVotes[r.id] === 'dislike'"
                  :title="t('review.vote.dislike')"
                  @click="handleVote(r, 'dislike')"
                >
                  <ThumbsDown :size="16" />
                  <span class="text-xs font-mono">{{ displayDislikeCount(r) }}</span>
                </button>
                <button
                  class="inline-flex items-center gap-1.5 py-1.5 px-3 rounded-full text-sm text-text-muted transition-colors duration-fast hover:text-text-primary hover:bg-bg-elevated"
                  :aria-expanded="expandedReviewID === r.id"
                  :aria-label="t('review.review.commentBtn')"
                  :title="t('review.review.commentBtn')"
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
        v-if="editingReview"
        :visible="showEditDialog"
        :review="editingReview"
        @confirm="handleAdminEdit"
        @close="showEditDialog = false"
      />
    </div>
  </CourseThemeProvider>
</template>

<script setup lang="ts">
import { ref, computed, onUnmounted, watch, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  ThumbsUp,
  ThumbsDown,
  MessageCircle,
  EyeOff,
  Eye,
  Pencil,
  ShieldAlert,
} from 'lucide-vue-next'

import CourseThemeProvider from '@/modules/course/theme/CourseThemeProvider.vue'
import EmojiRating from '@/components/business/review/EmojiRating.vue'
import ControversialBadge from '@/components/business/review/ControversialBadge.vue'
import FavoriteButton from '@/components/business/review/FavoriteButton.vue'
import LockedReviewContent from '@/components/business/review/LockedReviewContent.vue'
import ReplyCard from '@/components/business/review/ReplyCard.vue'
import ReplyForm from '@/components/business/review/ReplyForm.vue'
import ModerationDialog from '@/components/business/review/ModerationDialog.vue'
import AdminEditDialog from '@/components/business/review/AdminEditDialog.vue'
import { useReviewAdmin } from '@/modules/review/useReviewAdmin'
import { useReviewReplies } from '@/modules/review/useReviewReplies'
import { useReviewVoting } from '@/modules/review/useReviewVoting'

import { api } from '@/api'
import { getErrorMessage } from '@/api/errors'
import { readArrayPayload } from '@/api/responsePayload'
import { useToast } from '@/composables/useToast'
import { formatRelativeTime } from '@/utils/date'
import { useReviewPost } from '@/composables/useReviewPost'
import { rememberReviewPostCourse } from '@/modules/review/reviewPostNavigation'
import { useAuthStore } from '@/stores/auth'
import { useVerificationStore } from '@/stores/verification'
import { canListFullReviews } from '@/utils/adminAccess'
import {
  ratingBarColor,
  ratingDimensionLabel,
  reviewCardBorderClass,
  termReviewCount,
} from '@/modules/review/ratingHelpers'

import type { Course, CourseRatingStatsResponse, TeacherStats } from '@stuhelper/shared/course'
import type { Review } from '@stuhelper/shared/review'

type RatingTrendPoint = {
  termName: string
  avgRating: number
}

const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()
const toast = useToast()
const { ensureCanPostReview } = useReviewPost()
const authStore = useAuthStore()
const verificationStore = useVerificationStore()

const courseID = computed(() => Number(route.params.id))

const isPanelMode = computed(() => {
  return route.matched.some(r => r.name === 'review')
})

// ── Page state ──
const loading = ref(false)
const error = ref(false)
const partialLoadError = ref(false)
const contentReady = ref(false)
const course = ref<Course | null>(null)
const ratingStats = ref<CourseRatingStatsResponse | null>(null)
const courseTeachers = ref<TeacherStats[]>([])
const ratingTrend = ref<RatingTrendPoint[]>([])
// ── Reviews ──
const reviews = ref<Review[]>([])
const reviewsLoading = ref(false)
const page = ref(1)
const limit = 20
const total = ref(0)
const hasMore = computed(() => reviews.value.length < total.value)
const ratingTerms = computed(() => ratingStats.value?.byTerm ?? [])
const hasRatingTerms = computed(() => ratingTerms.value.length > 0)
const showRatingUnavailable = computed(() => !hasRatingTerms.value && total.value > 0)

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
const {
  reviewVotes,
  displayLikeCount,
  displayDislikeCount,
  handleVote,
} = useReviewVoting()

// ── Reply system ──
const {
  expandedReviewID,
  replies,
  repliesLoading,
  repliesError,
  replySubmitting,
  replyFormRef,
  replyCountMap,
  toggleExpand,
  loadReplies,
  handleReplySubmit,
  handleDeleteReply,
} = useReviewReplies()

// ── Admin tools ──
const {
  canManageReviews,
  showModerationDialog,
  showEditDialog,
  moderatingReviewID,
  editingReview,
  openModeration,
  openEdit,
  handleModerate,
  handleRestore,
  handleAdminEdit,
} = useReviewAdmin(() => {
  refreshReviews()
})

const isAuthenticated = computed(() => authStore.isAuthenticated)
const hasFullListCapability = computed(() => canListFullReviews(authStore.user))
const isReviewContentLocked = computed(() =>
  isAuthenticated.value &&
  !canManageReviews.value &&
  !(hasFullListCapability.value && verificationStore.canViewFullReviews)
)

function dimensionLabel(key: string, fallback?: string): string {
  return ratingDimensionLabel({ key, fallback, t })
}

function readCoursePayload(payload: unknown): Course {
  if (!payload || typeof payload !== 'object') {
    throw new Error('Invalid course response')
  }

  const { id } = payload as { id?: unknown }
  if (typeof id !== 'number' || !Number.isFinite(id) || id <= 0) {
    throw new Error('Invalid course response')
  }

  return payload as Course
}

function readRatingTrendPayload(payload: unknown): RatingTrendPoint[] {
  if (!payload || typeof payload !== 'object') {
    throw new Error('Invalid rating trend response')
  }

  return readArrayPayload<RatingTrendPoint>(
    (payload as { trend?: unknown }).trend,
    'Invalid rating trend response',
  )
}

// ── Navigation ──
function handleLogin() {
  void authStore.login()
}

function goVerify() {
  void router.push({ name: 'student-verification' })
}

async function goToPostPage() {
  if (!(await ensureCanPostReview())) {
    return
  }
  rememberReviewPostCourse(courseID.value)
  await router.push({ name: 'course-review-post' })
}

// ── Data fetching ──
let loadVersion = 0

const fetchReviews = async (append = false, expectedVersion?: number) => {
  reviewsLoading.value = true
  try {
    const pageData = await api.review.getReviewsPage(courseID.value, {
      page: page.value,
      pageSize: limit,
      sort: 'time',
    })
    if (expectedVersion !== undefined && expectedVersion !== loadVersion) return
    reviews.value = append ? [...reviews.value, ...pageData.list] : pageData.list
    total.value = pageData.total
  } catch (error) {
    if (expectedVersion === undefined || expectedVersion === loadVersion) {
      toast.error(getErrorMessage(error, t('common.loadFailed')))
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
    toast.error(getErrorMessage(error, t('common.loadFailed')))
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
  error.value = false
  partialLoadError.value = false

  try {
    const [courseRes, statsRes, reviewsRes, teachersRes, trendRes] = await Promise.allSettled([
      api.course.getCourse(id),
      api.rating.getCourseStats(id),
      api.review.getReviewsPage(id, { page: 1, pageSize: limit, sort: 'time' }),
      api.rating.getCourseTeachers(id),
      api.rating.getRatingTrend(id),
    ])

    if (version !== loadVersion) return

    if (courseRes.status === 'fulfilled') {
      try {
        course.value = readCoursePayload(courseRes.value.data?.data)
      } catch (err) {
        course.value = null
        error.value = true
        toast.error(getErrorMessage(err, t('common.loadFailed')))
        return
      }
    } else {
      course.value = null
      error.value = true
      toast.error(getErrorMessage(courseRes.reason, t('common.loadFailed')))
      return
    }

    let hasPartialError = [statsRes, reviewsRes].some(item => item.status === 'rejected')

    if (statsRes.status === 'fulfilled') {
      ratingStats.value = statsRes.value.data?.data ?? null
    } else {
      ratingStats.value = null
    }

    if (reviewsRes.status === 'fulfilled') {
      reviews.value = reviewsRes.value.list
      total.value = reviewsRes.value.total
    } else {
      reviews.value = []
      total.value = 0
    }

    if (teachersRes.status === 'fulfilled') {
      try {
        courseTeachers.value = readArrayPayload<TeacherStats>(
          teachersRes.value.data?.data,
          'Invalid course teachers response',
        )
      } catch (_error) { void _error;
        courseTeachers.value = []
        hasPartialError = true
      }
    } else {
      courseTeachers.value = []
      hasPartialError = true
    }

    if (trendRes.status === 'fulfilled') {
      try {
        ratingTrend.value = readRatingTrendPayload(trendRes.value.data?.data)
      } catch (_error) { void _error;
        ratingTrend.value = []
        hasPartialError = true
      }
    } else {
      ratingTrend.value = []
      hasPartialError = true
    }

    partialLoadError.value = hasPartialError
    if (partialLoadError.value) {
      toast.error(t('common.loadFailed'))
    }
  } finally {
    if (version === loadVersion) {
      loading.value = false
      await nextTick()
      contentReady.value = true
    }
  }
}

onUnmounted(() => {
  ++loadVersion
})

// 课程切换后重新加载主数据
watch(courseID, async (newID, oldID) => {
  if (oldID !== undefined && (newID === oldID || isNaN(newID) || newID <= 0)) return
  await fetchAll()
}, { immediate: true })
</script>
