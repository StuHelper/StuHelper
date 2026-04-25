<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ChevronDown, ChevronUp } from 'lucide-vue-next'
import CourseThemeProvider from '@/modules/course/theme/CourseThemeProvider.vue'
import SkeletonCard from '@/components/common/SkeletonCard.vue'
import { api } from '@/api'
import type { Course } from '@stuhelper/shared/types/business/course'

interface DepartmentGroup {
  name: string
  courses: Course[]
  expanded: boolean
}

const { t } = useI18n()
const router = useRouter()

const loading = ref(true)
const error = ref<string | null>(null)
const departmentGroups = ref<DepartmentGroup[]>([])

const allExpanded = computed(() =>
  departmentGroups.value.length > 0 &&
  departmentGroups.value.every((g) => g.expanded),
)

const allCollapsed = computed(() =>
  departmentGroups.value.length > 0 &&
  departmentGroups.value.every((g) => !g.expanded),
)

function expandAll(): void {
  departmentGroups.value = departmentGroups.value.map((g) => ({
    ...g,
    expanded: true,
  }))
}

function collapseAll(): void {
  departmentGroups.value = departmentGroups.value.map((g) => ({
    ...g,
    expanded: false,
  }))
}

function toggleDepartment(index: number): void {
  departmentGroups.value = departmentGroups.value.map((g, i) =>
    i === index ? { ...g, expanded: !g.expanded } : g,
  )
}

function navigateToCourse(courseId: number): void {
  void router.push(`/courses/${courseId}/reviews`)
}

async function fetchCourses(): Promise<void> {
  loading.value = true
  error.value = null
  try {
    const res = await api.course.getCoursesGrouped()
    const groups = res.data?.data?.groups ?? []
    departmentGroups.value = groups.map((g) => ({
      name: g.departmentName ?? t('review.filters.all'),
      courses: g.courses ?? [],
      expanded: true,
    }))
  } catch (_error) { void _error;
    error.value = t('review.courseList.loadFailed')
    departmentGroups.value = []
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void fetchCourses()
})
</script>

<template>
  <CourseThemeProvider>
    <div class="min-h-screen w-full bg-bg-base">
      <!-- Top toolbar -->
      <div class="flex items-center justify-between bg-bg-card border-b border-border-light px-6 py-4">
        <h1 class="text-xl font-bold text-text-primary">
          {{ t('review.courseList.title') }}
        </h1>
        <div class="flex gap-2">
          <button
            :disabled="allExpanded"
            :title="t('review.courseList.expandAll')"
            class="flex h-9 w-9 items-center justify-center rounded-lg transition-colors"
            :class="
              allExpanded
                ? 'bg-bg-elevated text-text-tertiary cursor-not-allowed'
                : 'bg-primary text-white hover:bg-primary/90 cursor-pointer'
            "
            @click="expandAll"
          >
            <ChevronDown :size="18" />
          </button>
          <button
            :disabled="allCollapsed"
            :title="t('review.courseList.collapseAll')"
            class="flex h-9 w-9 items-center justify-center rounded-lg transition-colors"
            :class="
              allCollapsed
                ? 'bg-bg-elevated text-text-tertiary cursor-not-allowed'
                : 'bg-primary text-white hover:bg-primary/90 cursor-pointer'
            "
            @click="collapseAll"
          >
            <ChevronUp :size="18" />
          </button>
        </div>
      </div>

      <!-- Loading state -->
      <div
        v-if="loading"
        class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 p-6"
      >
        <SkeletonCard v-for="i in 9" :key="i" />
      </div>

      <!-- Error state -->
      <div v-else-if="error" class="flex flex-col items-center justify-center py-20 text-text-secondary">
        <p class="mb-4">{{ error }}</p>
        <button
          class="rounded-lg bg-primary px-4 py-2 text-white hover:bg-primary/90 cursor-pointer"
          @click="fetchCourses"
        >
          {{ t('common.retry') }}
        </button>
      </div>

      <!-- Empty state -->
      <div v-else-if="departmentGroups.length === 0" class="flex items-center justify-center py-20 text-text-secondary">
        {{ t('review.courseList.empty') }}
      </div>

      <!-- Department groups -->
      <div v-else class="p-6 space-y-4">
        <div
          v-for="(group, groupIndex) in departmentGroups"
          :key="group.name"
          class="rounded-xl bg-bg-card border border-border-light overflow-hidden"
        >
          <button
            class="flex w-full items-center justify-between px-5 py-3 cursor-pointer hover:bg-bg-elevated/50 transition-colors"
            @click="toggleDepartment(groupIndex)"
          >
            <span class="text-base font-semibold text-text-primary">
              {{ group.name }}
              <span class="ml-2 text-sm font-normal text-text-tertiary">({{ group.courses.length }})</span>
            </span>
            <ChevronDown
              :size="16"
              class="text-text-tertiary transition-transform"
              :class="{ 'rotate-180': !group.expanded }"
            />
          </button>

          <div v-show="group.expanded" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3 px-5 pb-4">
            <div
              v-for="course in group.courses"
              :key="course.id"
              class="flex items-center justify-between rounded-lg border border-border-light px-4 py-3 cursor-pointer hover:bg-bg-elevated transition-colors"
              @click="navigateToCourse(course.id)"
            >
              <div class="min-w-0">
                <p class="text-sm font-medium text-text-primary truncate">{{ course.name }}</p>
                <p v-if="course.code" class="text-xs text-text-tertiary mt-0.5">{{ course.code }}</p>
              </div>
              <span class="ml-3 shrink-0 text-xs text-text-secondary">
                {{ course.reviewCount }} {{ t('review.courseList.reviews') }}
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </CourseThemeProvider>
</template>
