<template>
  <div class="home-page">
    <div class="hero-section">
      <h1 class="title">课程测评社区</h1>
      <p class="subtitle">分享你的课程体验，帮助同学选课</p>
      <SearchBar class="search" @select="handleCourseSelect" />
    </div>

    <div class="content-section">
      <el-row :gutter="24">
        <el-col :xs="24" :sm="12" :md="8">
          <div class="stat-card">
            <div class="stat-value">{{ stats.courseCount }}</div>
            <div class="stat-label">课程数量</div>
          </div>
        </el-col>
        <el-col :xs="24" :sm="12" :md="8">
          <div class="stat-card">
            <div class="stat-value">{{ stats.reviewCount }}</div>
            <div class="stat-label">测评数量</div>
          </div>
        </el-col>
        <el-col :xs="24" :sm="12" :md="8">
          <div class="stat-card">
            <div class="stat-value">{{ stats.departmentCount }}</div>
            <div class="stat-label">院系数量</div>
          </div>
        </el-col>
      </el-row>

      <div class="quick-links">
        <el-button type="primary" size="large" @click="router.push('/courses')">
          浏览全部课程
        </el-button>
        <el-button size="large" @click="router.push('/reviews/latest')">
          查看最新测评
        </el-button>
      </div>

      <div class="random-reviews" v-if="randomReviews.length">
        <h2 class="section-title">随机测评</h2>
        <el-row :gutter="16">
          <el-col
            v-for="review in randomReviews"
            :key="review.id"
            :xs="24"
            :sm="12"
            :md="8"
          >
            <ReviewCard :review="review" />
          </el-col>
        </el-row>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import SearchBar from '@/components/review/SearchBar.vue'
import ReviewCard from '@/components/review/ReviewCard.vue'
import { getLatestReviews } from '@/api/review'
import type { Course } from '@/types/course'
import type { Review } from '@/types/review'

const router = useRouter()

const stats = ref({
  courseCount: 0,
  reviewCount: 0,
  departmentCount: 0
})

const randomReviews = ref<Review[]>([])

const handleCourseSelect = (course: Course) => {
  router.push(`/courses/${course.id}`)
}

onMounted(async () => {
  try {
    const res = await getLatestReviews(1, 3)
    randomReviews.value = (res as any).data?.list || []
  } catch (e) {
    console.error('Failed to load reviews:', e)
  }
})
</script>

<style scoped>
.home-page {
  min-height: 100vh;
}

.hero-section {
  background: linear-gradient(135deg, #409eff 0%, #66b1ff 100%);
  padding: 60px 20px;
  text-align: center;
  color: #fff;
}

.title {
  font-size: 36px;
  font-weight: 600;
  margin: 0 0 12px 0;
}

.subtitle {
  font-size: 16px;
  opacity: 0.9;
  margin: 0 0 32px 0;
}

.search {
  max-width: 500px;
  margin: 0 auto;
}

.content-section {
  max-width: 1200px;
  margin: 0 auto;
  padding: 40px 20px;
}

.stat-card {
  background: #fff;
  border-radius: 8px;
  padding: 24px;
  text-align: center;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
  margin-bottom: 16px;
}

.stat-value {
  font-size: 32px;
  font-weight: 600;
  color: #409eff;
}

.stat-label {
  font-size: 14px;
  color: #909399;
  margin-top: 8px;
}

.quick-links {
  display: flex;
  justify-content: center;
  gap: 16px;
  margin: 40px 0;
}

.section-title {
  font-size: 20px;
  font-weight: 600;
  color: #303133;
  margin: 0 0 20px 0;
}
</style>
