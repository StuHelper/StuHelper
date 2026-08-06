<template>
  <section class="mx-auto max-w-[1040px] p-6 animate-fade-in max-sm:p-4">
    <header class="mb-6 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <div class="min-w-0">
        <p class="m-0 text-xs font-semibold uppercase text-primary">
          {{ t('user.accountProfile.eyebrow') }}
        </p>
        <h1 class="m-0 mt-2 text-2xl font-bold text-text-primary">
          {{ t('user.accountProfile.title') }}
        </h1>
        <p class="m-0 mt-2 max-w-[720px] text-sm leading-relaxed text-text-secondary">
          {{ t('user.accountProfile.subtitle') }}
        </p>
      </div>

      <router-link
        to="/identity"
        class="inline-flex h-10 shrink-0 items-center justify-center gap-2 rounded-md border border-border bg-bg-card px-3 text-sm font-medium text-text-secondary no-underline transition-colors duration-fast hover:border-primary/40 hover:text-primary"
      >
        <ArrowLeft class="size-4" aria-hidden="true" />
        {{ t('user.accountProfile.backToIdentity') }}
      </router-link>
    </header>

    <div
      v-if="statusPending"
      data-account-profile-status-loading
      class="mb-4 flex items-center gap-3 rounded-lg border border-border bg-bg-card px-4 py-3 text-sm text-text-secondary"
      role="status"
      aria-live="polite"
    >
      <span
        class="size-2 shrink-0 animate-pulse rounded-full bg-primary"
        aria-hidden="true"
      />
      {{ t('common.actions.loading') }}
    </div>

    <div
      v-else-if="statusLoadState === 'error'"
      data-account-profile-status-error
      class="mb-4 flex flex-col gap-3 rounded-lg border border-danger/30 bg-danger/5 px-4 py-3 text-sm text-danger sm:flex-row sm:items-center sm:justify-between"
      role="alert"
    >
      <span>{{ t('common.loadFailed') }}</span>
      <button
        type="button"
        data-account-profile-status-retry
        class="inline-flex h-9 shrink-0 items-center justify-center rounded-md border border-danger/30 bg-bg-card px-3 font-medium text-danger transition-colors duration-fast hover:bg-danger/10"
        @click="loadStatus"
      >
        {{ t('common.actions.retry') }}
      </button>
    </div>

    <section class="grid gap-4 lg:grid-cols-[1.05fr_0.95fr]">
      <section class="rounded-lg border border-border bg-bg-card p-5 shadow-sm">
        <div class="flex items-start gap-4">
          <span class="grid size-14 shrink-0 place-items-center overflow-hidden rounded-full bg-primary-alpha text-primary">
            <img
              v-if="user?.avatar"
              :src="user.avatar"
              :alt="displayName"
              class="size-full object-cover"
            />
            <UserRound v-else class="size-7" aria-hidden="true" />
          </span>
          <div class="min-w-0">
            <h2 class="m-0 text-lg font-semibold text-text-primary">
              {{ displayName }}
            </h2>
            <p class="m-0 mt-1 text-sm text-text-secondary">
              {{ user?.email || t('user.accountProfile.missing.email') }}
            </p>
          </div>
        </div>

        <dl class="m-0 mt-5 divide-y divide-border border-y border-border">
          <ProfileField
            :label="t('user.accountProfile.fields.displayName')"
            :value="displayName"
          />
          <ProfileField
            :label="t('user.accountProfile.fields.username')"
            :value="user?.name || t('user.accountProfile.missing.value')"
          />
          <ProfileField
            :label="t('user.accountProfile.fields.accountId')"
            :value="user?.id || t('user.accountProfile.missing.value')"
          />
        </dl>
      </section>

      <section class="rounded-lg border border-border bg-bg-card p-5 shadow-sm">
        <header class="flex items-start gap-3">
          <span class="grid size-10 shrink-0 place-items-center rounded-lg bg-success/10 text-success">
            <CheckCircle2 class="size-5" aria-hidden="true" />
          </span>
          <div class="min-w-0">
            <h2 class="m-0 text-base font-semibold text-text-primary">
              {{ t('user.accountProfile.contact.title') }}
            </h2>
            <p class="m-0 mt-1 text-sm leading-relaxed text-text-secondary">
              {{ t('user.accountProfile.contact.description') }}
            </p>
          </div>
        </header>

        <dl class="m-0 mt-4 divide-y divide-border border-y border-border">
          <ProfileField
            :label="t('user.accountProfile.fields.email')"
            :value="user?.email || t('user.accountProfile.missing.email')"
            :badge="emailBadge"
          />
          <template v-if="statusReady">
            <ProfileField
              :label="t('user.accountProfile.fields.phone')"
              :value="phoneDisplay"
              :badge="phoneBadge"
            />
            <ProfileField
              :label="t('user.accountProfile.fields.qq')"
              :value="qqBindingLabel"
              :badge="qqBadge"
            />
          </template>
        </dl>
      </section>
    </section>

    <section
      v-if="statusReady"
      class="mt-4 grid gap-4 md:grid-cols-2"
      :aria-label="t('user.accountProfile.verification.actionsLabel')"
    >
      <router-link
        v-for="item in verificationItems"
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
            <span
              class="mt-2 inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium"
              :class="item.badgeClass"
            >
              {{ item.status }}
            </span>
            <span class="mt-2 block text-sm leading-relaxed text-text-secondary">
              {{ item.description }}
            </span>
          </span>
        </div>
      </router-link>
    </section>

    <section
      v-else-if="statusPending"
      class="mt-4 grid gap-4 md:grid-cols-2"
      aria-hidden="true"
    >
      <div
        v-for="index in 2"
        :key="index"
        class="h-36 animate-pulse rounded-lg border border-border bg-bg-card"
      />
    </section>

    <section class="mt-4 rounded-lg border border-border bg-bg-card p-5 shadow-sm">
      <header class="flex items-start gap-3">
        <span class="grid size-10 shrink-0 place-items-center rounded-lg bg-primary-alpha text-primary">
          <ShieldCheck class="size-5" aria-hidden="true" />
        </span>
        <div class="min-w-0">
          <h2 class="m-0 text-base font-semibold text-text-primary">
            {{ t('user.accountProfile.disclosure.title') }}
          </h2>
          <p class="m-0 mt-1 text-sm leading-relaxed text-text-secondary">
            {{ t('user.accountProfile.disclosure.description') }}
          </p>
        </div>
      </header>

      <dl class="m-0 mt-4 grid gap-3 md:grid-cols-2">
        <DisclosureField
          v-for="field in disclosureFields"
          :key="field.key"
          :label="field.label"
          :description="field.description"
          :badge="field.badge"
          :badge-class="field.badgeClass"
        />
      </dl>
    </section>
  </section>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, markRaw, onMounted, ref, type PropType } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  ArrowLeft,
  CheckCircle2,
  GraduationCap,
  Phone,
  ShieldCheck,
  UserRound,
  type LucideIcon,
} from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import { useVerificationStore } from '@/stores/verification'

