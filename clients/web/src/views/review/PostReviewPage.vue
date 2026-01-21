<template>
  <div class="post-review-page">
    <div class="page-header">
      <h1>发布测评</h1>
      <p class="desc">分享你的课程体验，帮助其他同学选课</p>
    </div>

    <el-card class="form-card">
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-position="top"
      >
        <el-form-item label="选择课程" prop="courseId">
          <SearchBar @select="handleCourseSelect" />
          <div v-if="selectedCourse" class="selected-course">
            <el-tag>{{ selectedCourse.name }}</el-tag>
            <span v-if="selectedCourse.teacherName">
              {{ selectedCourse.teacherName }} 老师
            </span>
          </div>
        </el-form-item>

        <el-form-item label="测评标题" prop="title">
          <el-input
            v-model="form.title"
            placeholder="给你的测评起个标题（可选）"
            maxlength="50"
            show-word-limit
          />
        </el-form-item>

        <el-form-item label="课程评分" required>
          <RatingGroup v-model="form.ratings" />
        </el-form-item>

        <el-form-item label="测评内容" prop="content">
          <el-input
            v-model="form.content"
            type="textarea"
            :rows="8"
            placeholder="分享你的课程体验，包括课程内容、教学方式、考核方式等..."
            maxlength="2000"
            show-word-limit
          />
        </el-form-item>

        <el-form-item>
          <el-button type="primary" :loading="submitting" @click="handleSubmit">
            发布测评
          </el-button>
          <el-button @click="router.back()">取消</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage } from 'element-plus'
import SearchBar from '@/components/review/SearchBar.vue'
import RatingGroup from '@/components/review/RatingGroup.vue'
import { postReview } from '@/api/review'
import type { Course, RatingLevel } from '@/types/course'

const router = useRouter()
const formRef = ref<FormInstance>()
const submitting = ref(false)
const selectedCourse = ref<Course | null>(null)

const defaultRatings = () => ({
  recommend: 0 as RatingLevel,
  content: 0 as RatingLevel,
  workload: 0 as RatingLevel,
  exam: 0 as RatingLevel
})

const form = ref({
  courseId: 0,
  title: '',
  content: '',
  ratings: defaultRatings()
})

const rules: FormRules = {
  courseId: [
    { required: true, message: '请选择课程', trigger: 'change' }
  ],
  content: [
    { required: true, message: '请输入测评内容', trigger: 'blur' },
    { min: 10, message: '测评内容至少10个字符', trigger: 'blur' }
  ]
}

const handleCourseSelect = (course: Course) => {
  selectedCourse.value = course
  form.value.courseId = course.id
}

const handleSubmit = async () => {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  if (!form.value.courseId) {
    ElMessage.warning('请先选择课程')
    return
  }

  submitting.value = true
  try {
    await postReview({
      courseId: form.value.courseId,
      title: form.value.title,
      content: form.value.content,
      ratingRecommend: form.value.ratings.recommend,
      ratingContent: form.value.ratings.content,
      ratingWorkload: form.value.ratings.workload,
      ratingExam: form.value.ratings.exam
    })
    ElMessage.success('发布成功')
    router.push(`/courses/${form.value.courseId}`)
  } catch (e: any) {
    ElMessage.error(e.message || '发布失败')
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.post-review-page {
  max-width: 700px;
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

.form-card {
  padding: 8px;
}

.selected-course {
  margin-top: 8px;
  display: flex;
  align-items: center;
  gap: 8px;
}
</style>
