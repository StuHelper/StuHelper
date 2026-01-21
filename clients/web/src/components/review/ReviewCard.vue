<template>
  <el-card class="review-card" shadow="hover">
    <template #header>
      <div class="card-header">
        <span class="title">{{ review.title || '无标题' }}</span>
        <span class="date">{{ formatDate(review.createdAt) }}</span>
      </div>
    </template>
    <div class="content">{{ review.content }}</div>
    <div class="footer">
      <span class="teacher" v-if="review.teacherName">
        {{ review.teacherName }} 老师
      </span>
      <span class="actions">
        <el-button size="small" :icon="'el-icon-thumb'">
          {{ review.likeCount }}
        </el-button>
      </span>
    </div>
  </el-card>
</template>

<script setup lang="ts">
import type { Review } from '@/types/review'

defineProps<{ review: Review }>()

const formatDate = (date: string) => {
  return new Date(date).toLocaleDateString('zh-CN')
}
</script>

<style scoped>
.review-card { margin-bottom: 16px; }
.card-header { display: flex; justify-content: space-between; }
.title { font-weight: 600; }
.date { color: #909399; font-size: 12px; }
.content { color: #606266; line-height: 1.6; }
.footer { margin-top: 12px; display: flex; justify-content: space-between; }
.teacher { color: #409eff; font-size: 14px; }
</style>
