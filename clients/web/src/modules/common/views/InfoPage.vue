<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'

type PageKey = 'about' | 'privacy' | 'terms'
type SectionKey = 'currentCapabilities' | 'contentPrinciples' | 'roadmap' | 'collection' | 'usage' | 'control' | 'responsibility' | 'access' | 'changes'

const props = defineProps<{
  pageKey: PageKey
}>()

const { t } = useI18n()

const pageSections: Record<PageKey, SectionKey[]> = {
  about: ['currentCapabilities', 'contentPrinciples', 'roadmap'],
  privacy: ['collection', 'usage', 'control'],
  terms: ['responsibility', 'access', 'changes'],
}

const pageContent = computed(() => {
  const baseKey = `common.infoPages.${props.pageKey}`
  return {
    eyebrow: t(`${baseKey}.eyebrow`),
    title: t(`${baseKey}.title`),
    intro: t(`${baseKey}.intro`),
    sections: pageSections[props.pageKey].map((sectionKey) => ({
      heading: t(`${baseKey}.sections.${sectionKey}.heading`),
      body: t(`${baseKey}.sections.${sectionKey}.body`),
    })),
  }
})
</script>

<template>
  <main
    class="min-h-screen bg-[radial-gradient(circle_at_top,_rgba(14,165,233,0.14),_transparent_38%),linear-gradient(180deg,_#f8fafc_0%,_#ffffff_100%)] px-6 py-16"
  >
    <section
      class="mx-auto max-w-4xl overflow-hidden rounded-[32px] border border-slate-200/80 bg-white/90 shadow-[0_24px_80px_rgba(15,23,42,0.08)] backdrop-blur"
    >
      <div class="border-b border-slate-200/80 px-8 py-10">
        <p
          class="mb-3 text-xs font-semibold uppercase tracking-[0.28em] text-cyan-600"
        >
          {{ pageContent.eyebrow }}
        </p>
        <h1
          class="m-0 text-4xl font-black tracking-tight text-slate-900"
        >
          {{ pageContent.title }}
        </h1>
        <p class="mt-4 max-w-3xl text-base leading-7 text-slate-600">
          {{ pageContent.intro }}
        </p>
      </div>

      <div class="space-y-8 px-8 py-10">
        <article
          v-for="section in pageContent.sections"
          :key="section.heading"
          class="rounded-2xl border border-slate-200 bg-slate-50/70 p-6"
        >
          <h2 class="m-0 text-xl font-bold text-slate-900">
            {{ section.heading }}
          </h2>
          <p class="mt-3 mb-0 text-sm leading-7 text-slate-600">
            {{ section.body }}
          </p>
        </article>

        <div class="flex flex-wrap gap-3">
          <RouterLink
            to="/"
            class="rounded-full bg-slate-900 px-5 py-2.5 text-sm font-medium text-white no-underline transition-opacity hover:opacity-90"
          >
            {{ t('common.infoPages.actions.home') }}
          </RouterLink>
          <RouterLink
            to="/course"
            class="rounded-full border border-slate-300 px-5 py-2.5 text-sm font-medium text-slate-700 no-underline transition-colors hover:border-cyan-500 hover:text-cyan-600"
          >
            {{ t('common.infoPages.actions.learningCenter') }}
          </RouterLink>
        </div>
      </div>
    </section>
  </main>
</template>
