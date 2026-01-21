<template>
  <div class="department-list">
    <el-collapse v-model="activeNames">
      <el-collapse-item
        v-for="category in categories"
        :key="category.key"
        :name="category.key"
      >
        <template #title>
          <span class="category-title">{{ category.label }}</span>
          <el-tag size="small" type="info">
            {{ getDeptCount(category.key) }}
          </el-tag>
        </template>
        <div class="dept-grid">
          <div
            v-for="dept in getDeptsByCategory(category.key)"
            :key="dept.id"
            :class="['dept-item', { active: selectedId === dept.id }]"
            @click="handleSelect(dept)"
          >
            {{ dept.name }}
          </div>
        </div>
      </el-collapse-item>
    </el-collapse>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { Department } from '@/types/course'

const props = defineProps<{
  departments: Department[]
  selectedId?: number
}>()

const emit = defineEmits<{
  select: [dept: Department]
}>()

const activeNames = ref(['school'])

const categories = [
  { key: 'school', label: '院系课程' },
  { key: 'elective', label: '通选课' },
  { key: 'pe', label: '体育课' },
  { key: 'english', label: '英语课' },
  { key: 'pols', label: '政治课' }
]

const getDeptsByCategory = (category: string) => {
  return props.departments.filter(d => d.category === category)
}

const getDeptCount = (category: string) => {
  return getDeptsByCategory(category).length
}

const handleSelect = (dept: Department) => {
  emit('select', dept)
}
</script>

<style scoped>
.department-list {
  background: #fff;
  border-radius: 8px;
}

.category-title {
  font-weight: 500;
  margin-right: 8px;
}

.dept-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
  gap: 8px;
  padding: 8px 0;
}

.dept-item {
  padding: 8px 12px;
  font-size: 13px;
  color: #606266;
  background: #f5f7fa;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.2s;
  text-align: center;
}

.dept-item:hover {
  background: #ecf5ff;
  color: #409eff;
}

.dept-item.active {
  background: #409eff;
  color: #fff;
}
</style>
