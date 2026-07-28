<template>
  <div class="flex flex-col gap-4">
    <div v-if="loading" class="grid gap-4">
      <SkeletonCard v-for="i in 2" :key="i" variant="course" />
    </div>

    <EmptyState
      v-else-if="errorMessage"
      :title="t('common.loadFailed')"
      :description="errorMessage"
    >
      <template #action>
        <button
          class="inline-flex items-center justify-center rounded-sm bg-text-primary px-4 py-2 text-sm font-medium text-bg-base transition-colors duration-fast hover:bg-accent hover:text-white"
          type="button"
          @click="loadConsents"
        >
          {{ t('common.actions.retry') }}
        </button>
      </template>
    </EmptyState>

    <EmptyState
      v-else-if="apps.length === 0"
      :title="t('user.authorizedApps.empty')"
      :description="t('user.authorizedApps.emptyDesc')"
    >
      <template #icon>
        <ShieldCheck class="h-full w-full" aria-hidden="true" />
      </template>
    </EmptyState>

    <template v-else>
      <article
        v-for="authorizedApp in apps"
        :key="authorizedApp.app.id"
        class="rounded-lg border border-border bg-bg-card p-5 shadow-sm"
      >
        <header class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary-alpha text-primary">
                <ShieldCheck class="h-5 w-5" aria-hidden="true" />
              </div>
              <div class="min-w-0">
                <h3 class="m-0 truncate text-base font-semibold text-text-primary">
                  {{ authorizedApp.app.displayName }}
                </h3>
                <p class="m-0 mt-0.5 break-all text-xs text-text-muted">
                  {{ authorizedApp.app.clientID }}
                </p>
              </div>
            </div>

            <p
              v-if="authorizedApp.app.description"
              class="mb-0 mt-3 text-sm leading-relaxed text-text-secondary"
            >
              {{ authorizedApp.app.description }}
            </p>

            <div class="mt-3 flex flex-wrap items-center gap-3 text-sm">
              <a
                class="authorized-app-link"
                :href="authorizedApp.app.homepageURL"
                target="_blank"
                rel="noreferrer"
              >
                {{ t('user.authorizedApps.homepage') }}
                <ExternalLink class="h-3.5 w-3.5" aria-hidden="true" />
              </a>
              <a
                class="authorized-app-link"
                :href="authorizedApp.app.privacyPolicyURL"
                target="_blank"
                rel="noreferrer"
              >
                {{ t('user.authorizedApps.privacy') }}
                <ExternalLink class="h-3.5 w-3.5" aria-hidden="true" />
              </a>
              <span class="text-xs text-text-muted">
                {{ t('user.authorizedApps.scopeCount', { count: authorizedApp.scopes.length }) }}
              </span>
            </div>
          </div>

          <button
            class="inline-flex shrink-0 items-center justify-center gap-2 rounded-md border border-danger/40 bg-transparent px-3 py-2 text-sm font-medium text-danger transition-colors duration-fast hover:bg-danger/10 disabled:cursor-not-allowed disabled:opacity-50"
            type="button"
            :disabled="Boolean(revokingKey)"
            @click="openRevokeAppDialog(authorizedApp)"
          >
            <Trash2 class="h-4 w-4" aria-hidden="true" />
            {{ revokingKey === appRevokeKey(authorizedApp.app.id) ? t('user.authorizedApps.revoking') : t('user.authorizedApps.revokeApp') }}
          </button>
        </header>

        <section class="mt-5 border-t border-border pt-4">
          <ul class="m-0 grid list-none gap-3 p-0">
            <li
              v-for="scope in authorizedApp.scopes"
              :key="`${authorizedApp.app.id}:${scope.scope}`"
              class="grid gap-3 rounded-md bg-bg-elevated p-4 sm:grid-cols-[1fr_auto] sm:items-start"
            >
              <div class="min-w-0">
                <div class="flex flex-wrap items-center gap-2">
                  <h4 class="m-0 text-sm font-semibold text-text-primary">
                    {{ scope.displayName }}
                  </h4>
                  <span class="rounded-full px-2 py-0.5 text-xs font-medium" :class="sensitivityClass(scope.sensitivity)">
                    {{ sensitivityLabel(scope.sensitivity) }}
                  </span>
                </div>
                <p class="mb-0 mt-1 break-all text-xs text-text-muted">
                  {{ scope.scope }}
                </p>
                <p v-if="scope.fields.length" class="mb-0 mt-2 text-sm text-text-secondary">
                  {{ scope.fields.join(' / ') }}
                </p>
                <p v-if="scope.reason" class="mb-0 mt-2 text-sm leading-relaxed text-text-secondary">
                  {{ t('user.authorizedApps.reason', { reason: scope.reason }) }}
                </p>
                <p class="mb-0 mt-2 text-xs text-text-muted">
                  {{ t('user.authorizedApps.grantedAt', { time: formatDate(scope.grantedAt) }) }}
                </p>
                <p class="mb-0 mt-1 text-xs text-text-muted">
                  {{ scope.lastUsedAt ? t('user.authorizedApps.lastUsedAt', { time: formatDate(scope.lastUsedAt) }) : t('user.authorizedApps.neverUsed') }}
                </p>
              </div>

              <button
                class="inline-flex items-center justify-center gap-2 rounded-md border border-border bg-bg-card px-3 py-2 text-sm font-medium text-text-secondary transition-colors duration-fast hover:border-danger/40 hover:text-danger disabled:cursor-not-allowed disabled:opacity-50"
                type="button"
                :aria-label="t('user.authorizedApps.revokeScopeLabel', { scope: scope.displayName })"
                :disabled="Boolean(revokingKey)"
                @click="openRevokeScopeDialog(authorizedApp, scope)"
              >
                <Trash2 class="h-4 w-4" aria-hidden="true" />
                {{ revokingKey === scopeRevokeKey(authorizedApp.app.id, scope.scope) ? t('user.authorizedApps.revoking') : t('user.authorizedApps.revokeScope') }}
              </button>
            </li>
          </ul>
        </section>
      </article>
    </template>

    <section v-if="!loading" class="rounded-lg border border-border bg-bg-card p-5 shadow-sm">
      <header class="flex items-center justify-between gap-3">
        <div class="flex items-center gap-2">
          <Activity class="h-5 w-5 text-primary" aria-hidden="true" />
          <h3 class="m-0 text-base font-semibold text-text-primary">
            {{ t('user.authorizedApps.activityTitle') }}
          </h3>
        </div>
        <button
          v-if="activityErrorMessage"
          class="inline-flex items-center justify-center rounded-sm border border-border bg-transparent px-3 py-1.5 text-sm font-medium text-text-secondary transition-colors duration-fast hover:border-primary hover:text-primary"
          type="button"
          @click="loadActivities"
        >
          {{ t('common.actions.retry') }}
        </button>
      </header>

      <div v-if="activityLoading" class="mt-4 grid gap-3">
        <SkeletonCard v-for="i in 2" :key="i" />
      </div>

      <p v-else-if="activityErrorMessage" class="mb-0 mt-3 text-sm text-danger">
        {{ activityErrorMessage }}
      </p>

      <p v-else-if="activities.length === 0" class="mb-0 mt-3 text-sm text-text-muted">
        {{ t('user.authorizedApps.activityEmpty') }}
      </p>

      <ul v-else class="m-0 mt-4 grid list-none gap-3 p-0">
        <li
          v-for="event in activities"
          :key="event.id"
          class="grid gap-2 rounded-md bg-bg-elevated p-3 text-sm sm:grid-cols-[1fr_auto] sm:items-start"
        >
          <div class="min-w-0">
            <p class="m-0 font-medium text-text-primary">
              {{ activityLabel(event) }}
            </p>
            <p class="mb-0 mt-1 break-all text-xs text-text-muted">
              {{ activityAppLabel(event) }}
            </p>
            <p v-if="event.scopes.length" class="mb-0 mt-2 text-xs text-text-secondary">
              {{ t('user.authorizedApps.activityScopes', { scopes: event.scopes.join(' / ') }) }}
            </p>
            <p v-if="activityDetail(event)" class="mb-0 mt-1 text-xs text-text-muted">
              {{ activityDetail(event) }}
            </p>
          </div>
          <time class="text-xs text-text-muted sm:text-right" :datetime="event.createdAt">
            {{ formatDate(event.createdAt) }}
          </time>
        </li>
      </ul>
    </section>
  </div>

  <Teleport to="body">
    <div
      v-if="revokeDialog.open"
      class="fixed inset-0 z-50 grid place-items-center bg-black/45 p-4"
      role="presentation"
      @click.self="closeRevokeDialog"
    >
      <section
        ref="revokeDialogRef"
        class="w-full max-w-md rounded-lg border border-border bg-bg-card p-5 shadow-xl"
        role="dialog"
        aria-modal="true"
        :aria-labelledby="revokeDialogTitleID"
        :aria-describedby="revokeDialogDescriptionID"
        tabindex="-1"
        @keydown="handleRevokeDialogKeydown"
      >
        <header class="flex items-start justify-between gap-3">
          <div class="min-w-0">
            <h3 :id="revokeDialogTitleID" class="m-0 text-base font-semibold text-text-primary">
              {{ revokeDialogTitle }}
            </h3>
            <p
              :id="revokeDialogDescriptionID"
              class="m-0 mt-1 text-sm leading-relaxed text-text-secondary"
            >
              {{ revokeDialogDescription }}
            </p>
          </div>
          <button
            type="button"
            class="grid h-8 w-8 shrink-0 place-items-center rounded-md text-text-muted transition-colors hover:bg-bg-hover hover:text-text-primary disabled:cursor-not-allowed disabled:opacity-50"
            :aria-label="t('common.actions.close')"
            :disabled="revokeDialogSubmitting"
            @click="closeRevokeDialog"
          >
            <X class="h-4 w-4" aria-hidden="true" />
          </button>
        </header>

        <p
          v-if="revokeDialog.error"
          class="m-0 mt-3 rounded-md bg-danger/10 px-3 py-2 text-sm text-danger"
        >
          {{ revokeDialog.error }}
        </p>

        <div class="mt-5 flex flex-wrap justify-end gap-2">
          <button
            type="button"
            class="inline-flex h-9 items-center justify-center gap-2 rounded-md border border-border px-3 text-sm font-medium text-text-secondary transition-colors hover:bg-bg-hover hover:text-text-primary disabled:cursor-not-allowed disabled:opacity-50"
            :disabled="revokeDialogSubmitting"
            @click="closeRevokeDialog"
          >
            <X class="h-4 w-4" aria-hidden="true" />
            {{ t('common.actions.cancel') }}
          </button>
          <button
            type="button"
            class="inline-flex h-9 items-center justify-center gap-2 rounded-md bg-danger px-3 text-sm font-semibold text-white transition-colors hover:bg-danger/90 disabled:cursor-not-allowed disabled:opacity-50"
            :disabled="revokeDialogSubmitting"
            @click="submitRevokeDialog"
          >
            <Trash2 class="h-4 w-4" aria-hidden="true" />
            {{ revokeDialogSubmitting ? t('user.authorizedApps.revoking') : t('user.authorizedApps.confirmRevoke') }}
          </button>
        </div>
      </section>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Activity, ExternalLink, ShieldCheck, Trash2, X } from 'lucide-vue-next'
