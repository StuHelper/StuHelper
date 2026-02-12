<template>
  <aside class="flex flex-col gap-1 overflow-y-auto">
    <!-- 分类标签 -->
    <div class="flex gap-1.5 overflow-x-auto pb-2 mb-2 border-b border-border scrollbar-none">
      <button
        v-for="cat in allCategories"
        :key="cat"
        class="shrink-0 px-3 py-1 text-xs rounded-full transition-colors duration-fast cursor-pointer border-none"
        :class="activeCategory === cat
          ? 'bg-primary text-white font-medium'
          : 'bg-bg-secondary text-text-secondary hover:bg-bg-hover'"
        @click="selectCategory(cat)"
      >
        {{ cat }}
      </button>
    </div>

    <!-- 可折叠院系分组 -->
    <div class="flex flex-col gap-1">
      <div v-for="dept in departments" :key="dept.id">
        <!-- 院系标题 -->
        <button
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
    <div v-if="departments.length === 0 && !deptLoading" class="text-center text-text-muted text-xs p-4">
      {{ t('common.empty.result') }}
    </div>

    <!-- 院系加载中 -->
    <div v-if="deptLoading" class="flex justify-center p-4">
      <div class="size-[18px] border-2 border-border border-t-primary rounded-full animate-spin" />
    </div>
  </aside>
</template>

<script setup lang="ts">
import { ref, computed, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { getDepartments, getCourses, getCourseCategories } from '@/api/course'
import type { Department, Course, CourseCategory } from '@/types/course'
import CourseListItem from './CourseListItem.vue'

const route = useRoute()

const { t } = useI18n()

// 分类相关
const categories = ref<CourseCategory[]>([])
const activeCategory = ref('')  // 空字符串表示"全部"

// 计算所有分类选项（"全部" + 后端分类）
const allCategories = computed(() => {
  return [t('review.filters.all'), ...categories.value.map(c => c.name)]
})

// 院系相关
const departments = ref<Department[]>([])
const deptLoading = ref(false)

// 展开/收起状态
const expandedDepts = reactive(new Set<number>())
const loadingDepts = reactive(new Set<number>())
const deptCourses = reactive(new Map<number, Course[]>())

// 判断某院系下是否有当前选中的课程
function isDeptActive(deptId: number): boolean {
  const activeCourseId = Number(route.params.id)
  if (!activeCourseId) return false
  const courses = deptCourses.get(deptId)
  return !!courses?.some(c => c.id === activeCourseId)
}

async function loadCategories() {
  try {
    const res = await getCourseCategories()
    categories.value = res.data || []
  } catch {
    categories.value = []
  }
}

async function loadDepartments() {
  deptLoading.value = true
  try {
    const categoryParam = activeCategory.value || undefined
    const res = await getDepartments(categoryParam)
    departments.value = res.data || []
    // 清空展开状态
    expandedDepts.clear()
  } catch {
    departments.value = []
  } finally {
    deptLoading.value = false
  }
}

function selectCategory(cat: string) {
  const allLabel = t('review.filters.all')
  activeCategory.value = cat === allLabel ? '' : cat
  // 分类变了，清空课程缓存
  deptCourses.clear()
  loadDepartments()
}

async function toggleDept(id: number) {
  if (expandedDepts.has(id)) {
    expandedDepts.delete(id)
    return
  }

  expandedDepts.add(id)

  // 已有缓存则不重新加载
  if (deptCourses.has(id)) return

  loadingDepts.add(id)
  try {
    const res = await getCourses(id)
    deptCourses.set(id, res.data?.list || [])
  } catch {
    deptCourses.set(id, [])
  } finally {
    loadingDepts.delete(id)
  }
}

onMounted(() => {
  loadCategories()
  loadDepartments()
})
</script>
