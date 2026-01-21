<template>
  <div class="course-detail-page" v-loading="loading">
    <div class="course-header" v-if="course">
      <div class="course-info">
        <h1 class="name">{{ course.name }}</h1>
        <div class="meta">
          <span v-if="course.teacherName" class="teacher">
            <el-icon><User /></el-icon>
            {{ course.teacherName }}
          </span>
          <span class="department">{{ course.departmentName }}</span>
        </div>
      </div>

      <div class="rating-summary">
        <div class="overall">
          <span class="score">{{ (course.avgRating || 0).toFixed(1) }}</span>
          <span class="label">综合评分</span>
        </div>
        <div class="dimensions">
          <div class="dim-item">
            <span class="dim-label">推荐</span>
            <el-progress
              :percentage="getRatingPercent(course.avgRecommend)"
              :stroke-width="8"
            />
          </div>
          <div class="dim-item">
            <span class="dim-label">内容</span>
            <el-progress
              :percentage="getRatingPercent(course.avgContent)"
              :stroke-width="8"
            />
          </div>
          <div class="dim-item">
            <span class="dim-label">工作量</span>
            <el-progress
              :percentage="getRatingPercent(course.avgWorkload)"
              :stroke-width="8"
            />
          </div>
          <div class="dim-item">
            <span class="dim-label">考核</span>
            <el-progress
              :percentage="getRatingPercent(course.avgExam)"
              :stroke-width="8"
            />
          </div>
        </div>
      </div>
    </div>

    <div class="reviews-section">
      <div class="section-header">
        <h2>课程测评 ({{ total }})</h2>
        <el-button type="primary" @click="showPostDialog = true">
          发布测评
        </el-button>
      </div>

      <div class="review-list">
        <ReviewCard v-for="r in reviews" :key="r.id" :review="r" />
        <el-empty v-if="!reviews.length" description="暂无测评" />
      </div>

      <el-pagination
        v-if="total > pageSize"
        :current-page="page"
        :page-size="pageSize"
        :total="total"
        layout="prev, pager, next"
        @current-change="handlePageChange"
      />
    </div>

    <PostReviewDialog
      v-model="showPostDialog"
      :course-id="courseId"
      @success="handlePostSuccess"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { User } from '@element-plus/icons-vue'
import ReviewCard from '@/components/review/ReviewCard.vue'
import PostReviewDialog from '@/components/review/PostReviewDialog.vue'
import { getCourseDetail } from '@/api/course'
import { getCourseReviews } from '@/api/review'
import type { Course } from '@/types/course'
import type { Review } from '@/types/review'

const route = useRoute()
const courseId = Number(route.params.id)

const loading = ref(false)
const course = ref<Course | null>(null)
const reviews = ref<Review[]>([])
const page = ref(1)
const pageSize = 10
const total = ref(0)
const showPostDialog = ref(false)

const getRatingPercent = (val?: number) => {
  if (val === undefined) return 50
  return ((val + 2) / 4) * 100
}

const fetchCourse = async () => {
  const res = await getCourseDetail(courseId)
  course.value = (res as any).data
}

const fetchReviews = async () => {
  const res = await getCourseReviews(courseId, page.value, pageSize)
  reviews.value = (res as any).data?.list || []
  total.value = (res as any).data?.total || 0
}

const handlePageChange = (p: number) => {
  page.value = p
  fetchReviews()
}

const handlePostSuccess = () => {
  showPostDialog.value = false
  fetchReviews()
  fetchCourse()
}

onMounted(async () => {
  loading.value = true
  try {
    await Promise.all([fetchCourse(), fetchReviews()])
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.course-detail-page {
  max-width: 900px;
  margin: 0 auto;
  padding: 24px;
}

.course-header {
  background: #fff;
  border-radius: 8px;
  padding: 24px;
  margin-bottom: 24px;
}

.name {
  font-size: 24px;
  font-weight: 600;
  margin: 0 0 12px 0;
}

.meta {
  display: flex;
  gap: 16px;
  color: #909399;
}

.teacher {
  display: flex;
  align-items: center;
  gap: 4px;
  color: #409eff;
}

.rating-summary {
  display: flex;
  gap: 32px;
  margin-top: 24px;
  padding-top: 24px;
  border-top: 1px solid #ebeef5;
}

.overall {
  text-align: center;
}

.score {
  font-size: 48px;
  font-weight: 600;
  color: #409eff;
}

.label {
  display: block;
  font-size: 14px;
  color: #909399;
}

.dimensions {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.dim-item {
  display: flex;
  align-items: center;
  gap: 12px;
}

.dim-label {
  width: 50px;
  font-size: 14px;
  color: #606266;
}

.reviews-section {
  background: #fff;
  border-radius: 8px;
  padding: 24px;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.section-header h2 {
  margin: 0;
  font-size: 18px;
}

.review-list {
  margin-bottom: 20px;
}
</style>
