<template>
  <div class="course-list-page">
    <el-row :gutter="24">
      <el-col :xs="24" :md="6">
        <div class="sidebar">
          <h3 class="sidebar-title">选择院系</h3>
          <DepartmentList
            :departments="store.departments"
            :selected-id="selectedDeptId"
            @select="handleDeptSelect"
          />
        </div>
      </el-col>

      <el-col :xs="24" :md="18">
        <div class="main-content">
          <div class="header">
            <h2 class="page-title">
              {{ selectedDept?.name || '全部课程' }}
            </h2>
            <SearchBar @select="handleCourseSelect" />
          </div>

          <div v-loading="store.loading" class="course-grid">
            <CourseCard
              v-for="course in store.courses"
              :key="course.id"
              :course="course"
              @click="handleCourseSelect"
            />
            <el-empty v-if="!store.loading && !store.courses.length" />
          </div>
        </div>
      </el-col>
    </el-row>
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

const selectedDept = computed(() =>
  store.departments.find(d => d.id === selectedDeptId.value)
)

const handleDeptSelect = async (dept: Department) => {
  selectedDeptId.value = dept.id
  await store.fetchCourses(dept.id)
}

const handleCourseSelect = (course: Course) => {
  router.push(`/courses/${course.id}`)
}

onMounted(() => {
  store.fetchDepartments()
})
</script>

<style scoped>
.course-list-page {
  max-width: 1400px;
  margin: 0 auto;
  padding: 24px;
}

.sidebar {
  background: #fff;
  border-radius: 8px;
  padding: 16px;
  position: sticky;
  top: 24px;
}

.sidebar-title {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
  margin: 0 0 16px 0;
}

.main-content {
  background: #fff;
  border-radius: 8px;
  padding: 24px;
  min-height: 600px;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
  flex-wrap: wrap;
  gap: 16px;
}

.page-title {
  font-size: 20px;
  font-weight: 600;
  color: #303133;
  margin: 0;
}

.course-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}
</style>
