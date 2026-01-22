<template>
  <div class="latest-reviews-page">
    <div class="page-header">
      <h1>最新测评</h1>
      <p class="desc">查看社区最新发布的课程测评</p>
    </div>

    <div v-loading="loading" class="review-list">
      <ReviewCard v-for="r in reviews" :key="r.id" :review="r" />
      <el-empty v-if="!loading && !reviews.length" />
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
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import ReviewCard from '@/components/review/ReviewCard.vue'
import { getLatestReviews } from '@/api/review'
import type { Review } from '@/types/review'

const loading = ref(false)
const reviews = ref<Review[]>([])
const page = ref(1)
const pageSize = 20
const total = ref(0)

const fetchReviews = async () => {
  loading.value = true
  try {
    const res = await getLatestReviews(page.value, pageSize)
    reviews.value = res.data?.list || []
    total.value = res.data?.total || 0
  } finally {
    loading.value = false
  }
}

const handlePageChange = (p: number) => {
  page.value = p
  fetchReviews()
}

onMounted(fetchReviews)
</script>

<style scoped>
.latest-reviews-page {
  max-width: 800px;
  margin: 0 auto;
  padding: 24px;
}

.page-header {
  margin-bottom: 24px;
}

.page-header h1 {
  font-size: 24px;
  margin: 0 0 8px 0;
}

.desc {
  color: #909399;
  margin: 0;
}

.review-list {
  min-height: 400px;
  margin-bottom: 24px;
}
</style>
