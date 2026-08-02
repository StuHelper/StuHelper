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
      <div v-else-if="error" class="text-center py-12" role="alert">
        <p class="mb-4 text-text-muted">{{ courseLoadErrorMessage }}</p>
        <button
          type="button"
          v-if="canRetryCourseLoad"
          class="py-2 px-6 text-sm font-medium text-white bg-primary rounded-full transition-opacity duration-fast hover:opacity-90"
          data-course-detail-retry-button
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
              type="button"
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
                <div
                  v-for="dim in term.dimensions"
                  :key="dim.key"
                  class="flex items-center gap-2"
                  role="img"
                  :aria-label="dimensionRatingBarAriaLabel(dim)"
                >
                  <span class="text-xs w-16 shrink-0 text-right text-text-secondary">
                    {{ dimensionLabel(dim.key, dim.name) }}
                  </span>
                  <div
                    class="flex-1 h-2 bg-bg-secondary rounded-full overflow-hidden"
                    aria-hidden="true"
                  >
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
        <section v-if="teacherChips.length > 1" class="mb-6">
          <h3 class="text-sm font-semibold uppercase tracking-wider text-text-muted mb-3">
            {{ t('review.detail.teacherTitle') }} &gt;
          </h3>
          <div class="flex flex-wrap gap-2">
            <button
              type="button"
              v-for="teacher in teacherChips"
              :key="teacher.teacherID ?? 'all'"
              class="rounded-full px-3 py-1 text-[0.8125rem] font-semibold transition-all duration-base"
              :aria-pressed="selectedTeacherID === teacher.teacherID"
              :class="selectedTeacherID === teacher.teacherID
                ? 'bg-primary text-white shadow-glow-primary scale-105'
                : 'bg-bg-elevated text-text-secondary hover:scale-[1.02] hover:shadow-sm'"
              @click="selectTeacher(teacher)"
            >
              {{ teacher.teacherName }}
            </button>
          </div>
        </section>

        <!-- ========== 4. Reviews List ========== -->
        <section>
          <div class="flex flex-col gap-4 mb-4">
            <div
              v-for="r in reviews"
              :key="r.id"
              class="relative bg-bg-card rounded-xl shadow-card p-5 transition-all duration-slow hover:-translate-y-0.5 hover:shadow-lg"
              :class="reviewCardBorderClass(r)"
            >
              <!-- Admin toolbar -->
              <div v-if="canManageReviews" class="absolute top-3 right-3 flex items-center gap-1 z-10">
                <button
                  type="button"
                  v-if="r.status !== 'hidden'"
                  class="p-1.5 rounded-lg text-text-muted transition-colors duration-fast hover:text-text-primary hover:bg-bg-elevated"
                  :title="t('review.admin.hide')"
                  @click="openModeration(r)"
                >
                  <EyeOff :size="16" />
                </button>
                <template v-else>
                  <button
                    type="button"
                    class="p-1.5 rounded-lg text-text-muted transition-colors duration-fast hover:text-text-primary hover:bg-bg-elevated"
                    :title="t('review.admin.restore')"
                    @click="handleRestore(r)"
                  >
                    <Eye :size="16" />
                  </button>
                  <button
                    v-if="canEditReviewContent"
                    type="button"
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
                  type="button"
                  :data-testid="`review-like-${r.id}`"
                  class="inline-flex items-center gap-1.5 py-1.5 px-3 rounded-full text-sm text-text-muted transition-colors duration-fast hover:text-vote-up hover:bg-vote-up/10"
                  :class="currentUserVote(r) === 'like' ? '!text-vote-up-active !bg-vote-up/12' : ''"
                  :aria-label="t('review.vote.like')"
                  :aria-pressed="currentUserVote(r) === 'like'"
                  :title="t('review.vote.like')"
                  @click="handleVote(r, 'like')"
                >
                  <ThumbsUp :size="16" />
                  <span class="text-xs font-mono">{{ displayLikeCount(r) }}</span>
                </button>
                <button
                  type="button"
                  :data-testid="`review-dislike-${r.id}`"
                  class="inline-flex items-center gap-1.5 py-1.5 px-3 rounded-full text-sm text-text-muted transition-colors duration-fast hover:text-vote-down hover:bg-vote-down/10"
                  :class="currentUserVote(r) === 'dislike' ? '!text-vote-down-active !bg-vote-down/12' : ''"
                  :aria-label="t('review.vote.dislike')"
                  :aria-pressed="currentUserVote(r) === 'dislike'"
                  :title="t('review.vote.dislike')"
                  @click="handleVote(r, 'dislike')"
                >
                  <ThumbsDown :size="16" />
                  <span class="text-xs font-mono">{{ displayDislikeCount(r) }}</span>
                </button>
                <button
                  type="button"
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
                  <button type="button" class="text-sm cursor-pointer underline ml-2 text-primary" @click="loadReplies(r.id)">
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

                <ReplyLoginPrompt
                  v-if="!isAuthenticated"
                  @login="handleLogin"
                />
                <ReplyForm
                  v-else
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
            v-if="reviewsLoadError"
            role="alert"
            class="bg-bg-card rounded-xl shadow-card p-8 text-center"
          >
            <p class="mb-4 text-danger">{{ t('common.loadFailed') }}</p>
            <button
              type="button"
              class="py-2 px-6 text-sm font-medium text-white bg-primary rounded-full transition-opacity duration-fast hover:opacity-90 disabled:opacity-50"
              :disabled="reviewsLoading"
              @click="retryReviews"
            >
              {{ reviewsLoading ? t('common.actions.loading') : t('common.actions.retry') }}
            </button>
          </div>
          <div
            v-else-if="!reviewsLoading && reviews.length === 0"
            class="bg-bg-card rounded-xl shadow-card p-8 text-center"
          >
            <p class="text-text-muted">{{ t('review.course.noReviews') }}</p>
          </div>

          <!-- Load more -->
          <div v-if="hasMore && !reviewsLoadError" class="flex justify-center py-6">
            <button
              type="button"
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
        :submitting="moderationSubmitting"
        @confirm="handleModerate"
        @close="showModerationDialog = false"
      />

      <!-- Admin Edit Dialog -->
      <AdminEditDialog
        v-if="editingReview"
        :visible="showEditDialog"
        :review="editingReview"
        :submitting="editSubmitting"
        @confirm="handleAdminEdit"
        @close="showEditDialog = false"
      />
    </div>
  </CourseThemeProvider>