import SkeletonCard from '@/components/common/SkeletonCard.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import { useBodyScrollLock } from '@/composables/useBodyScrollLock'
import { useDialogFocus } from '@/composables/useDialogFocus'
import { useAuthorizedAppsController } from '../authorizedAppsController'
import type {
  OpenPlatformUserConsentAuditEvent,
  OpenPlatformUserConsentScope,
} from '@stuhelper/shared/api'

type Sensitivity = OpenPlatformUserConsentScope['sensitivity']

const revokeDialogTitleID = 'authorized-apps-revoke-dialog-title'
const revokeDialogDescriptionID = 'authorized-apps-revoke-dialog-description'

const { t } = useI18n()
const translate = (key: string, params?: Record<string, unknown>) => t(key, params ?? {})
const {
  apps,
  activities,
  loading,
  activityLoading,
  errorMessage,
  activityErrorMessage,
  revokingKey,
  revokeDialog,
  revokeDialogTitle,
  revokeDialogDescription,
  revokeDialogSubmitting,
  loadConsents,
  loadActivities,
  openRevokeAppDialog,
  openRevokeScopeDialog,
  closeRevokeDialog,
  submitRevokeDialog,
  appRevokeKey,
  scopeRevokeKey,
} = useAuthorizedAppsController(translate)

