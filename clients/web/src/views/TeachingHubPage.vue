<template>
  <div class="max-w-[720px] mx-auto py-8 px-6 animate-fade-in max-sm:py-6 max-sm:px-4">
    <header class="text-center mb-8">
      <h1 class="font-sans text-2xl font-extrabold tracking-tight text-text-primary m-0 mb-2">{{ t('teaching.title') }}</h1>
      <p class="text-sm text-text-muted m-0">{{ t('teaching.subtitle') }}</p>
    </header>

    <div class="grid grid-cols-2 gap-4 max-sm:grid-cols-1">
      <!-- 评课模块 -->
      <router-link
        to="/review"
        class="flex flex-col items-start p-5 bg-bg-card border border-border rounded-xl no-underline text-text-primary transition-all duration-base ease-smooth relative cursor-pointer shadow-card hover:border-primary/40 hover:shadow-md hover:-translate-y-0.5"
      >
        <div class="w-10 h-10 flex items-center justify-center rounded-lg mb-3 text-primary bg-primary/10">
          <MessageSquare :size="24" />
        </div>
        <h2 class="text-base font-semibold m-0 mb-1">{{ t('teaching.modules.review.name') }}</h2>
        <p class="text-xs text-text-muted m-0 leading-relaxed">{{ t('teaching.modules.review.description') }}</p>
      </router-link>

      <!-- 教师模块 -->
      <router-link
        to="/teacher"
        class="flex flex-col items-start p-5 bg-bg-card border border-border rounded-xl no-underline text-text-primary transition-all duration-base ease-smooth relative cursor-pointer shadow-card hover:border-primary/40 hover:shadow-md hover:-translate-y-0.5"
      >
        <div class="w-10 h-10 flex items-center justify-center rounded-lg mb-3 text-accent bg-accent/10">
          <User :size="24" />
        </div>
        <h2 class="text-base font-semibold m-0 mb-1">{{ t('teaching.modules.teacher.name') }}</h2>
        <p class="text-xs text-text-muted m-0 leading-relaxed">{{ t('teaching.modules.teacher.description') }}</p>
      </router-link>

      <!-- SPOC 模块（即将上线） -->
      <div class="flex flex-col items-start p-5 bg-bg-card border border-border rounded-xl text-text-primary relative opacity-60 cursor-default shadow-card">
        <div class="w-10 h-10 flex items-center justify-center rounded-lg mb-3 text-text-muted bg-bg-secondary">
          <BookOpen :size="24" />
        </div>
        <h2 class="text-base font-semibold m-0 mb-1">{{ t('teaching.modules.spoc.name') }}</h2>
        <p class="text-xs text-text-muted m-0 leading-relaxed">{{ t('teaching.modules.spoc.description') }}</p>
        <span class="inline-block mt-3 py-0.5 px-2.5 text-xs text-text-muted bg-bg-secondary rounded-full">{{ t('teaching.comingSoon') }}</span>
      </div>

      <!-- 资源模块（即将上线） -->
      <div class="flex flex-col items-start p-5 bg-bg-card border border-border rounded-xl text-text-primary relative opacity-60 cursor-default shadow-card">
        <div class="w-10 h-10 flex items-center justify-center rounded-lg mb-3 text-text-muted bg-bg-secondary">
          <FileText :size="24" />
        </div>
        <h2 class="text-base font-semibold m-0 mb-1">{{ t('teaching.modules.resource.name') }}</h2>
        <p class="text-xs text-text-muted m-0 leading-relaxed">{{ t('teaching.modules.resource.description') }}</p>
        <span class="inline-block mt-3 py-0.5 px-2.5 text-xs text-text-muted bg-bg-secondary rounded-full">{{ t('teaching.comingSoon') }}</span>
      </div>
    </div>

    <footer v-if="stats" class="flex items-center justify-center gap-3 mt-8 text-text-muted text-xs">
      <span class="font-mono">
        {{ t('teaching.stats.courses', { count: stats.courseCount }) }}
      </span>
      <span class="w-[3px] h-[3px] rounded-full bg-text-muted"></span>
      <span class="font-mono">
        {{ t('teaching.stats.reviews', { count: stats.reviewCount }) }}
      </span>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { MessageSquare, User, BookOpen, FileText } from 'lucide-vue-next'
import { getStats, type Stats } from '@/api/course'

const { t } = useI18n()
const stats = ref<Stats | null>(null)

onMounted(async () => {
  try {
    const res = await getStats()
    stats.value = res.data
  } catch {
    stats.value = null
  }
})
</script>
