<template>
  <aside class="flex flex-col gap-1 overflow-y-auto">
    <!-- 分类标签 -->
    <div class="flex flex-wrap gap-1.5 pb-2 mb-2 border-b border-border-light">
      <button
        type="button"
        v-for="cat in allCategories"
        :key="cat.id"
        class="max-w-full px-3 py-1 text-xs rounded-full transition-colors duration-fast cursor-pointer border-none leading-tight break-words text-left"
        :class="activeCategory === cat.id
          ? 'bg-primary text-white font-medium'
          : 'bg-bg-secondary text-text-secondary hover:bg-bg-hover'"
        @click="selectCategory(cat.id)"
      >
        {{ cat.name }}
      </button>
    </div>

    <div v-if="sidebarError" class="rounded-md bg-warning/10 p-3 text-xs text-warning" role="alert">
      <p class="m-0 mb-2">{{ sidebarError }}</p>
      <button
        type="button"
        class="text-primary underline underline-offset-2"
        @click="retrySidebar"
      >
        {{ t('common.actions.retry') }}
      </button>
    </div>

    <!-- 可折叠院系分组 -->
    <div v-else class="flex flex-col gap-1">
      <div v-for="dept in departments" :key="dept.id">
        <!-- 院系标题 -->
        <button
          type="button"
          class="w-full text-left px-3 py-1.5 text-sm font-medium cursor-pointer transition-colors duration-fast bg-transparent border-none rounded-md"
          :class="expandedDepts.has(dept.id)
            ? 'text-primary font-semibold'
            : 'text-text-muted hover:text-text-primary'"
          @click="toggleDept(dept.id)"
        >
          {{ dept.name }}
        </button>

        <!-- 展开的课程列表（整体框） -->
        <div
          v-if="expandedDepts.has(dept.id)"
          class="ml-2 rounded-lg border transition-colors duration-fast overflow-hidden"
          :class="isDeptActive(dept.id)
            ? 'border-primary/40 bg-primary/[0.04]'
            : 'border-border-light bg-bg-card'"
        >
          <div v-if="loadingDepts.has(dept.id)" class="flex justify-center p-3">
            <div class="size-4 border-2 border-border border-t-primary rounded-full animate-spin" />
          </div>
          <div v-else-if="failedDepts.has(dept.id)" class="text-center text-warning text-xs p-3" role="alert">
            <p class="m-0 mb-2">{{ t('common.loadFailed') }}</p>
            <button
              type="button"
              class="text-primary underline underline-offset-2"
              @click.stop="retryDept(dept.id)"
            >
              {{ t('common.actions.retry') }}
            </button>
          </div>
          <template v-else-if="(deptCourses.get(dept.id) || []).length > 0">
            <CourseListItem
              v-for="course in deptCourses.get(dept.id)"
              :key="course.id"
              :course="course"
            />
          </template>
          <div v-else class="text-center text-text-muted text-xs p-3">
            {{ t('common.empty.result') }}
          </div>
        </div>
      </div>
    </div>

    <!-- 空状态 -->
    <div v-if="!sidebarError && departments.length === 0 && !deptLoading" class="text-center text-text-muted text-xs p-4">
      {{ t('common.empty.result') }}
    </div>

    <!-- 院系加载中 -->
    <div v-if="deptLoading" class="flex justify-center p-4">
      <div class="size-[18px] border-2 border-border border-t-primary rounded-full animate-spin" />
    </div>
  </aside>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { api } from '@/api'
import type { Department, Course, CourseCategory } from '@stuhelper/shared/course'
import {
  readCourseCategoryArrayPayload,
  readCourseListPayload,
  readDepartmentArrayPayload,
} from '@/modules/course/coursePayload'
import CourseListItem from './CourseListItem.vue'

const route = useRoute()

const { t } = useI18n()

// 分类相关
const categories = ref<CourseCategory[]>([])
const activeCategory = ref('')  // 空字符串表示"全部"
const sidebarError = ref('')