const revokeDialogRef = ref<HTMLElement | null>(null)
const revokeDialogOpen = computed(() => revokeDialog.open)

useBodyScrollLock(revokeDialogOpen)
const { handleKeydown: handleRevokeDialogKeydown } = useDialogFocus({
  close: closeRevokeDialog,
  dialogRef: revokeDialogRef,
  open: revokeDialogOpen,
})

function formatDate(value: string) {
  return new Date(value).toLocaleString()
}

function sensitivityLabel(value: Sensitivity) {
  return t(`common.openPlatformConsent.sensitivity.${value}`)
}

function sensitivityClass(value: Sensitivity) {
  return {
    'bg-secondary/10 text-secondary': value === 'low',
    'bg-warning/10 text-warning': value === 'medium',
    'bg-danger/10 text-danger': value === 'high' || value === 'very_high',
  }
}

function activityLabel(event: OpenPlatformUserConsentAuditEvent) {
  switch (event.eventType) {
    case 'open_platform.consent.granted':
      return t('user.authorizedApps.activityTypes.granted')
    case 'open_platform.consent.denied':
      return t('user.authorizedApps.activityTypes.denied')
    case 'open_platform.consent.revoked':
      return t('user.authorizedApps.activityTypes.revoked')
    case 'open_platform.disclosure.granted':
      return t('user.authorizedApps.activityTypes.disclosureGranted')
    case 'open_platform.disclosure.denied':
      return t('user.authorizedApps.activityTypes.disclosureDenied')
    case 'open_platform.disclosure.replay_detected':
      return t('user.authorizedApps.activityTypes.replayDetected')
    default:
      return event.eventType
  }
}

function activityAppLabel(event: OpenPlatformUserConsentAuditEvent) {
  return event.appDisplayName ?? event.clientID ?? t('user.authorizedApps.unknownApp')
}

function activityDetail(event: OpenPlatformUserConsentAuditEvent) {
  const parts: string[] = []
  if (event.endpoint) {
    parts.push(t('user.authorizedApps.activityEndpoint', { endpoint: event.endpoint }))
  }
  if (event.result) {
    parts.push(t('user.authorizedApps.activityResult', { result: event.result }))
  }
  return parts.join(' · ')
}

onMounted(() => {
  void loadConsents()
  void loadActivities()
})
</script>

<style scoped>
.authorized-app-link {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  color: var(--color-primary);
  font-weight: 600;
  text-decoration: none;
}

.authorized-app-link:hover {
  color: var(--color-primary-dark);
}
</style>
