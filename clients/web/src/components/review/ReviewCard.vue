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
        <el-button size="small" :loading="voting" @click="handleLike">
          <el-icon><Pointer /></el-icon>
          {{ review.likeCount }}
        </el-button>
      </span>
    </div>
  </el-card>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Pointer } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import type { Review } from '@/types/review'
import { formatDate } from '@/utils/date'
import { voteReview } from '@/api/review'

const props = defineProps<{ review: Review }>()
const emit = defineEmits<{ voted: [] }>()

const voting = ref(false)

const handleLike = async () => {
  voting.value = true
  try {
    await voteReview(Number(props.review.id), 'like')
    emit('voted')
  } catch (e) {
    const message = e instanceof Error ? e.message : '投票失败'
    ElMessage.error(message)
  } finally {
    voting.value = false
  }
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