</template>

<script setup lang="ts">
import { ref, computed, onUnmounted, watch, nextTick } from 'vue'
import { useRoute, useRouter, type LocationQueryRaw, type LocationQueryValue } from 'vue-router'
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
import ReplyLoginPrompt from '@/components/business/review/ReplyLoginPrompt.vue'
import ModerationDialog from '@/components/business/review/ModerationDialog.vue'
import AdminEditDialog from '@/components/business/review/AdminEditDialog.vue'
import { useReviewAdmin } from '@/modules/review/useReviewAdmin'
import { useReviewReplies } from '@/modules/review/useReviewReplies'
import { useReviewVoting } from '@/modules/review/useReviewVoting'

import { api } from '@/api'
import { getErrorMessage, getErrorStatus, isApiError } from '@/api/errors'
import { updatePageMeta } from '@/composables/usePageMeta'
import { useToast } from '@/composables/useToast'
import { formatRelativeTime } from '@/utils/date'
import { useReviewPost } from '@/composables/useReviewPost'
import { rememberReviewPostCourse } from '@/modules/review/reviewPostNavigation'
import {
  readCoursePayload,
  readTeacherStatsArrayPayload,
} from '@/modules/course/coursePayload'
import { useAuthStore } from '@/stores/auth'
import { useVerificationStore } from '@/stores/verification'
import { canListFullReviews } from '@/utils/adminAccess'
import { accountCenterURLWithRedirect } from '@/utils/redirect'
import {
  ratingBarAriaLabel,
  ratingBarColor,
  ratingDimensionLabel,
  reviewCardBorderClass,
  termReviewCount,
} from '@/modules/review/ratingHelpers'

