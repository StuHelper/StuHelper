<template>
  <main class="mx-auto max-w-[960px] p-6 animate-fade-in max-sm:p-4">
    <header class="mb-6 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <div class="min-w-0">
        <p class="m-0 text-xs font-semibold uppercase text-primary">
          {{ t('user.accountSecurity.eyebrow') }}
        </p>
        <h1 class="m-0 mt-2 text-2xl font-bold text-text-primary">
          {{ t('user.accountSecurity.title') }}
        </h1>
        <p class="m-0 mt-2 max-w-[720px] text-sm leading-relaxed text-text-secondary">
          {{ t('user.accountSecurity.subtitle') }}
        </p>
      </div>

      <router-link
        to="/identity"
        class="inline-flex h-10 shrink-0 items-center justify-center gap-2 rounded-md border border-border bg-bg-card px-3 text-sm font-medium text-text-secondary no-underline transition-colors duration-fast hover:border-primary/40 hover:text-primary"
      >
        <ArrowLeft class="size-4" aria-hidden="true" />
        {{ t('user.accountSecurity.backToIdentity') }}
      </router-link>
    </header>

    <section class="grid gap-4 lg:grid-cols-[1.1fr_0.9fr]">
      <section class="rounded-lg border border-border bg-bg-card p-5 shadow-sm">
        <div class="flex items-start gap-4">
          <span class="grid size-12 shrink-0 place-items-center rounded-lg bg-primary-alpha text-primary">
            <UserRound class="size-6" aria-hidden="true" />
          </span>
          <div class="min-w-0">
            <h2 class="m-0 text-lg font-semibold text-text-primary">
              {{ displayName }}
            </h2>
            <p class="m-0 mt-1 text-sm text-text-secondary">
              {{ user?.email || t('user.accountSecurity.emailMissing') }}
            </p>
          </div>
        </div>

        <dl class="m-0 mt-5 divide-y divide-border border-y border-border">
          <div class="py-3">
            <dt class="text-xs font-medium uppercase text-text-muted">
              {{ t('user.accountSecurity.fields.accountId') }}
            </dt>
            <dd class="m-0 mt-1 break-all text-sm text-text-primary">
              {{ user?.id || '-' }}
            </dd>
          </div>
          <div class="py-3">
            <dt class="text-xs font-medium uppercase text-text-muted">
              {{ t('user.accountSecurity.fields.username') }}
            </dt>
            <dd class="m-0 mt-1 break-all text-sm text-text-primary">
              {{ user?.name || '-' }}
            </dd>
          </div>
        </dl>
      </section>

      <section class="rounded-lg border border-border bg-bg-card p-5 shadow-sm">
        <header class="flex items-start gap-3">
          <span class="grid size-10 shrink-0 place-items-center rounded-lg bg-success/10 text-success">
            <ShieldCheck class="size-5" aria-hidden="true" />
          </span>
          <div class="min-w-0">
            <h2 class="m-0 text-base font-semibold text-text-primary">
              {{ t('user.accountSecurity.session.title') }}
            </h2>
            <p class="m-0 mt-1 text-sm leading-relaxed text-text-secondary">
              {{ t('user.accountSecurity.session.description') }}
            </p>
          </div>
        </header>

        <div class="mt-4 rounded-md border border-success/20 bg-success/10 p-3 text-sm font-medium text-success">
          {{ t('user.accountSecurity.session.active') }}
        </div>

        <button
          type="button"
          class="mt-4 inline-flex h-10 items-center justify-center gap-2 rounded-md border border-danger/30 bg-bg-card px-3 text-sm font-semibold text-danger transition-colors duration-fast hover:bg-danger/10"
          :disabled="loggingOut"
          @click="logoutCurrentSession"
        >
          <LogOut class="size-4" aria-hidden="true" />
          {{ loggingOut ? t('user.accountSecurity.session.loggingOut') : t('user.accountSecurity.session.logout') }}
        </button>
      </section>
    </section>

    <section
      class="mt-4 grid gap-4 md:grid-cols-2"
      :aria-label="t('user.accountSecurity.actionsLabel')"
    >
      <a
        v-if="accountSettingsUrl"
        :href="accountSettingsUrl"
        target="_blank"
        rel="noopener noreferrer"
        class="group rounded-lg border border-border bg-bg-card p-5 text-left no-underline shadow-sm transition-all duration-fast hover:-translate-y-0.5 hover:border-primary/40 hover:shadow-md"
      >
        <AccountAction
          :icon="KeyRound"
          :title="t('user.accountSecurity.provider.title')"
          :description="t('user.accountSecurity.provider.description')"
          :action="t('user.accountSecurity.provider.open')"
          external
        />
      </a>

      <router-link
        to="/user/phone-binding"
        class="group rounded-lg border border-border bg-bg-card p-5 text-left no-underline shadow-sm transition-all duration-fast hover:-translate-y-0.5 hover:border-primary/40 hover:shadow-md"
      >
        <AccountAction
          :icon="Phone"
          :title="t('user.accountSecurity.phone.title')"
          :description="t('user.accountSecurity.phone.description')"
          :action="t('user.identityHome.open')"
        />
      </router-link>

      <router-link
        to="/user/authorized-apps"
        class="group rounded-lg border border-border bg-bg-card p-5 text-left no-underline shadow-sm transition-all duration-fast hover:-translate-y-0.5 hover:border-primary/40 hover:shadow-md"
      >
        <AccountAction
          :icon="ShieldCheck"
          :title="t('user.accountSecurity.authorizedApps.title')"
          :description="t('user.accountSecurity.authorizedApps.description')"
          :action="t('user.identityHome.open')"
        />
      </router-link>

      <router-link
        to="/user/identity-verification"
        class="group rounded-lg border border-border bg-bg-card p-5 text-left no-underline shadow-sm transition-all duration-fast hover:-translate-y-0.5 hover:border-primary/40 hover:shadow-md"
      >
        <AccountAction
          :icon="UserRoundCheck"
          :title="t('user.accountSecurity.identity.title')"
          :description="t('user.accountSecurity.identity.description')"
          :action="t('user.identityHome.open')"
        />
      </router-link>
    </section>
  </main>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, markRaw, ref, type PropType } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  ArrowLeft,
  ArrowUpRight,
  KeyRound,
  LogOut,
  Phone,
  ShieldCheck,
  UserRound,
  UserRoundCheck,
  type LucideIcon,
} from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import { useToast } from '@/composables/useToast'

