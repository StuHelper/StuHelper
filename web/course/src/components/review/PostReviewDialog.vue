<template>
  <el-dialog
    v-model="visible"
    title="发布测评"
    width="600px"
    :close-on-click-modal="false"
  >
    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-position="top"
    >
      <el-form-item label="测评标题" prop="title">
        <el-input
          v-model="form.title"
          placeholder="给你的测评起个标题（可选）"
          maxlength="50"
          show-word-limit
        />
      </el-form-item>

      <el-form-item label="评分" prop="ratings" required>
        <RatingGroup v-model="form.ratings" />
      </el-form-item>

      <el-form-item label="测评内容" prop="content">
        <el-input
          v-model="form.content"
          type="textarea"
          :rows="6"
          placeholder="分享你的课程体验..."
          maxlength="2000"
          show-word-limit
        />
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" :loading="submitting" @click="handleSubmit">
        发布
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage } from 'element-plus'
import RatingGroup from './RatingGroup.vue'
import { postReview } from '@/api/review'
import type { RatingLevel } from '@/types/course'

const props = defineProps<{
  modelValue: boolean
  courseId: number
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  success: []
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

const formRef = ref<FormInstance>()
const submitting = ref(false)

const defaultRatings = () => ({
  recommend: 0 as RatingLevel,
  content: 0 as RatingLevel,
  workload: 0 as RatingLevel,
  exam: 0 as RatingLevel
})

const form = ref({
  title: '',
  content: '',
  ratings: defaultRatings()
})

const rules: FormRules = {
  content: [
    { required: true, message: '请输入测评内容', trigger: 'blur' },
    { min: 10, message: '测评内容至少10个字符', trigger: 'blur' }
  ]
}

watch(visible, (val) => {
  if (!val) {
    form.value = { title: '', content: '', ratings: defaultRatings() }
  }
})

const handleSubmit = async () => {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  submitting.value = true
  try {
    await postReview({
      courseId: props.courseId,
      title: form.value.title,
      content: form.value.content,
      ratingRecommend: form.value.ratings.recommend,
      ratingContent: form.value.ratings.content,
      ratingWorkload: form.value.ratings.workload,
      ratingExam: form.value.ratings.exam
    })
    ElMessage.success('发布成功')
    emit('success')
  } catch (e: any) {
    ElMessage.error(e.message || '发布失败')
  } finally {
    submitting.value = false
  }
}
</script>