import type { Course, CourseRatingStatsResponse, TeacherStats } from '@stuhelper/shared/course'
import type { Review } from '@stuhelper/shared/review'
import { isNonArrayRecord as isRecord } from '@stuhelper/shared/utils'

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
const courseLoadErrorMessage = ref('')
const canRetryCourseLoad = ref(false)
const partialLoadError = ref(false)
const contentReady = ref(false)
const course = ref<Course | null>(null)
const ratingStats = ref<CourseRatingStatsResponse | null>(null)
const courseTeachers = ref<TeacherStats[]>([])
const ratingTrend = ref<RatingTrendPoint[]>([])
// ── Reviews ──
const reviews = ref<Review[]>([])
const reviewsLoading = ref(false)
const reviewsLoadError = ref(false)
const page = ref(1)
const limit = 20
const total = ref(0)
const hasMore = computed(() => reviews.value.length < total.value)
const ratingTerms = computed(() => ratingStats.value?.byTerm ?? [])
const hasRatingTerms = computed(() => ratingTerms.value.length > 0)
const showRatingUnavailable = computed(() => !hasRatingTerms.value && total.value > 0)

// ── Teacher filter ──
const selectedTeacherID = ref<number | null>(null)
const sortedTeachers = computed(() => {
  return courseTeachers.value
    .filter(t => t.teacherID > 0 && t.teacherName.trim())
    .slice()
    .sort((a, b) => a.teacherName.localeCompare(b.teacherName))
})
const teacherChips = computed(() => [
  { teacherID: null, teacherName: t('review.filter.all') },
  ...sortedTeachers.value.map(teacher => ({
    teacherID: teacher.teacherID,
    teacherName: teacher.teacherName,
  })),
])

function firstRouteQueryValue(value: LocationQueryValue | LocationQueryValue[] | undefined): string | null {
  if (Array.isArray(value)) {
    return value.find((item): item is string => typeof item === 'string') ?? null
  }
  return typeof value === 'string' ? value : null
}

function readRouteTeacherID(): number | null {
  const raw = firstRouteQueryValue(route.query.teacherID)
  if (!raw || !/^[1-9]\d*$/.test(raw)) return null
  const id = Number(raw)
  return Number.isSafeInteger(id) ? id : null
}

function courseReviewQueryWithTeacher(teacherID: number | null): LocationQueryRaw {
  const nextQuery: LocationQueryRaw = {}
  for (const [key, value] of Object.entries(route.query)) {
    if (key === 'teacherID') continue
    nextQuery[key] = value
  }
  if (teacherID !== null) {
    nextQuery.teacherID = String(teacherID)
  }
  return nextQuery
}

function selectTeacher(teacher: { teacherID: number | null }) {
  if (selectedTeacherID.value === teacher.teacherID) return
  void router.push({
    path: route.path,
    query: courseReviewQueryWithTeacher(teacher.teacherID),
  })
}

