<template>
  <main class="mx-auto max-w-[1120px] p-6 animate-fade-in max-sm:p-4">
    <header class="mb-6">
      <p class="m-0 text-xs font-semibold uppercase text-primary">
        {{ t('user.identityHome.eyebrow') }}
      </p>
      <h1 class="m-0 mt-2 text-2xl font-bold text-text-primary">
        {{ t('user.identityHome.title') }}
      </h1>
      <p class="m-0 mt-2 max-w-[720px] text-sm leading-relaxed text-text-secondary">
        {{ t('user.identityHome.subtitle') }}
      </p>
    </header>

    <ProfileSection />

    <section
      class="grid gap-4 md:grid-cols-2 xl:grid-cols-3"
      :aria-label="t('user.identityHome.sectionsLabel')"
    >
      <router-link
        v-for="item in portalItems"
        :key="item.to"
        :to="item.to"
        class="group rounded-lg border border-border bg-bg-card p-5 text-left no-underline shadow-sm transition-all duration-fast hover:-translate-y-0.5 hover:border-primary/40 hover:shadow-md"
      >
        <div class="flex items-start gap-4">
          <span class="grid size-11 shrink-0 place-items-center rounded-lg bg-primary-alpha text-primary">
            <component :is="item.icon" class="size-5" aria-hidden="true" />
          </span>
          <span class="min-w-0">
            <span class="block text-base font-semibold text-text-primary">
              {{ item.title }}
            </span>
            <span class="mt-1 block text-sm leading-relaxed text-text-secondary">
              {{ item.description }}
            </span>
            <span class="mt-3 inline-flex items-center gap-1 text-sm font-medium text-primary">
              {{ t('user.identityHome.open') }}
              <ArrowRight class="size-4 transition-transform duration-fast group-hover:translate-x-0.5" aria-hidden="true" />
            </span>
          </span>
        </div>
      </router-link>
    </section>
  </main>
</template>

<script setup lang="ts">
import { computed, markRaw } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  ArrowRight,
  Bot,
  FileText,
  GraduationCap,
  KeyRound,
  Phone,
  ShieldCheck,
  UserRoundCheck,
  type LucideIcon,
} from 'lucide-vue-next'
import ProfileSection from './ProfileSection.vue'

interface PortalItem {
  to: string
  icon: LucideIcon
  title: string
  description: string
}

const { t } = useI18n()

const portalItems = computed<PortalItem[]>(() => [
  {
    to: '/user/authorized-apps',
    icon: markRaw(ShieldCheck),
    title: t('user.identityHome.authorizedApps.title'),
    description: t('user.identityHome.authorizedApps.description'),
  },
  {
    to: '/user/identity-verification',
    icon: markRaw(UserRoundCheck),
    title: t('user.identityHome.identityVerification.title'),
    description: t('user.identityHome.identityVerification.description'),
  },
  {
    to: '/user/student-verification',
    icon: markRaw(GraduationCap),
    title: t('user.identityHome.studentVerification.title'),
    description: t('user.identityHome.studentVerification.description'),
  },
  {
    to: '/user/phone-binding',
    icon: markRaw(Phone),
    title: t('user.identityHome.phoneBinding.title'),
    description: t('user.identityHome.phoneBinding.description'),
  },
  {
    to: '/user/qq-binding',
    icon: markRaw(Bot),
    title: t('user.identityHome.qqBinding.title'),
    description: t('user.identityHome.qqBinding.description'),
  },
  {
    to: '/user/academic-info',
    icon: markRaw(FileText),
    title: t('user.identityHome.academicInfo.title'),
    description: t('user.identityHome.academicInfo.description'),
  },
  {
    to: '/developers/apps',
    icon: markRaw(KeyRound),
    title: t('user.identityHome.developerApps.title'),
    description: t('user.identityHome.developerApps.description'),
  },
])
</script>
