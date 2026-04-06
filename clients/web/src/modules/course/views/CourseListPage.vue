<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ChevronDown, ChevronUp } from 'lucide-vue-next'
import CourseThemeProvider from '@/modules/course/theme/CourseThemeProvider.vue'
import SkeletonCard from '@/components/common/SkeletonCard.vue'
import { api } from '@/api'
import type { components } from '@stuhelper/shared/types'

type Course = components['schemas']['Course']

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

function groupByDepartment(courses: readonly Course[]): DepartmentGroup[] {
  const map = new Map<string, Course[]>()

  for (const course of courses) {
    const dept = course.departmentName ?? t('review.filters.all')
    const existing = map.get(dept)
    if (existing) {
      existing.push(course)
    } else {
      map.set(dept, [course])
    }
  }

  const groups: DepartmentGroup[] = []
  for (const [name, items] of map) {
    groups.push({ name, courses: items, expanded: true })
  }

  groups.sort((a, b) => a.name.localeCompare(b.name, 'zh-CN'))

  return groups
}

async function fetchCourses(): Promise<void> {
  loading.value = true
  error.value = null
  try {
    const pageSize = 100
    const firstPageResponse = await api.course.getCourses({
      page: 1,
      pageSize,
      sort: 'name',
    })
    const firstPageData = firstPageResponse.data?.data
    const firstPageItems: Course[] = firstPageData?.list ?? []
    const total = firstPageData?.total ?? firstPageItems.length
    const totalPages = Math.max(1, Math.ceil(total / pageSize))

    const remainingResponses = totalPages > 1
      ? await Promise.all(
          Array.from({ length: totalPages - 1 }, (_, index) =>
            api.course.getCourses({
              page: index + 2,
              pageSize,
              sort: 'name',
            }),
          ),
        )
      : []

    const courses: Course[] = [
      ...firstPageItems,
      ...remainingResponses.flatMap((response) => response.data?.data?.list ?? []),
    ]

    departmentGroups.value = groupByDepartment(courses)
  } catch {
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
        class="flex flex-col gap-3 px-4 py-4"
        role="status"
        aria-busy="true"
      >
        <SkeletonCard v-for="i in 5" :key="i" variant="course" />
      </div>

      <!-- Error state -->
      <div
        v-else-if="error"
        class="px-6 py-16 text-center text-destructive"
      >
        <p class="text-lg font-medium">{{ error }}</p>
        <button
          class="mt-4 rounded-lg bg-primary px-6 py-2 text-white transition-colors hover:bg-primary/90"
          @click="fetchCourses"
        >
          {{ t('review.hub.retry') }}
        </button>
      </div>

      <!-- Empty state -->
      <div
        v-else-if="departmentGroups.length === 0"
        class="px-6 py-16 text-center text-text-secondary"
      >
        <p class="text-lg">{{ t('review.courseList.noCourses') }}</p>
      </div>

      <!-- Department list -->
      <div v-else class="space-y-3 px-4 py-4">
        <div
          v-for="(group, index) in departmentGroups"
          :key="group.name"
          class="bg-bg-card rounded-xl shadow-card overflow-hidden stagger-item"
          :style="{ animationDelay: `${Math.min(index, 10) * 60}ms` }"
        >
          <!-- Department header -->
          <div
            class="flex items-center justify-between bg-bg-elevated px-4 py-3 cursor-pointer select-none transition-colors hover:bg-bg-hover"
            role="button"
            tabindex="0"
            @click="toggleDepartment(index)"
            @keydown.enter="toggleDepartment(index)"
            @keydown.space.prevent="toggleDepartment(index)"
          >
            <div class="flex items-center gap-3">
              <span class="text-base font-semibold text-text-primary">
                {{ group.name }}
              </span>
              <span class="text-xs text-text-tertiary">
                {{ t('review.courseList.courseCount', { count: group.courses.length }) }}
              </span>
            </div>
            <component
              :is="group.expanded ? ChevronUp : ChevronDown"
              :size="18"
              class="text-text-secondary transition-colors"
            />
          </div>

          <!-- Course items -->
          <div v-if="group.expanded" class="px-2 pb-2 pt-1">
            <div
              v-for="course in group.courses"
              :key="course.id"
              class="flex items-center justify-between rounded-lg bg-bg-elevated/50 px-4 py-2.5 my-1 cursor-pointer transition-all hover:bg-bg-hover hover:translate-x-0.5"
              role="button"
              tabindex="0"
              @click="navigateToCourse(course.id)"
              @keydown.enter="navigateToCourse(course.id)"
              @keydown.space.prevent="navigateToCourse(course.id)"
            >
              <span class="text-sm text-text-primary">
                {{ course.name }}
              </span>
              <span class="whitespace-nowrap rounded-full bg-primary/10 px-2 py-0.5 text-xs text-primary">
                {{ t('review.courseList.reviewCount', { count: course.reviewCount }) }}
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </CourseThemeProvider>
</template>