type BadgeVariant = 'success' | 'muted'

interface BadgeInfo {
  label: string
  className: string
}

interface VerificationItem {
  to: string
  icon: LucideIcon
  title: string
  description: string
  status: string
  badgeClass: string
}

interface DisclosureFieldInfo {
  key: string
  label: string
  description: string
  badge: string
  badgeClass: string
}

type StatusLoadState = 'idle' | 'loading' | 'ready' | 'error'

const ProfileField = defineComponent({
  name: 'ProfileField',
  props: {
    label: {
      type: String,
      required: true,
    },
    value: {
      type: String,
      required: true,
    },
    badge: {
      type: Object as PropType<BadgeInfo | null>,
      default: null,
    },
  },
  setup(props) {
    return () =>
      h('div', { class: 'grid gap-1 py-3 sm:grid-cols-[140px_1fr] sm:gap-4' }, [
        h('dt', { class: 'text-xs font-medium uppercase text-text-muted' }, props.label),
        h('dd', { class: 'm-0 flex min-w-0 flex-wrap items-center gap-2 text-sm text-text-primary' }, [
          h('span', { class: 'break-all' }, props.value),
          props.badge
            ? h(
                'span',
                {
                  class: [
                    'inline-flex shrink-0 items-center rounded-full px-2 py-0.5 text-xs font-medium',
                    props.badge.className,
                  ],
                },
                props.badge.label,
              )
            : null,
        ]),
      ])
  },
})

const DisclosureField = defineComponent({
  name: 'DisclosureField',
  props: {
    label: {
      type: String,
      required: true,
    },
    description: {
      type: String,
      required: true,
    },
    badge: {
      type: String,
      required: true,
    },
    badgeClass: {
      type: String,
      required: true,
    },
  },
  setup(props) {
    return () =>
      h('div', { class: 'rounded-md border border-border p-3' }, [
        h('dt', { class: 'flex items-center justify-between gap-3 text-sm font-semibold text-text-primary' }, [
          h('span', props.label),
          h(
            'span',
            {
              class: [
                'inline-flex shrink-0 items-center rounded-full px-2 py-0.5 text-xs font-medium',
                props.badgeClass,
              ],
            },
            props.badge,
          ),
        ]),
        h('dd', { class: 'm-0 mt-2 text-sm leading-relaxed text-text-secondary' }, props.description),
      ])
  },
})

