<template>
  <div class="post-review-page">
    <header class="page-header">
      <h1 class="page-title">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M12 5v14M5 12h14"/>
        </svg>
        发布测评
      </h1>
      <p class="page-desc">分享你的课程体验，帮助其他同学选课</p>
    </header>

    <form class="review-form" @submit.prevent="handleSubmit">
      <!-- Course Selection -->
      <div class="form-group">
        <label class="form-label required">选择课程</label>
        <SearchBar @select="handleCourseSelect" />
        <div v-if="selectedCourse" class="selected-course">
          <span class="course-tag">{{ selectedCourse.name }}</span>
          <span v-if="selectedCourse.teacherName" class="teacher-name">
            {{ selectedCourse.teacherName }} 老师
          </span>
        </div>
        <span v-if="errors.courseId" class="error-msg">{{ errors.courseId }}</span>
      </div>

      <!-- Title -->
      <div class="form-group">
        <label class="form-label">测评标题</label>
        <input
          v-model="form.title"
          type="text"
          class="form-input"
          placeholder="给你的测评起个标题（可选）"
          maxlength="50"
        />
        <span class="char-count">{{ form.title.length }}/50</span>
      </div>

      <!-- Ratings -->
      <div class="form-group">
        <label class="form-label required">课程评分</label>
        <RatingGroup ref="ratingGroupRef" v-model="form.ratings" />
        <span v-if="errors.ratings" class="error-msg">{{ errors.ratings }}</span>
      </div>

      <!-- Content -->
      <div class="form-group">
        <label class="form-label required">测评内容</label>
        <textarea
          v-model="form.content"
          class="form-textarea"
          rows="8"
          placeholder="分享你的课程体验，包括课程内容、教学方式、考核方式等..."
          maxlength="2000"
        ></textarea>
        <span class="char-count">{{ form.content.length }}/2000</span>
        <span v-if="errors.content" class="error-msg">{{ errors.content }}</span>
      </div>

      <!-- Actions -->
      <div class="form-actions">
        <button type="submit" class="btn-primary" :disabled="submitting">
          <span v-if="submitting" class="spinner"></span>
          {{ submitting ? '发布中...' : '发布测评' }}
        </button>
        <button type="button" class="btn-secondary" @click="router.back()">
          取消
        </button>
      </div>
    </form>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import SearchBar from '@/components/review/SearchBar.vue'
import RatingGroup from '@/components/review/RatingGroup.vue'
import { postReview } from '@/api/review'
import type { Course } from '@/types/course'
import type { ReviewRatings } from '@/types/review'

const router = useRouter()
const ratingGroupRef = ref<InstanceType<typeof RatingGroup>>()
const submitting = ref(false)
const selectedCourse = ref<Course | null>(null)

const form = ref({
  courseId: 0,
  title: '',
  content: '',
  ratings: {} as ReviewRatings
})

const errors = reactive({
  courseId: '',
  content: '',
  ratings: ''
})

const handleCourseSelect = (course: Course) => {
  selectedCourse.value = course
  form.value.courseId = course.id
  errors.courseId = ''
}

const handleSubmit = async () => {
  // 重置错误
  errors.courseId = ''
  errors.content = ''
  errors.ratings = ''

  // 验证课程
  if (!form.value.courseId) {
    errors.courseId = '请选择课程'
    return
  }

  // 验证内容
  if (!form.value.content) {
    errors.content = '请输入测评内容'
    return
  }
  if (form.value.content.length < 10) {
    errors.content = '测评内容至少10个字符'
    return
  }

  // 验证评分
  const dimensions = ratingGroupRef.value?.dimensions || []
  const ratings = form.value.ratings
  const hasAllRatings = dimensions.every(d => ratings[d.key])
  if (!hasAllRatings) {
    errors.ratings = '请完成所有评分项'
    return
  }

  submitting.value = true
  try {
    await postReview({
      courseId: form.value.courseId,
      title: form.value.title,
      content: form.value.content,
      ratings: form.value.ratings
    })
    router.push(`/review/courses/${form.value.courseId}`)
  } catch (e) {
    console.error('Failed to post review:', e)
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.post-review-page {
  max-width: 700px;
  margin: 0 auto;
  padding: var(--space-6);
}

.page-header {
  margin-bottom: var(--space-8);
}

.page-title {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  font-family: var(--font-display);
  font-size: var(--text-2xl);
  font-weight: 700;
  color: var(--text-primary);
  margin: 0 0 var(--space-2) 0;
}

.page-title svg {
  width: 28px;
  height: 28px;
  color: var(--accent);
}

.page-desc {
  font-size: var(--text-base);
  color: var(--text-muted);
  margin: 0;
}

/* Form */
.review-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  position: relative;
}

.form-label {
  font-size: var(--text-sm);
  font-weight: 500;
  color: var(--text-secondary);
}

.form-label.required::after {
  content: '*';
  color: #ef4444;
  margin-left: var(--space-1);
}

/* Inputs */
.form-input,
.form-textarea {
  width: 100%;
  padding: var(--space-3) var(--space-4);
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  color: var(--text-primary);
  font-size: var(--text-base);
  font-family: inherit;
  transition: all var(--duration-fast);
}

.form-input:focus,
.form-textarea:focus {
  outline: none;
  border-color: var(--accent);
  box-shadow: 0 0 0 3px rgba(201, 162, 39, 0.1);
}

.form-input::placeholder,
.form-textarea::placeholder {
  color: var(--text-muted);
}

.form-textarea {
  resize: vertical;
  min-height: 160px;
  line-height: 1.6;
}

/* Character Count */
.char-count {
  position: absolute;
  right: 0;
  top: 0;
  font-size: var(--text-xs);
  color: var(--text-muted);
}

/* Selected Course */
.selected-course {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin-top: var(--space-2);
}

.course-tag {
  display: inline-flex;
  align-items: center;
  padding: var(--space-1) var(--space-3);
  background: rgba(201, 162, 39, 0.1);
  border: 1px solid var(--border-accent);
  border-radius: var(--radius-full);
  font-size: var(--text-sm);
  color: var(--accent);
}

.teacher-name {
  font-size: var(--text-sm);
  color: var(--text-muted);
}

/* Error Message */
.error-msg {
  font-size: var(--text-sm);
  color: #ef4444;
  margin-top: var(--space-1);
}

/* Form Actions */
.form-actions {
  display: flex;
  gap: var(--space-3);
  margin-top: var(--space-4);
}

.btn-primary,
.btn-secondary {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
  padding: var(--space-3) var(--space-6);
  font-size: var(--text-base);
  font-weight: 600;
  border-radius: var(--radius-md);
  transition: all var(--duration-fast);
  cursor: pointer;
}

.btn-primary {
  background: var(--accent);
  color: var(--bg-primary);
}

.btn-primary:hover:not(:disabled) {
  background: var(--accent-hover);
  transform: translateY(-1px);
}

.btn-primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-secondary {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  color: var(--text-secondary);
}

.btn-secondary:hover {
  border-color: var(--border-light);
  color: var(--text-primary);
}

/* Spinner */
.spinner {
  width: 16px;
  height: 16px;
  border: 2px solid transparent;
  border-top-color: currentColor;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

/* Responsive */
@media (max-width: 640px) {
  .post-review-page {
    padding: var(--space-4);
  }

  .form-actions {
    flex-direction: column;
  }

  .btn-primary,
  .btn-secondary {
    width: 100%;
  }
}
</style>
