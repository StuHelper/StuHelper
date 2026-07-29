<script setup lang="ts">
import { computed, ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import A11yButton from '@/components/A11yButton.vue'
import { api } from '@/api'
import type { components } from '@/api'
import type { ReviewRatings } from '@stuhelper/shared/review'
import { normalizeReviewGrade } from '@stuhelper/shared/constants'
import { assertMutationSuccess, unwrapData, unwrapOptionalData } from '@/api/result'
import { setPageTitle, translate } from '@/i18n'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()
const t = translate
const courseID = ref(0)
const submitting = ref(false)
const savingDraft = ref(false)
const loading = ref(false)
const course = ref<components['schemas']['Course'] | null>(null)
const terms = ref<components['schemas']['Term'][]>([])
const teachers = ref<components['schemas']['CourseTeacherStats'][]>([])
const dimensions = ref<components['schemas']['RatingDimension'][]>([])
type ReviewGrade = NonNullable<components['schemas']['PostReviewRequest']['grade']>
const ratingValues = [1, 2, 3, 4, 5] as const

const form = ref<{
  teacherID: number
  termID: string
  title: string
  content: string
  grade: string
  ratings: ReviewRatings
}>({
  teacherID: 0,
  termID: '',
  title: '',
  content: '',
  grade: '',
  ratings: {} as ReviewRatings,
})

function normalizedGrade(): ReviewGrade | undefined {
  return normalizeReviewGrade(form.value.grade)
}

const canSubmit = computed(() =>
  !!course.value && !!form.value.termID && form.value.content.trim().length >= 20 && Object.keys(form.value.ratings).length === dimensions.value.length
)

async function loadPage() {
  if (!courseID.value) return
  loading.value = true
  try {
    const [courseResult, termsResult, teachersResult, dimensionsResult] = await Promise.all([
      api.course.getCourse(courseID.value),
      api.course.getTerms(),
      api.rating.getCourseTeachers(courseID.value),
      api.rating.getDimensions(),
    ])

    course.value = unwrapOptionalData<components['schemas']['Course']>(courseResult)
    terms.value = unwrapData<components['schemas']['Term'][]>(termsResult)
    teachers.value = unwrapData<components['schemas']['CourseTeacherStats'][]>(teachersResult)
    dimensions.value = unwrapData<components['schemas']['RatingDimension'][]>(dimensionsResult)

    if (terms.value.length > 0 && !form.value.termID) {
      form.value.termID = terms.value[0].id
    }

    const draftResult = await api.draft.getDraft()
    const draft = unwrapOptionalData<components['schemas']['ReviewDraft']>(draftResult)
    if (draft) {
      form.value.teacherID = draft.teacherID || 0
      form.value.termID = draft.termID || form.value.termID
      form.value.title = draft.title || ''
      form.value.content = draft.content || ''
      form.value.grade = draft.grade || ''
      form.value.ratings = (draft.ratings ?? {}) as ReviewRatings
    }
  } catch (error) {
    uni.showToast({
      title: error instanceof Error ? error.message : t('review.post.loadFailed'),
      icon: 'none',
    })
  } finally {
    loading.value = false
  }
}

function updateTerm(event: { detail: { value: number } }) {
  const selected = terms.value[event.detail.value]
  form.value.termID = selected?.id || ''
}

function updateTeacher(event: { detail: { value: number } }) {
  const selected = teachers.value[event.detail.value]
  form.value.teacherID = selected?.teacherID || 0
}

function setRating(key: string, value: 1 | 2 | 3 | 4 | 5) {
  form.value.ratings = { ...form.value.ratings, [key]: value }
}

async function saveDraft() {
  if (!course.value || savingDraft.value) return
  savingDraft.value = true
  try {
    if (!(await authStore.requireAuth(t('review.post.requireAuth')))) return
    assertMutationSuccess(await api.draft.saveDraft({
      courseID: course.value.id,
      teacherID: form.value.teacherID || undefined,
      termID: form.value.termID || undefined,
      title: form.value.title || undefined,
      content: form.value.content || undefined,
      grade: normalizedGrade(),
      ratings: (Object.keys(form.value.ratings).length > 0 ? form.value.ratings : undefined) as ReviewRatings | undefined,
    }))
    uni.showToast({ title: t('review.post.draftSaved'), icon: 'success' })
  } catch (error) {
    uni.showToast({
      title: error instanceof Error ? error.message : t('review.post.draftSaveFailed'),
      icon: 'none',
    })
  } finally {
    savingDraft.value = false
  }
}

async function submitReview() {
  if (!canSubmit.value || !course.value || submitting.value) return
  submitting.value = true
  try {
    if (!(await authStore.requireAuth(t('review.post.requireAuth')))) return
    assertMutationSuccess(await api.review.createReview({
      courseID: course.value.id,
      teacherID: form.value.teacherID || undefined,
      termID: form.value.termID,
      title: form.value.title.trim() || t('review.post.defaultTitle'),
      content: form.value.content.trim(),
      grade: normalizedGrade(),
      ratings: form.value.ratings as ReviewRatings,
    }))
    uni.showToast({ title: t('review.post.submitSuccess'), icon: 'success' })
    try {
      await api.draft.deleteDraft()
    } catch (_error) {
      void _error
      // Draft cleanup is best-effort after a successful review submission.
    }
    setTimeout(() => {
      uni.redirectTo({ url: `/pages/course/detail?id=${course.value?.id}` })
    }, 300)
  } catch (error) {
    uni.showToast({
      title: error instanceof Error ? error.message : t('review.post.submitFailed'),
      icon: 'none',
    })
  } finally {
    submitting.value = false
  }
}

onLoad(async (options) => {
  setPageTitle('common.pageTitles.postReview')
  courseID.value = Number(options?.courseID || 0)
  if (!(await authStore.requireAuth(t('review.post.requireAuth')))) return
  void loadPage()
})
</script>

<template>
  <scroll-view class="post-page" scroll-y>
    <view v-if="loading" class="state-card"><text>{{ t('common.loading') }}</text></view>
    <view v-else-if="!course" class="state-card"><text>{{ t('review.post.missingCourse') }}</text></view>
    <view v-else class="form-card">
      <text class="page-title">{{ t('review.post.title') }}</text>
      <text class="course-name">{{ course.name }}</text>

      <view class="field-group">
        <text class="field-label">{{ t('review.post.term') }}</text>
        <picker :range="terms" range-key="name" @change="updateTerm">
          <view class="picker-field">
            {{ terms.find((item) => item.id === form.termID)?.name || t('review.post.selectTerm') }}
          </view>
        </picker>
      </view>

      <view class="field-group">
        <text class="field-label">{{ t('review.post.teacher') }}</text>
        <picker data-testid="uni-review-teacher-picker" :range="teachers" range-key="teacherName" @change="updateTeacher">
          <view class="picker-field" data-testid="uni-review-teacher-value">
            {{
              teachers.find((item) => item.teacherID === form.teacherID)?.teacherName ||
              t('review.post.selectTeacher')
            }}
          </view>
        </picker>
      </view>

      <view class="field-group">
        <text class="field-label">{{ t('review.post.reviewTitle') }}</text>
        <input
          v-model="form.title"
          class="input-field"
          :aria-label="t('review.post.reviewTitle')"
          data-testid="uni-review-title"
          maxlength="40"
          :placeholder="t('review.post.titlePlaceholder')"
        />
      </view>

      <view class="field-group">
        <text class="field-label">{{ t('review.post.grade') }}</text>
        <input
          v-model="form.grade"
          class="input-field"
          :aria-label="t('review.post.grade')"
          data-testid="uni-review-grade"
          maxlength="20"
          :placeholder="t('review.post.gradePlaceholder')"
        />
      </view>

      <view class="field-group">
        <text class="field-label">{{ t('review.post.rating') }}</text>
        <view v-for="dimension in dimensions" :key="dimension.key" class="rating-row">
          <text class="rating-label">{{ dimension.name }}</text>
          <view class="rating-options">
            <A11yButton
              v-for="value in ratingValues"
              :key="value"
              class="rating-pill"
              :class="{ active: form.ratings[dimension.key] === value }"
              :data-testid="`uni-review-rating-${dimension.key}-${value}`"
              :aria-label="`${dimension.name} ${value}`"
              :aria-pressed="form.ratings[dimension.key] === value"
              @tap="setRating(dimension.key, value)"
            >
              {{ value }}
            </A11yButton>
          </view>
        </view>
      </view>

      <view class="field-group">
        <text class="field-label">{{ t('review.post.content') }}</text>
        <textarea
          v-model="form.content"
          class="textarea-field"
          :aria-label="t('review.post.content')"
          data-testid="uni-review-content"
          maxlength="5000"
          :placeholder="t('review.post.contentPlaceholder')"
        />
        <text class="hint">
          {{ t('review.post.contentHint', { min: 20, count: form.content.trim().length }) }}
        </text>
      </view>

      <A11yButton class="secondary-btn" data-testid="uni-review-save-draft" :disabled="savingDraft" @tap="saveDraft">
        {{ savingDraft ? t('common.processing') : t('review.post.saveDraft') }}
      </A11yButton>
      <A11yButton class="primary-btn" data-testid="uni-review-submit" :disabled="!canSubmit || submitting" @tap="submitReview">
        {{ submitting ? t('review.post.submitting') : t('review.post.submit') }}
      </A11yButton>
    </view>
  </scroll-view>
</template>

<style scoped>
.post-page {
  min-height: 100vh;
  background: #f8fafc;
}

.state-card,
.form-card {
  margin: 24rpx;
  background: #ffffff;
  border-radius: 24rpx;
  box-shadow: 0 10rpx 30rpx rgba(15, 23, 42, 0.05);
}

.state-card {
  padding: 40rpx;
  text-align: center;
  color: #64748b;
}

.form-card {
  padding: 32rpx;
}

.page-title {
  display: block;
  font-size: 36rpx;
  font-weight: 700;
  color: #0f172a;
}

.course-name {
  display: block;
  margin-top: 10rpx;
  font-size: 24rpx;
  color: #64748b;
}

.field-group {
  margin-top: 28rpx;
}

.field-label {
  display: block;
  margin-bottom: 12rpx;
  font-size: 26rpx;
  font-weight: 600;
  color: #334155;
}

.picker-field,
.input-field,
.textarea-field {
  width: 100%;
  box-sizing: border-box;
  border-radius: 22rpx;
  border: 2rpx solid #e2e8f0;
  background: #f8fafc;
  padding: 22rpx 24rpx;
  font-size: 28rpx;
}

.textarea-field {
  min-height: 280rpx;
}

.rating-row {
  margin-bottom: 18rpx;
}

.rating-label {
  display: block;
  margin-bottom: 10rpx;
  font-size: 24rpx;
  color: #475569;
}

.rating-options {
  display: flex;
  gap: 12rpx;
}

.rating-pill {
  flex: 1;
  text-align: center;
  padding: 16rpx 0;
  border-radius: 18rpx;
  background: #eef2ff;
  color: #4338ca;
  font-size: 24rpx;
}

.rating-pill.active {
  background: #4f46e5;
  color: #ffffff;
}

.hint {
  display: block;
  margin-top: 10rpx;
  font-size: 22rpx;
  color: #64748b;
}

.primary-btn,
.secondary-btn {
  margin-top: 20rpx;
  height: 84rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 22rpx;
  font-size: 28rpx;
  font-weight: 600;
}

.primary-btn {
  background: #4f46e5;
  color: #ffffff;
}

.secondary-btn {
  background: #eef2ff;
  color: #4338ca;
}
</style>
