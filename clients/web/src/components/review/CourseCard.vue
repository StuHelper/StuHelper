<template>
  <el-card class="course-card" shadow="hover" @click="handleClick">
    <div class="course-info">
      <h3 class="name">{{ course.name }}</h3>
      <div class="meta">
        <span v-if="course.teacherName" class="teacher">
          <el-icon><User /></el-icon>
          {{ course.teacherName }}
        </span>
        <span class="department">{{ course.departmentName }}</span>
      </div>
    </div>
    <div class="course-stats">
      <div class="rating" v-if="course.avgRating !== undefined">
        <span class="score">{{ course.avgRating.toFixed(1) }}</span>
        <span class="label">综合评分</span>
      </div>
      <div class="review-count">
        <span class="count">{{ course.reviewCount || 0 }}</span>
        <span class="label">条测评</span>
      </div>
    </div>
  </el-card>
</template>

<script setup lang="ts">
import { User } from '@element-plus/icons-vue'
import type { Course } from '@/types/course'

const props = defineProps<{
  course: Course
}>()

const emit = defineEmits<{
  click: [course: Course]
}>()

const handleClick = () => {
  emit('click', props.course)
}
</script>

<style scoped>
.course-card {
  cursor: pointer;
  transition: transform 0.2s;
}

.course-card:hover {
  transform: translateY(-2px);
}

.course-info {
  margin-bottom: 12px;
}

.name {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
  margin: 0 0 8px 0;
}

.meta {
  display: flex;
  gap: 12px;
  font-size: 13px;
  color: #909399;
}

.teacher {
  display: flex;
  align-items: center;
  gap: 4px;
  color: #409eff;
}

.course-stats {
  display: flex;
  gap: 24px;
  padding-top: 12px;
  border-top: 1px solid #ebeef5;
}

.rating,
.review-count {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.score {
  font-size: 24px;
  font-weight: 600;
  color: #409eff;
}

.count {
  font-size: 20px;
  font-weight: 600;
  color: #606266;
}

.label {
  font-size: 12px;
  color: #909399;
}
</style>