const { t } = useI18n()
const authStore = useAuthStore()
const verificationStore = useVerificationStore()
const statusLoadState = ref<StatusLoadState>('idle')

const user = computed(() => authStore.user)
const surface = computed(() => verificationStore.userSurface)
const phoneStatus = computed(() => verificationStore.phoneStatus)
const qqBinding = computed(() => verificationStore.qqBinding)
const statusPending = computed(
  () =>
    authStore.isAuthenticated &&
    (statusLoadState.value === 'idle' || statusLoadState.value === 'loading'),
)
const statusReady = computed(() => statusLoadState.value === 'ready')

const displayName = computed(
  () => user.value?.displayName || user.value?.name || t('nav.user'),
)

const successClass = 'bg-success/10 text-success'
const mutedClass = 'bg-bg-elevated text-text-muted'

function badgeClass(variant: BadgeVariant): string {
  if (variant === 'success') return successClass
  return mutedClass
}

function statusBadge(label: string, variant: BadgeVariant): BadgeInfo {
  return { label, className: badgeClass(variant) }
}

const emailBadge = computed(() =>
  user.value?.email
    ? statusBadge(t('user.accountProfile.status.available'), 'success')
    : statusBadge(t('user.accountProfile.status.missing'), 'muted'),
)
const phoneBadge = computed(() =>
  verificationStore.phoneVerified
    ? statusBadge(t('user.accountProfile.status.verified'), 'success')
    : statusBadge(t('user.accountProfile.status.unverified'), 'muted'),
)
const phoneDisplay = computed(
  () => surface.value?.phone || phoneStatus.value?.maskedPhone || t('user.accountProfile.missing.phone'),
)
const qqBadge = computed(() =>
  qqBinding.value
    ? statusBadge(t('user.accountProfile.status.bound'), 'success')
    : statusBadge(t('user.accountProfile.status.unbound'), 'muted'),
)
const qqBindingLabel = computed(() => {
  if (!qqBinding.value) return t('user.accountProfile.missing.qq')
  return qqBinding.value.qqID
})

const studentStatus = computed(() => {
  if (verificationStore.studentVerified) {
    return { label: t('user.verification.student.verified'), variant: 'success' as const }
  }
  return { label: t('user.verification.student.unverified'), variant: 'muted' as const }
})

const verificationItems = computed<VerificationItem[]>(() => [
  {
    to: '/user/student-verification',
    icon: markRaw(GraduationCap),
    title: t('user.accountProfile.verification.student.title'),
    description: t('user.accountProfile.verification.student.description'),
    status: studentStatus.value.label,
    badgeClass: badgeClass(studentStatus.value.variant),
  },
  {
    to: '/user/phone-binding',
    icon: markRaw(Phone),
    title: t('user.accountProfile.verification.phone.title'),
    description: t('user.accountProfile.verification.phone.description'),
    status: phoneBadge.value.label,
    badgeClass: phoneBadge.value.className,
  },
])

const disclosureFields = computed<DisclosureFieldInfo[]>(() => {
  const reliableFields: DisclosureFieldInfo[] = [
    {
      key: 'profile',
      label: t('user.accountProfile.disclosure.profile.title'),
      description: t('user.accountProfile.disclosure.profile.description'),
      badge: t('user.accountProfile.status.available'),
      badgeClass: successClass,
    },
    {
      key: 'email',
      label: t('user.accountProfile.disclosure.email.title'),
      description: t('user.accountProfile.disclosure.email.description'),
      badge: emailBadge.value.label,
      badgeClass: emailBadge.value.className,
    },
  ]

  if (!statusReady.value) return reliableFields

  return [
    ...reliableFields,
    {
      key: 'phone',
      label: t('user.accountProfile.disclosure.phone.title'),
      description: t('user.accountProfile.disclosure.phone.description'),
      badge: phoneBadge.value.label,
      badgeClass: phoneBadge.value.className,
    },
    {
      key: 'student',
      label: t('user.accountProfile.disclosure.student.title'),
      description: t('user.accountProfile.disclosure.student.description'),
      badge: studentStatus.value.label,
      badgeClass: badgeClass(studentStatus.value.variant),
    },
  ]
})

async function loadStatus() {
  if (!authStore.isAuthenticated || statusLoadState.value === 'loading') return

  statusLoadState.value = 'loading'
  try {
    await verificationStore.fetchStatus()
    statusLoadState.value = 'ready'
  } catch (error) {
    statusLoadState.value = 'error'
    if (import.meta.env.DEV) {
      console.warn('[AccountProfilePage] failed to fetch verification status', error)
    }
  }
}

onMounted(() => {
  void loadStatus()
})
</script>
