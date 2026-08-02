<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { api } from '@/api'
import { isNonArrayRecord as isRecord } from '@stuhelper/shared/utils'
import HeroSection from '@/components/business/HeroSection.vue'
import FeatureCard from '@/components/business/FeatureCard.vue'
import CountUp from '@/components/animated/CountUp.vue'
import ScrollReveal from '@/components/animated/ScrollReveal.vue'
import {
  AcademicCapIcon,
  DocumentTextIcon,
  UserGroupIcon,
} from '@heroicons/vue/24/outline'

const router = useRouter()
const { t } = useI18n()

const courseCount = ref(0)
const reviewCount = ref(0)
const userCount = ref(0)
const statsLoadError = ref('')

function isNonNegativeNumber(value: unknown): value is number {
  return typeof value === 'number' && Number.isFinite(value) && value >= 0
}

function readCourseStatsPayload(payload: unknown): { courseCount: number } {
  if (!isRecord(payload)) {
    throw new Error('Invalid course stats response')
  }

  const courseCount = payload.courseCount
  const departmentCount = payload.departmentCount
  if (!isNonNegativeNumber(courseCount) || !isNonNegativeNumber(departmentCount)) {
    throw new Error('Invalid course stats response')
  }

  return { courseCount }
}

function readReviewStatsPayload(payload: unknown): { reviewCount: number; userCount: number } {
  if (!isRecord(payload)) {
    throw new Error('Invalid review stats response')
  }

  const courseCount = payload.courseCount
  const reviewCount = payload.reviewCount
  const departmentCount = payload.departmentCount
  const userCount = payload.userCount
  if (
    !isNonNegativeNumber(courseCount) ||
    !isNonNegativeNumber(reviewCount) ||
    !isNonNegativeNumber(departmentCount) ||
    !isNonNegativeNumber(userCount)
  ) {
    throw new Error('Invalid review stats response')
  }

  return { reviewCount, userCount }
}

onMounted(async () => {
  statsLoadError.value = ''
  const [courseStatsRes, reviewStatsRes] = await Promise.allSettled([
    api.course.getStats(),
    api.review.getReviewStats(),
  ])

  try {
    if (courseStatsRes.status !== 'fulfilled' || reviewStatsRes.status !== 'fulfilled') {
      throw new Error('Stats request failed')
    }

    const courseStats = readCourseStatsPayload(courseStatsRes.value.data?.data)
    const reviewStats = readReviewStatsPayload(reviewStatsRes.value.data?.data)
    courseCount.value = courseStats.courseCount
    reviewCount.value = reviewStats.reviewCount
    userCount.value = reviewStats.userCount
  } catch (_error) { void _error;
    courseCount.value = 0
    reviewCount.value = 0
    userCount.value = 0
    statsLoadError.value = t('common.loadFailed')
  }
})

const features = computed(() => [
  {
    icon: AcademicCapIcon,
    title: t('home.landing.features.reviewCenter.title'),
    description: t('home.landing.features.reviewCenter.description'),
    color: 'var(--color-secondary)',
    to: '/courses/reviews',
    stats: reviewCount.value > 0
      ? t('home.landing.features.reviewCenter.statsCount', { count: reviewCount.value })
      : t('home.landing.features.reviewCenter.stats')
  },
  {
    icon: UserGroupIcon,
    title: t('home.landing.features.teacherProfile.title'),
    description: t('home.landing.features.teacherProfile.description'),
    color: 'var(--color-primary)',
    to: '/teachers',
    stats: t('home.landing.features.teacherProfile.stats')
  },
  {
    icon: DocumentTextIcon,
    title: t('home.landing.features.resourceHub.title'),
    description: t('home.landing.features.resourceHub.description'),
    color: 'var(--color-accent)',
    to: '/resources',
    stats: t('home.landing.features.resourceHub.stats')
  },
])

const stats = computed(() => [
  { target: courseCount.value, label: t('home.landing.stats.courses') },
  { target: reviewCount.value, label: t('home.landing.stats.reviews') },
  { target: userCount.value, label: t('home.landing.stats.users') },
])

const onExplore = () => {
  router.push('/courses')
}

const onLearnMore = () => {
  router.push('/courses/about')
}
</script>

<template>
  <div class="min-h-screen">
    <HeroSection
      :title="t('home.landing.hero.title')"
      :subtitle="t('home.landing.hero.subtitle')"
      @explore="onExplore"
      @learn-more="onLearnMore"
    />

    <section class="py-16 px-6 max-w-[1200px] mx-auto">
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-8 max-w-[1000px] mx-auto">
        <ScrollReveal
          v-for="(feature, i) in features"
          :key="feature.title"
          :delay="i * 100"
        >
          <FeatureCard v-bind="feature" />
        </ScrollReveal>
      </div>
    </section>

    <section class="py-12 px-6 mx-4 lg:mx-auto max-w-[1200px]">
      <div class="glass-card rounded-2xl py-12 px-6">
        <div
          v-if="statsLoadError"
          role="alert"
          class="text-center text-sm font-medium text-danger"
        >
          {{ statsLoadError }}
        </div>
        <div v-else class="grid grid-cols-1 sm:grid-cols-3 gap-8 text-center">
          <ScrollReveal
            v-for="(stat, i) in stats"
            :key="stat.label"
            :delay="i * 150"
          >
            <div class="py-4">
              <CountUp
                :target="stat.target"
                :duration="2000"
                class="text-5xl font-extrabold gradient-text tabular-nums"
              />
              <p class="mt-3 text-base text-text-secondary">{{ stat.label }}</p>
            </div>
          </ScrollReveal>
        </div>
      </div>
    </section>

    <footer class="py-8 px-6 border-t border-border-light mt-8">
      <div class="max-w-[1200px] mx-auto flex justify-between items-center flex-wrap gap-4 text-sm text-text-muted max-sm:flex-col max-sm:text-center">
        <p>{{ t('home.landing.footer.copyright') }}</p>
        <div class="flex gap-8">
          <RouterLink to="/about" class="text-text-muted hover:text-text-primary transition-colors duration-fast no-underline">{{ t('home.landing.footer.about') }}</RouterLink>
          <RouterLink to="/privacy" class="text-text-muted hover:text-text-primary transition-colors duration-fast no-underline">{{ t('home.landing.footer.privacy') }}</RouterLink>
          <RouterLink to="/terms" class="text-text-muted hover:text-text-primary transition-colors duration-fast no-underline">{{ t('home.landing.footer.terms') }}</RouterLink>
        </div>
      </div>
    </footer>
  </div>
</template>