// 计算所有分类选项（"全部" + 后端分类），使用 id 而非翻译文本作为标识
const allCategories = computed(() => {
  return [
    { id: '', name: t('review.filters.all') },
    ...categories.value.map(c => ({ id: c.name, name: c.name }))
  ]
})

// 院系相关
const departments = ref<Department[]>([])
const deptLoading = ref(false)

// 展开/收起状态
const expandedDepts = ref(new Set<number>())
const loadingDepts = ref(new Set<number>())
const failedDepts = ref(new Set<number>())
const deptCourses = ref(new Map<number, Course[]>())

// 判断某院系下是否有当前选中的课程
function isDeptActive(deptId: number): boolean {
  const activeCourseId = Number(route.params.id)
  if (!activeCourseId) return false
  const courses = deptCourses.value.get(deptId)
  return !!courses?.some(c => c.id === activeCourseId)
}

async function loadCategories() {
  try {
    const res = await api.course.getCategories()
    categories.value = readCourseCategoryArrayPayload(
      res.data?.data,
      'Invalid course categories response',
    )
  } catch (_error) { void _error;
    categories.value = []
    sidebarError.value = t('common.loadFailed')
  }
}

async function loadDepartments() {
  deptLoading.value = true
  sidebarError.value = ''
  try {
    const normalizedCategory = activeCategory.value.trim()
    const res = await api.course.getDepartments(
      normalizedCategory ? { category: normalizedCategory } : {}
    )
    departments.value = readDepartmentArrayPayload(
      res.data?.data,
      'Invalid departments response',
    )
    // 清空展开状态
    expandedDepts.value = new Set()
  } catch (_error) { void _error;
    departments.value = []
    sidebarError.value = t('common.loadFailed')
  } finally {
    deptLoading.value = false
  }
}

function retrySidebar() {
  sidebarError.value = ''
  loadCategories()
  loadDepartments()
}

// 分类版本号，用于防止旧分类的请求写入新分类的缓存
let categoryVersion = 0

function selectCategory(catId: string) {
  activeCategory.value = catId
  categoryVersion++
  // 分类变了，清空课程缓存
  deptCourses.value.clear()
  failedDepts.value = new Set()
  loadDepartments()
}

function removeFailedDept(id: number) {
  const next = new Set(failedDepts.value)
  next.delete(id)
  failedDepts.value = next
}

function addFailedDept(id: number) {
  failedDepts.value = new Set([...failedDepts.value, id])
}

function retryDept(id: number) {
  removeFailedDept(id)
  deptCourses.value.delete(id)
  loadDeptCourses(id)
}

async function toggleDept(id: number) {
  if (expandedDepts.value.has(id)) {
    const next = new Set(expandedDepts.value)
    next.delete(id)
    expandedDepts.value = next
    return
  }

  expandedDepts.value = new Set([...expandedDepts.value, id])

  // 已有缓存则不重新请求
  if (deptCourses.value.has(id)) return
  // 正在加载时不重复请求
  if (loadingDepts.value.has(id)) return

  await loadDeptCourses(id)
}

async function loadDeptCourses(id: number) {
  const version = categoryVersion
  loadingDepts.value = new Set([...loadingDepts.value, id])
  try {
    const normalizedCategory = activeCategory.value.trim()
    const res = await api.course.getCourses(
      {
        page: 1,
        pageSize: 100,
        departmentID: id,
        ...(normalizedCategory ? { category: normalizedCategory } : {}),
      }
    )
    // 仅当分类未切换时才写入缓存
    if (version === categoryVersion) {
      const courses = readCourseListPayload(
        res.data?.data,
        'Invalid department courses response',
      )
      deptCourses.value.set(id, courses)
      removeFailedDept(id)
    }
  } catch (_error) { void _error;
    if (version === categoryVersion) {
      deptCourses.value.delete(id)
      addFailedDept(id)
    }
  } finally {
    const next = new Set(loadingDepts.value)
    next.delete(id)
    loadingDepts.value = next
  }
}

onMounted(() => {
  loadCategories()
  loadDepartments()
})
</script>
