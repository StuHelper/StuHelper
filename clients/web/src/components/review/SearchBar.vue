<template>
  <div class="search-bar">
    <el-autocomplete
      v-model="keyword"
      :fetch-suggestions="handleSearch"
      :trigger-on-focus="false"
      :debounce="300"
      placeholder="搜索课程名称、教师..."
      clearable
      @select="handleSelect"
    >
      <template #prefix>
        <el-icon><Search /></el-icon>
      </template>
      <template #default="{ item }">
        <div class="search-item">
          <span class="name">{{ item.name }}</span>
          <span class="teacher" v-if="item.teacherName">{{ item.teacherName }}</span>
        </div>
      </template>
    </el-autocomplete>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Search } from '@element-plus/icons-vue'
import { searchCourses } from '@/api/course'
import type { Course } from '@/types/course'

const keyword = ref('')

const emit = defineEmits<{
  select: [course: Course]
}>()

interface SuggestionItem extends Course {
  value: string
}

const handleSearch = async (
  query: string,
  cb: (results: SuggestionItem[]) => void
) => {
  if (!query.trim()) {
    cb([])
    return
  }
  try {
    const res = await searchCourses(query)
    const courses = (res as any).data || []
    cb(courses.map((c: Course) => ({ ...c, value: c.name })))
  } catch {
    cb([])
  }
}

const handleSelect = (item: SuggestionItem) => {
  emit('select', item)
}
</script>

<style scoped>
.search-bar {
  width: 100%;
  max-width: 500px;
}

.search-bar :deep(.el-autocomplete) {
  width: 100%;
}

.search-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 4px 0;
}

.name {
  font-size: 14px;
  color: #303133;
}

.teacher {
  font-size: 12px;
  color: #909399;
}
</style>
