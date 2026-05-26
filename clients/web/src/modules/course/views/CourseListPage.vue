<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
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

function readDepartmentGroups(payload: unknown): DepartmentGroup[] {
  if (!payload || typeof payload !== 'object') {
    throw new Error('Invalid grouped courses response')
  }

  const { groups } = payload as { groups?: unknown }
  if (!Array.isArray(groups)) {
    throw new Error('Invalid grouped courses response')
  }

  return groups.map((group) => {
    if (!group || typeof group !== 'object') {
      throw new Error('Invalid grouped courses response')
    }

    const { departmentName, courses } = group as {
      departmentName?: unknown
      courses?: unknown
    }
    if (!Array.isArray(courses)) {
      throw new Error('Invalid grouped courses response')
    }

    return {
      name: typeof departmentName === 'string' && departmentName
        ? departmentName
        : t('review.filters.all'),
      courses: courses as Course[],
      expanded: true,
    }
  })
}

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

async function fetchCourses(): Promise<void> {
  loading.value = true
  error.value = null
  try {
    const res = await api.course.getCoursesGrouped()
    departmentGroups.value = readDepartmentGroups(res.data?.data)
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
            :aria-label="t('review.courseList.expandAll')"
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
            :aria-label="t('review.courseList.collapseAll')"
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
          {{ t('common.actions.retry') }}
        </button>
      </div>

      <!-- Empty state -->
      <div v-else-if="departmentGroups.length === 0" class="flex items-center justify-center py-20 text-text-secondary">
        {{ t('review.courseList.noCourses') }}
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
            :aria-expanded="group.expanded"
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
            <router-link
              v-for="course in group.courses"
              :key="course.id"
              :to="`/courses/${course.id}/reviews`"
              class="flex items-center justify-between rounded-lg border border-border-light px-4 py-3 cursor-pointer hover:bg-bg-elevated transition-colors"
            >
              <div class="min-w-0">
                <p class="text-sm font-medium text-text-primary truncate">{{ course.name }}</p>
                <p v-if="course.code" class="text-xs text-text-tertiary mt-0.5">{{ course.code }}</p>
              </div>
              <span class="ml-3 shrink-0 text-xs text-text-secondary">
                {{ t('review.courseList.reviewCount', { count: course.reviewCount }) }}
              </span>
            </router-link>
          </div>
        </div>
      </div>
    </div>
  </CourseThemeProvider>
</template>
