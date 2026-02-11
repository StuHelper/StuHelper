<template>
  <aside class="dept-sidebar">
    <div class="dept-sidebar__header">
      <h3 class="dept-sidebar__title">{{ t('nav.courses') }}</h3>
    </div>

    <!-- 院系列表 -->
    <div class="dept-list">
      <button
        v-for="dept in departments"
        :key="dept.id"
        class="dept-item"
        :class="{ active: activeDeptID === dept.id }"
        @click="selectDepartment(dept.id)"
      >
        {{ dept.name }}
      </button>
    </div>

    <!-- 课程列表 -->
    <div v-if="activeDeptID" class="course-section">
      <div v-if="coursesLoading" class="courses-loading">
        <div class="spinner" />
      </div>
      <template v-else-if="courses.length > 0">
        <CourseListItem
          v-for="course in courses"
          :key="course.id"
          :course="course"
        />
      </template>
      <div v-else class="courses-empty">
        {{ t('common.empty.result') }}
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { getDepartments, getCourses } from '@/api/course'
import type { Department, Course } from '@/types/course'
import CourseListItem from './CourseListItem.vue'

const { t } = useI18n()

const departments = ref<Department[]>([])
const activeDeptID = ref<number | null>(null)
const courses = ref<Course[]>([])
const coursesLoading = ref(false)
// 缓存已加载的院系课程列表，避免重复请求
const coursesCache = new Map<number, Course[]>()

async function loadDepartments() {
  try {
    const res = await getDepartments()
    departments.value = res.data || []
    if (departments.value.length > 0) {
      selectDepartment(departments.value[0].id)
    }
  } catch {
    departments.value = []
  }
}

async function selectDepartment(id: number) {
  activeDeptID.value = id

  // 命中缓存则直接使用
  const cached = coursesCache.get(id)
  if (cached) {
    courses.value = cached
    return
  }

  coursesLoading.value = true
  try {
    const res = await getCourses(id)
    const list = res.data || []
    coursesCache.set(id, list)
    courses.value = list
  } catch {
    courses.value = []
  } finally {
    coursesLoading.value = false
  }
}

onMounted(loadDepartments)
</script>

<style scoped>
.dept-sidebar {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  overflow-y: auto;
  max-height: calc(100vh - 120px);
}

.dept-sidebar__header {
  padding: var(--space-2) var(--space-3);
}

.dept-sidebar__title {
  font-size: var(--text-xs);
  font-weight: var(--weight-semibold);
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: var(--tracking-wide);
  margin: 0;
}

.dept-list {
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.dept-item {
  display: block;
  width: 100%;
  text-align: left;
  padding: var(--space-2) var(--space-3);
  font-size: var(--text-sm);
  color: var(--text-secondary);
  border-left: 2px solid transparent;
  border-radius: 0;
  cursor: pointer;
  transition: all var(--duration-fast) var(--ease-smooth);
  background: none;
}

.dept-item:hover {
  color: var(--text-primary);
  background: var(--bg-hover);
}

.dept-item.active {
  color: var(--brand-primary);
  border-left-color: var(--brand-primary);
  background: color-mix(in srgb, var(--brand-primary) 6%, transparent);
  font-weight: var(--weight-medium);
}

.course-section {
  margin-top: var(--space-2);
  padding-top: var(--space-2);
  border-top: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.courses-loading {
  display: flex;
  justify-content: center;
  padding: var(--space-4);
}

.spinner {
  width: 18px;
  height: 18px;
  border: 2px solid var(--border);
  border-top-color: var(--brand-primary);
  border-radius: var(--radius-full);
  animation: spin 0.6s linear infinite;
}

.courses-empty {
  text-align: center;
  color: var(--text-muted);
  font-size: var(--text-xs);
  padding: var(--space-4);
}
</style>
