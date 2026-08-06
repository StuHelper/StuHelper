<template>
  <section class="mb-6 rounded-xl bg-bg-card p-5 shadow-card">
    <header class="mb-5 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
      <div class="flex min-w-0 items-center gap-4">
        <span class="grid size-14 shrink-0 place-items-center overflow-hidden rounded-full bg-gradient-to-br from-primary to-accent p-[3px]">
          <span class="grid size-full place-items-center overflow-hidden rounded-full bg-bg-card">
            <img v-if="user?.avatar" :src="user.avatar" :alt="displayName" class="size-full object-cover" />
            <User v-else class="size-7 text-text-muted" aria-hidden="true" />
          </span>
        </span>
        <div class="min-w-0">
          <h2 class="m-0 truncate text-base font-bold text-text-primary">{{ displayName }}</h2>
          <p class="m-0 mt-0.5 truncate text-sm text-text-muted">{{ user?.email ?? '' }}</p>
        </div>
      </div>
      <router-link
        to="/account/profile"
        class="inline-flex h-9 shrink-0 items-center justify-center gap-2 rounded-md border border-border bg-bg-base px-3 text-sm font-medium text-text-secondary no-underline transition-colors duration-fast hover:border-primary/40 hover:text-primary"
      >
        <User class="size-4" aria-hidden="true" />
        {{ t('user.identityHome.accountProfile.title') }}
      </router-link>
    </header>

    <div class="grid gap-4 sm:grid-cols-3">
      <StatusCard
        :icon="GraduationCap"
        :title="t('user.verification.student.title')"
        :complete="verificationStore.studentVerified"
        :complete-label="t('user.verification.student.verified')"
        :incomplete-label="t('user.verification.student.unverified')"
        to="/user/student-verification"
      />
      <StatusCard
        :icon="Phone"
        :title="t('user.verification.phone.title')"
        :complete="verificationStore.phoneVerified"
        :complete-label="t('user.verification.phone.verified')"
        :incomplete-label="t('user.verification.phone.unverified')"
        to="/user/phone-binding"
      />
      <StatusCard
        :icon="Bot"
        :title="t('user.verification.qq.title')"
        :complete="verificationStore.qqBound"
        :complete-label="t('user.verification.qq.bound')"
        :incomplete-label="t('user.verification.qq.unbound')"
        to="/user/qq-binding"
      />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, markRaw, onMounted, type PropType } from 'vue'
import { useI18n } from 'vue-i18n'
import { Bot, GraduationCap, Phone, User, type LucideIcon } from 'lucide-vue-next'

import { useAuthStore } from '@/stores/auth'
import { useVerificationStore } from '@/stores/verification'

const StatusCard = defineComponent({
  name: 'VerificationStatusCard',
  props: {
    icon: { type: [Object, Function] as PropType<LucideIcon>, required: true },
    title: { type: String, required: true },
    complete: { type: Boolean, required: true },
    completeLabel: { type: String, required: true },
    incompleteLabel: { type: String, required: true },
    to: { type: String, required: true },
  },
  setup(props) {
    const icon = markRaw(props.icon)
    return () => h(
      'a',
      {
        href: props.to,
        class: 'group rounded-lg border border-border bg-bg-base/50 p-4 no-underline transition-all duration-fast hover:border-primary/40 hover:bg-bg-base',
      },
      [
        h('div', { class: 'flex items-center gap-3' }, [
          h('span', {
            class: props.complete
              ? 'grid size-10 place-items-center rounded-lg bg-success/10 text-success'
              : 'grid size-10 place-items-center rounded-lg bg-bg-card text-text-muted',
          }, [h(icon, { class: 'size-5', 'aria-hidden': 'true' })]),
          h('span', { class: 'min-w-0' }, [
            h('span', { class: 'block truncate text-sm font-semibold text-text-primary' }, props.title),
            h('span', {
              class: props.complete
                ? 'mt-1 inline-flex rounded-full bg-success/10 px-2 py-0.5 text-xs font-medium text-success'
                : 'mt-1 inline-flex rounded-full bg-bg-card px-2 py-0.5 text-xs font-medium text-text-muted',
            }, props.complete ? props.completeLabel : props.incompleteLabel),
          ]),
        ]),
      ],
    )
  },
})

const { t } = useI18n()
const authStore = useAuthStore()
const verificationStore = useVerificationStore()
const user = computed(() => authStore.user)
const displayName = computed(() => user.value?.displayName ?? user.value?.name ?? '')

onMounted(() => {
  if (authStore.isAuthenticated) {
    void verificationStore.fetchStatus().catch((error) => {
      if (import.meta.env.DEV) console.warn('[ProfileSection] failed to load account gates', error)
    })
  }
})
</script>