const AccountAction = defineComponent({
  name: 'AccountAction',
  props: {
    icon: {
      type: Object as PropType<LucideIcon>,
      required: true,
    },
    title: {
      type: String,
      required: true,
    },
    description: {
      type: String,
      required: true,
    },
    action: {
      type: String,
      required: true,
    },
    external: {
      type: Boolean,
      default: false,
    },
  },
  setup(props) {
    const icon = markRaw(props.icon)
    return () =>
      h('div', { class: 'flex items-start gap-4' }, [
        h(
          'span',
          {
            class:
              'grid size-11 shrink-0 place-items-center rounded-lg bg-primary-alpha text-primary',
          },
          [h(icon, { class: 'size-5', 'aria-hidden': 'true' })],
        ),
        h('span', { class: 'min-w-0' }, [
          h('span', { class: 'block text-base font-semibold text-text-primary' }, props.title),
          h(
            'span',
            { class: 'mt-1 block text-sm leading-relaxed text-text-secondary' },
            props.description,
          ),
          h(
            'span',
            {
              class:
                'mt-3 inline-flex items-center gap-1 text-sm font-medium text-primary',
            },
            [
              props.action,
              props.external
                ? h(ArrowUpRight, { class: 'size-4', 'aria-hidden': 'true' })
                : null,
            ],
          ),
        ]),
      ])
  },
})

const { t } = useI18n()
const router = useRouter()
const authStore = useAuthStore()
const toast = useToast()
const loggingOut = ref(false)

const user = computed(() => authStore.user)
const displayName = computed(
  () => user.value?.displayName || user.value?.name || t('nav.user'),
)
const accountSettingsUrl = computed(() => user.value?.accountSettingsUrl ?? '')

async function logoutCurrentSession() {
  if (loggingOut.value) return
  loggingOut.value = true
  try {
    const result = await authStore.logout()
    if (result.ok) {
      await router.push({ path: '/login', query: { redirect: '/identity' } })
      return
    }
    toast.error(
      result.reason === 'server'
        ? t('nav.logoutServerError')
        : t('nav.logoutNetworkError'),
    )
  } finally {
    loggingOut.value = false
  }
}
</script>
