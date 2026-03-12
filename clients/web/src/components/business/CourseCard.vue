<script setup lang="ts">
import { StarIcon, UserIcon, AcademicCapIcon } from '@heroicons/vue/24/solid'
import Card from '@/components/ui/Card.vue'
import type { components } from '@stuhelper/shared/types'

interface Props {
  course: components['schemas']['Course'] & {
    averageRating?: number
    teacherName?: string
    hours?: number
    isRequired?: boolean
  }
}

defineProps<Props>()
</script>

<template>
  <Card class="group p-6 cursor-pointer transition-all duration-300 hover:scale-105 hover:shadow-xl">
    <!-- 课程名称和代码 -->
    <div class="mb-3">
      <h3 class="text-xl font-bold mb-1 group-hover:text-primary-600 dark:group-hover:text-primary-400 transition-colors">
        {{ course.name }}
      </h3>
      <p class="text-sm text-gray-500 dark:text-gray-400">{{ course.code }}</p>
    </div>

    <!-- 评分 -->
    <div v-if="course.averageRating" class="flex items-center gap-1 mb-3">
      <StarIcon v-for="i in 5" :key="i"
        :class="i <= Math.round(course.averageRating ?? 0) ? 'text-yellow-400' : 'text-gray-300 dark:text-gray-600'"
        class="w-5 h-5" />
      <span class="ml-1 text-sm text-gray-600 dark:text-gray-300">{{ (course.averageRating ?? 0).toFixed(1) }}</span>
    </div>

    <!-- 教师信息 -->
    <div v-if="course.teacherName" class="flex items-center gap-2 mb-3">
      <div class="w-8 h-8 rounded-full bg-gradient-to-br from-primary-400 to-secondary-400 flex items-center justify-center">
        <UserIcon class="w-5 h-5 text-white" />
      </div>
      <span class="text-sm text-gray-700 dark:text-gray-200">{{ course.teacherName }}</span>
    </div>

    <!-- 学分和学时 -->
    <div class="flex items-center gap-4 mb-3 text-sm text-gray-600 dark:text-gray-300">
      <div class="flex items-center gap-1">
        <AcademicCapIcon class="w-4 h-4" />
        <span>{{ course.credits }} 学分</span>
      </div>
      <span>{{ course.hours ?? 0 }} 学时</span>
    </div>

    <!-- 标签 -->
    <div class="flex gap-2 flex-wrap">
      <span class="px-2 py-1 text-xs rounded-full bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300">
        {{ course.isRequired ?? false ? '必修' : '选修' }}
      </span>
      <span class="px-2 py-1 text-xs rounded-full bg-secondary-100 text-secondary-700 dark:bg-secondary-900/30 dark:text-secondary-300">
        {{ course.departmentName }}
      </span>
    </div>
  </Card>
</template>
