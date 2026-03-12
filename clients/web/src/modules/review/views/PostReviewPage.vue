<template>
  <div class="max-w-[760px] mx-auto px-6 py-8 animate-fade-in max-sm:px-4 max-sm:py-6">
    <button
      class="mb-4 px-0 py-1 bg-transparent border-none text-sm text-text-muted cursor-pointer transition-colors duration-fast hover:text-text-primary"
      @click="router.push({ name: 'course-reviews', params: { id: courseID } })"
    >
      {{ t('common.actions.back') }}
    </button>

    <div v-if="loading" class="space-y-4" role="status" aria-busy="true">
      <div class="h-24 rounded-xl bg-bg-secondary shimmer-box" />
      <div class="h-80 rounded-xl bg-bg-secondary shimmer-box" />
    </div>

    <div v-else-if="course" class="space-y-6">
      <header class="bg-bg-card border border-border rounded-2xl p-6 shadow-card">
        <p class="text-xs uppercase tracking-[0.18em] text-text-muted m-0 mb-3">
          {{ t('review.post.submit') }}
        </p>
        <h1 class="font-sans text-2xl font-extrabold tracking-tight text-text-primary m-0 mb-2">
          {{ course.name }}
        </h1>
        <div class="flex flex-wrap items-center gap-2 text-xs text-text-secondary">
          <span v-if="course.departmentName" class="px-2.5 py-1 rounded-full bg-bg-secondary">{{ course.departmentName }}</span>
          <span v-if="course.code" class="px-2.5 py-1 rounded-full bg-bg-secondary font-mono">{{ course.code }}</span>
          <span v-if="course.credits" class="px-2.5 py-1 rounded-full bg-bg-secondary">
            {{ t('review.course.credits', { n: course.credits }) }}
          </span>
        </div>
      </header>

      <ReviewForm :course-i-d="courseID" @posted="handlePosted" />
    </div>

    <div v-else class="bg-bg-card border border-border rounded-2xl p-10 text-center text-text-muted">
      <p class="m-0 mb-4">{{ t('common.loadFailed') }}</p>
      <button
        class="px-4 py-2 bg-primary text-white rounded-lg border-none cursor-pointer text-sm"
        @click="fetchCourseDetail"
      >
        {{ t('common.actions.retry') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import ReviewForm from '@/components/business/review/ReviewForm.vue'
import { api } from '@/api'
import type { Course } from '@/types/course'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const courseID = computed(() => Number(route.params.id))
const loading = ref(false)
const course = ref<Course | null>(null)

async function fetchCourseDetail() {
  if (Number.isNaN(courseID.value) || courseID.value <= 0) {
    router.replace({ name: 'course-list' })
    return
  }

  loading.value = true
  try {
    const res = await api.course.getCourse(courseID.value)
    course.value = res.data?.data ?? null
  } catch (error) {
    console.error('Failed to fetch course:', error)
    course.value = null
  } finally {
    loading.value = false
  }
}

function handlePosted() {
  router.push({ name: 'course-reviews', params: { id: courseID.value } })
}

onMounted(() => {
  fetchCourseDetail()
})
</script>

<style scoped>
.shimmer-box {
  position: relative;
  overflow: hidden;
}

.shimmer-box::after {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(90deg, transparent 25%, var(--color-bg-hover) 50%, transparent 75%);
  transform: translateX(-100%);
  animation: shimmer-slide 1.5s infinite;
}

@keyframes shimmer-slide {
  to { transform: translateX(100%); }
}
</style>