// ── Voting state (per-review) ──
const {
  currentUserVote,
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
  canEditReviewContent,
  showModerationDialog,
  showEditDialog,
  editingReview,
  moderationSubmitting,
  editSubmitting,
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

function dimensionRatingBarAriaLabel(
  dimension: CourseRatingStatsResponse['overall']['dimensions'][number],
): string {
  return ratingBarAriaLabel({
    key: dimension.key,
    fallback: dimension.name,
    avgRating: dimension.avgRating,
    t,
  })
}

function readArray(payload: unknown, message: string): unknown[] {
  if (!Array.isArray(payload)) {
    throw new Error(message)
  }
  return payload
}

function readOptionalString(record: Record<string, unknown>, key: string, message: string): string | undefined {
  const value = record[key]
  if (value === undefined) {
    return undefined
  }
  if (typeof value !== 'string') {
    throw new Error(message)
  }
  return value
}

function readString(record: Record<string, unknown>, key: string, message: string): string {
  const value = record[key]
  if (typeof value !== 'string') {
    throw new Error(message)
  }
  return value
}

function readNonNegativeNumber(
  record: Record<string, unknown>,
  key: string,
  message: string,
): number {
  const value = record[key]
  if (typeof value !== 'number' || !Number.isFinite(value) || value < 0) {
    throw new Error(message)
  }
  return value
}

function readPositiveInteger(
  record: Record<string, unknown>,
  key: string,
  message: string,
): number {
  const value = record[key]
  if (typeof value !== 'number' || !Number.isInteger(value) || value <= 0) {
    throw new Error(message)
  }
  return value
}

function readRatingTrendPayload(payload: unknown): RatingTrendPoint[] {
  if (!isRecord(payload)) {
    throw new Error('Invalid rating trend response')
  }

  return readArray(payload.trend, 'Invalid rating trend response').map(item => {
    if (!isRecord(item)) {
      throw new Error('Invalid rating trend response')
    }

    return {
      termName: readString(item, 'termName', 'Invalid rating trend response'),
      avgRating: readNonNegativeNumber(item, 'avgRating', 'Invalid rating trend response'),
    }
  })
}

function readDimensionStats(payload: unknown): CourseRatingStatsResponse['overall']['dimensions'][number] {
  if (!isRecord(payload)) {
    throw new Error('Invalid rating stats response')
  }

  const distributionValue = payload.distribution
  let distribution: Record<string, number> | undefined
  if (distributionValue !== undefined) {
    if (!isRecord(distributionValue)) {
      throw new Error('Invalid rating stats response')
    }
    distribution = {}
    for (const [key, value] of Object.entries(distributionValue)) {
      if (typeof value !== 'number' || !Number.isFinite(value) || value < 0) {
        throw new Error('Invalid rating stats response')
      }
      distribution[key] = value
    }
  }

  return {
    key: readString(payload, 'key', 'Invalid rating stats response'),
    name: readString(payload, 'name', 'Invalid rating stats response'),
    avgRating: readNonNegativeNumber(payload, 'avgRating', 'Invalid rating stats response'),
    ratingCount: readNonNegativeNumber(payload, 'ratingCount', 'Invalid rating stats response'),
    distribution,
  }
}

function readTermRatingStats(payload: unknown): CourseRatingStatsResponse['overall'] {
  if (!isRecord(payload)) {
    throw new Error('Invalid rating stats response')
  }

  return {
    termID: readOptionalString(payload, 'termID', 'Invalid rating stats response'),
    termName: readString(payload, 'termName', 'Invalid rating stats response'),
    dimensions: readArray(payload.dimensions, 'Invalid rating stats response')
      .map(readDimensionStats),
  }
}

function readRatingStatsPayload(payload: unknown): CourseRatingStatsResponse {
  if (!isRecord(payload)) {
    throw new Error('Invalid rating stats response')
  }

  const allDimensionKeys = readArray(payload.allDimensionKeys, 'Invalid rating stats response')
    .map((key) => {
      if (typeof key !== 'string') {
        throw new Error('Invalid rating stats response')
      }
      return key
    })

  return {
    courseID: readPositiveInteger(payload, 'courseID', 'Invalid rating stats response'),
    overall: readTermRatingStats(payload.overall),
    byTerm: readArray(payload.byTerm, 'Invalid rating stats response').map(readTermRatingStats),
    allDimensionKeys,
  }
}

function isCourseNotFoundError(error: unknown) {
  return getErrorStatus(error) === 404 || (isApiError(error) && error.code === 'A0100001')
}

function isValidCourseID(value: number) {
  return Number.isSafeInteger(value) && value > 0
}

function updateCoursePageMeta() {
  if (!course.value) return

  const detailParts = [
    course.value.departmentName,
    total.value > 0
      ? `${total.value} ${t('review.course.reviewUnit')}`
      : undefined,
  ].filter((part): part is string => Boolean(part))

  updatePageMeta({
    title: course.value.name,
    description: detailParts.length > 0
      ? `${course.value.name} · ${detailParts.join(' · ')}`
      : course.value.name,
  })
}

// ── Navigation ──
function handleLogin() {
  void authStore.login()
}

function goVerify() {
  window.location.assign(accountCenterURLWithRedirect('/user/student-verification', route.fullPath))
}

async function goToPostPage() {
  rememberReviewPostCourse(courseID.value)
  const postRoute = router.resolve({ name: 'course-review-post' })
  if (!(await ensureCanPostReview({ redirect: postRoute.fullPath }))) {
    return
  }
  await router.push({ name: 'course-review-post' })
}

// ── Data fetching ──
let loadVersion = 0

function resetCourseDetailData() {
  course.value = null
  ratingStats.value = null
  courseTeachers.value = []
  ratingTrend.value = []
  reviews.value = []
  reviewsLoading.value = false
  reviewsLoadError.value = false
  total.value = 0
  page.value = 1
}

function showCourseNotFound() {
  resetCourseDetailData()
  loading.value = false
  error.value = true
  courseLoadErrorMessage.value = t('review.course.notFound')
  canRetryCourseLoad.value = false
  partialLoadError.value = false
  contentReady.value = true
  updatePageMeta({
    title: t('review.courseDetail'),
    description: t('review.course.notFound'),
  })
}

const fetchReviews = async (append = false, expectedVersion?: number) => {
  reviewsLoading.value = true
  try {
    const pageData = await api.review.getReviewsPage(courseID.value, {
      page: page.value,
      pageSize: limit,
      sort: 'time',
      teacherID: selectedTeacherID.value ?? undefined,
    })
    if (expectedVersion !== undefined && expectedVersion !== loadVersion) return
    reviews.value = append ? [...reviews.value, ...pageData.list] : pageData.list
    total.value = pageData.total
    reviewsLoadError.value = false
  } catch (error) {
    if (expectedVersion === undefined || expectedVersion === loadVersion) {
      if (!append) {
        reviews.value = []
        total.value = 0
      } else {
        page.value = Math.max(1, page.value - 1)
      }
      reviewsLoadError.value = true
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

function retryReviews() {
  page.value = 1
  const version = ++loadVersion
  void fetchReviews(false, version)
}

const fetchRatingStats = async () => {
  try {
    const res = await api.rating.getCourseStats(courseID.value)
    ratingStats.value = readRatingStatsPayload(res.data?.data)
  } catch (error) {
    ratingStats.value = null
    toast.error(getErrorMessage(error, t('common.loadFailed')))
  }
}

const fetchAll = async () => {
  const id = courseID.value
  const version = ++loadVersion
  resetCourseDetailData()
  selectedTeacherID.value = readRouteTeacherID()
  contentReady.value = false
  loading.value = true
  error.value = false
  courseLoadErrorMessage.value = ''
  canRetryCourseLoad.value = false
  partialLoadError.value = false

  if (!isValidCourseID(id)) {
    showCourseNotFound()
    return
  }

  try {
    try {
      const courseRes = await api.course.getCourse(id)
      if (version !== loadVersion) return
      course.value = readCoursePayload(courseRes.data?.data, 'Invalid course response')
    } catch (courseError) {
      if (version !== loadVersion) return
      if (isCourseNotFoundError(courseError)) {
        showCourseNotFound()
      } else {
        course.value = null
        error.value = true
        courseLoadErrorMessage.value = t('common.loadFailed')
        canRetryCourseLoad.value = true
        toast.error(getErrorMessage(courseError, t('common.loadFailed')))
      }
      return
    }

    const [statsRes, reviewsRes, teachersRes, trendRes] = await Promise.allSettled([
      api.rating.getCourseStats(id),
      api.review.getReviewsPage(id, {
        page: 1,
        pageSize: limit,
        sort: 'time',
        teacherID: selectedTeacherID.value ?? undefined,
      }),
      api.rating.getCourseTeachers(id),
      api.rating.getRatingTrend(id),
    ])

    if (version !== loadVersion) return

    let hasPartialError = [statsRes, reviewsRes].some(item => item.status === 'rejected')

    if (statsRes.status === 'fulfilled') {
      try {
        ratingStats.value = readRatingStatsPayload(statsRes.value.data?.data)
      } catch (_error) { void _error;
        ratingStats.value = null
        hasPartialError = true
      }
    } else {
      ratingStats.value = null
      hasPartialError = true
    }

    if (reviewsRes.status === 'fulfilled') {
      reviews.value = reviewsRes.value.list
      total.value = reviewsRes.value.total
      reviewsLoadError.value = false
    } else {
      reviews.value = []
      total.value = 0
      reviewsLoadError.value = true
    }

    if (teachersRes.status === 'fulfilled') {
      try {
        courseTeachers.value = readTeacherStatsArrayPayload(
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
    updateCoursePageMeta()
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
  if (oldID !== undefined && newID === oldID) return
  await fetchAll()
}, { immediate: true })

watch(
  () => route.query.teacherID,
  () => {
    const nextTeacherID = readRouteTeacherID()
    if (selectedTeacherID.value === nextTeacherID) return
    selectedTeacherID.value = nextTeacherID
    page.value = 1
    const version = ++loadVersion
    void fetchReviews(false, version)
  },
)
</script>
