<template>
  <Teleport to="body">
    <Transition name="overlay">
      <div v-if="visible" class="modal-overlay" @click.self="$emit('close')">
        <div class="modal-panel animate-modal-in">
          <div class="modal-header">
            <h2 class="modal-title">{{ t('review.hub.postReview') }}</h2>
            <button class="modal-close" @click="$emit('close')">&times;</button>
          </div>

          <div class="modal-body">
            <!-- 课程搜索 -->
            <div v-if="!selectedCourse" class="course-search">
              <input
                v-model="courseQuery"
                class="search-input"
                :placeholder="t('review.post.searchCourse')"
              />
              <div v-if="courseResults.length > 0" class="course-results">
                <button
                  v-for="c in courseResults"
                  :key="c.id"
                  class="course-result-item"
                  @click="selectCourse(c)"
                >
                  <span class="course-result-name">{{ c.name }}</span>
                  <span class="course-result-dept">{{ c.departmentName }}</span>
                </button>
              </div>
            </div>

            <!-- 已选课程 -->
            <div v-else class="selected-course">
              <span class="selected-name">{{ selectedCourse.name }}</span>
              <button class="change-btn" @click="selectedCourse = null">
                {{ t('common.actions.edit') }}
              </button>
            </div>

            <!-- 表单 -->
            <template v-if="selectedCourse">
              <input
                v-model="title"
                class="form-input"
                :placeholder="t('review.post.title')"
              />

              <RatingGroup v-model="ratings" />

              <textarea
                v-model="content"
                class="form-textarea"
                :placeholder="t('review.post.contentPlaceholder')"
                rows="5"
              />
            </template>
          </div>

          <div v-if="selectedCourse" class="modal-footer">
            <button class="cancel-btn" @click="$emit('close')">
              {{ t('common.actions.cancel') }}
            </button>
            <button
              class="submit-btn"
              :disabled="!canSubmit || submitting"
              @click="handleSubmit"
            >
              {{ submitting ? t('common.actions.loading') : t('review.post.submit') }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, watch, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { searchCourses } from '@/api/course'
import { postReview } from '@/api/review'
import { useToast } from '@/composables/useToast'
import RatingGroup from './RatingGroup.vue'
import type { Course } from '@/types/course'
import type { ReviewRatings } from '@/types/review'

const props = defineProps<{ visible: boolean }>()
const emit = defineEmits<{ close: []; posted: [] }>()

const { t } = useI18n()
const toast = useToast()

const courseQuery = ref('')
const courseResults = ref<Course[]>([])
const selectedCourse = ref<Course | null>(null)
const title = ref('')
const content = ref('')
const ratings = ref<ReviewRatings>({})
const submitting = ref(false)

let searchTimer: ReturnType<typeof setTimeout> | null = null

watch(courseQuery, (val) => {
  if (searchTimer) clearTimeout(searchTimer)
  const q = val.trim()
  if (!q) { courseResults.value = []; return }

  searchTimer = setTimeout(async () => {
    try {
      const res = await searchCourses(q, 8)
      courseResults.value = res.data || []
    } catch { courseResults.value = [] }
  }, 300)
})

onUnmounted(() => {
  if (searchTimer) clearTimeout(searchTimer)
})

// 对话框打开时重置表单
watch(() => props.visible, (val) => {
  if (val) {
    courseQuery.value = ''
    courseResults.value = []
    selectedCourse.value = null
    title.value = ''
    content.value = ''
    ratings.value = {}
    submitting.value = false
  }
})

function selectCourse(course: Course) {
  selectedCourse.value = course
  courseQuery.value = ''
  courseResults.value = []
}

const canSubmit = computed(() => {
  return selectedCourse.value &&
    content.value.trim().length > 0 &&
    Object.keys(ratings.value).length > 0
})

async function handleSubmit() {
  if (!canSubmit.value || !selectedCourse.value) return
  submitting.value = true
  try {
    await postReview({
      courseID: selectedCourse.value.id,
      title: title.value.trim() || undefined,
      content: content.value.trim(),
      ratings: ratings.value
    })
    toast.success(t('review.post.success'))
    emit('posted')
    emit('close')
  } catch {
    toast.error(t('review.post.failed'))
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: var(--bg-overlay);
  z-index: var(--z-modal);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--space-4);
}

.modal-panel {
  width: 100%;
  max-width: 560px;
  max-height: 85vh;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-xl);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-5);
  border-bottom: 1px solid var(--border);
}

.modal-title {
  font-size: var(--text-lg);
  font-weight: var(--weight-bold);
  letter-spacing: var(--tracking-tight);
}

.modal-close {
  font-size: var(--text-2xl);
  color: var(--text-muted);
  line-height: 1;
  cursor: pointer;
}

.modal-close:hover { color: var(--text-primary); }

.modal-body {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-5);
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.search-input,
.form-input {
  width: 100%;
  padding: var(--space-3);
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  color: var(--text-primary);
  font-family: var(--font-sans);
}

.search-input:focus,
.form-input:focus {
  outline: none;
  border-color: var(--brand-primary);
  box-shadow: var(--shadow-glow-sm);
}

.course-results {
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  max-height: 200px;
  overflow-y: auto;
}

.course-result-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding: var(--space-3);
  text-align: left;
  font-size: var(--text-sm);
  color: var(--text-primary);
  cursor: pointer;
  transition: background var(--duration-fast);
}

.course-result-item:hover { background: var(--bg-hover); }

.course-result-dept {
  font-size: var(--text-xs);
  color: var(--text-muted);
}

.selected-course {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-3);
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
}

.selected-name {
  font-weight: var(--weight-semibold);
  font-size: var(--text-sm);
}

.change-btn {
  font-size: var(--text-xs);
  color: var(--brand-primary);
  cursor: pointer;
}

.form-textarea {
  width: 100%;
  padding: var(--space-3);
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  color: var(--text-primary);
  font-family: var(--font-sans);
  resize: vertical;
  min-height: 120px;
}

.form-textarea:focus {
  outline: none;
  border-color: var(--brand-primary);
  box-shadow: var(--shadow-glow-sm);
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-3);
  padding: var(--space-4) var(--space-5);
  border-top: 1px solid var(--border);
}

.cancel-btn {
  padding: var(--space-2) var(--space-4);
  font-size: var(--text-sm);
  color: var(--text-secondary);
  border: 1px solid var(--border);
  border-radius: var(--radius-full);
  cursor: pointer;
}

.submit-btn {
  padding: var(--space-2) var(--space-5);
  font-size: var(--text-sm);
  font-weight: var(--weight-medium);
  color: white;
  background: var(--gradient-brand);
  border-radius: var(--radius-full);
  cursor: pointer;
}

.submit-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.overlay-enter-active { animation: overlayIn var(--duration-base) var(--ease-out); }
.overlay-leave-active { animation: overlayIn var(--duration-fast) var(--ease-out) reverse; }
</style>
