<template>
  <div class="course-list-page">
    <!-- Sidebar -->
    <aside class="sidebar" :class="{ open: sidebarOpen }">
      <div class="sidebar-header">
        <h3 class="sidebar-title">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M3 21h18"/>
            <path d="M5 21V7l8-4v18"/>
            <path d="M19 21V11l-6-4"/>
          </svg>
          选择院系
        </h3>
        <button class="sidebar-close" @click="sidebarOpen = false">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M18 6L6 18M6 6l12 12"/>
          </svg>
        </button>
      </div>
      <DepartmentList
        :departments="store.departments"
        :selected-id="selectedDeptId"
        @select="handleDeptSelect"
      />
    </aside>

    <!-- Mobile Sidebar Toggle -->
    <button class="sidebar-toggle" @click="sidebarOpen = true">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M3 12h18M3 6h18M3 18h18"/>
      </svg>
      <span>筛选院系</span>
    </button>

    <!-- Main Content -->
    <main class="main-content">
      <header class="page-header">
        <div class="header-info">
          <h1 class="page-title">{{ selectedDept?.name || '全部课程' }}</h1>
          <span v-if="store.courses.length" class="course-count">
            {{ store.courses.length }} 门课程
          </span>
        </div>
        <SearchBar @select="handleCourseSelect" />
      </header>

      <!-- Loading State -->
      <div v-if="store.loading" class="loading-grid">
        <div v-for="i in 6" :key="i" class="skeleton-card">
          <div class="skeleton-content">
            <div class="skeleton-title"></div>
            <div class="skeleton-meta"></div>
          </div>
          <div class="skeleton-stat"></div>
        </div>
      </div>

      <!-- Course Grid -->
      <div v-else-if="store.courses.length" class="course-grid">
        <CourseCard
          v-for="course in store.courses"
          :key="course.id"
          :course="course"
          @click="handleCourseSelect"
        />
      </div>

      <!-- Empty State -->
      <div v-else class="empty-state">
        <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/>
          <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/>
        </svg>
        <p class="empty-text">暂无课程</p>
        <p class="empty-hint">请选择其他院系或使用搜索</p>
      </div>
    </main>

    <!-- Sidebar Overlay -->
    <div v-if="sidebarOpen" class="sidebar-overlay" @click="sidebarOpen = false"></div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useCourseStore } from '@/stores/courseReview'
import DepartmentList from '@/components/review/DepartmentList.vue'
import CourseCard from '@/components/review/CourseCard.vue'
import SearchBar from '@/components/review/SearchBar.vue'
import type { Department, Course } from '@/types/course'

const router = useRouter()
const store = useCourseStore()

const selectedDeptId = ref<number>()
const sidebarOpen = ref(false)

const selectedDept = computed(() =>
  store.departments.find(d => d.id === selectedDeptId.value)
)

const handleDeptSelect = async (dept: Department) => {
  selectedDeptId.value = dept.id
  sidebarOpen.value = false
  await store.fetchCourses(dept.id)
}

const handleCourseSelect = (course: Course) => {
  router.push(`/review/courses/${course.id}`)
}

onMounted(() => {
  store.fetchDepartments()
})
</script>

<style scoped>
.course-list-page {
  display: flex;
  gap: var(--space-6);
  max-width: 1400px;
  margin: 0 auto;
  padding: var(--space-6);
  min-height: calc(100vh - 120px);
}

/* Sidebar */
.sidebar {
  width: 280px;
  flex-shrink: 0;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: var(--space-4);
  position: sticky;
  top: calc(var(--space-6) + 60px);
  max-height: calc(100vh - 120px);
  overflow-y: auto;
}

.sidebar-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--space-4);
  padding-bottom: var(--space-3);
  border-bottom: 1px solid var(--border);
}

.sidebar-title {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-family: var(--font-display);
  font-size: var(--text-base);
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.sidebar-title svg {
  width: 18px;
  height: 18px;
  color: var(--accent);
}

.sidebar-close {
  display: none;
  width: 32px;
  height: 32px;
  align-items: center;
  justify-content: center;
  color: var(--text-muted);
  border-radius: var(--radius-md);
  transition: all var(--duration-fast);
}

.sidebar-close:hover {
  color: var(--text-primary);
  background: var(--bg-hover);
}

.sidebar-close svg {
  width: 20px;
  height: 20px;
}

/* Sidebar Toggle (Mobile) */
.sidebar-toggle {
  display: none;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-3) var(--space-4);
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  color: var(--text-secondary);
  font-size: var(--text-sm);
  margin-bottom: var(--space-4);
}

.sidebar-toggle svg {
  width: 18px;
  height: 18px;
}

/* Main Content */
.main-content {
  flex: 1;
  min-width: 0;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: var(--space-4);
  margin-bottom: var(--space-6);
  flex-wrap: wrap;
}

.header-info {
  display: flex;
  align-items: baseline;
  gap: var(--space-3);
}

.page-title {
  font-family: var(--font-display);
  font-size: var(--text-xl);
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.course-count {
  font-size: var(--text-sm);
  color: var(--text-muted);
}

/* Loading Grid */
.loading-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: var(--space-4);
}

.skeleton-card {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  padding: var(--space-4);
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}

.skeleton-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.skeleton-title {
  width: 70%;
  height: 18px;
  background: var(--bg-elevated);
  border-radius: var(--radius-sm);
  animation: pulse 1.5s ease-in-out infinite;
}

.skeleton-meta {
  width: 50%;
  height: 14px;
  background: var(--bg-elevated);
  border-radius: var(--radius-sm);
  animation: pulse 1.5s ease-in-out infinite;
}

.skeleton-stat {
  width: 50px;
  height: 40px;
  background: var(--bg-elevated);
  border-radius: var(--radius-sm);
  animation: pulse 1.5s ease-in-out infinite;
}

/* Course Grid */
.course-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: var(--space-4);
}

/* Empty State */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--space-16) var(--space-4);
  text-align: center;
}

.empty-icon {
  width: 80px;
  height: 80px;
  color: var(--text-muted);
  opacity: 0.4;
  margin-bottom: var(--space-4);
}

.empty-text {
  font-size: var(--text-lg);
  color: var(--text-secondary);
  margin: 0 0 var(--space-2) 0;
}

.empty-hint {
  font-size: var(--text-sm);
  color: var(--text-muted);
  margin: 0;
}

/* Sidebar Overlay */
.sidebar-overlay {
  display: none;
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  z-index: 40;
}

/* Responsive */
@media (max-width: 768px) {
  .course-list-page {
    flex-direction: column;
    padding: var(--space-4);
  }

  .sidebar {
    position: fixed;
    top: 0;
    left: 0;
    bottom: 0;
    width: 280px;
    max-height: 100vh;
    border-radius: 0;
    z-index: 50;
    transform: translateX(-100%);
    transition: transform var(--duration-base) var(--ease-out);
  }

  .sidebar.open {
    transform: translateX(0);
  }

  .sidebar-close {
    display: flex;
  }

  .sidebar-toggle {
    display: flex;
  }

  .sidebar-overlay {
    display: block;
  }

  .loading-grid,
  .course-grid {
    grid-template-columns: 1fr;
  }
}
</style>
