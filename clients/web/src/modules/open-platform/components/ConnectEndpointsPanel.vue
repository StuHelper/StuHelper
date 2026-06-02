<template>
  <section class="rounded-lg border border-border bg-bg-card p-5 shadow-sm">
    <header class="flex items-start gap-3">
      <span class="grid size-10 shrink-0 place-items-center rounded-lg bg-primary-alpha text-primary">
        <Network class="size-5" aria-hidden="true" />
      </span>
      <div class="min-w-0">
        <h2 class="m-0 text-base font-semibold text-text-primary">
          {{ t('developer.apps.connect.title') }}
        </h2>
        <p class="m-0 mt-1 text-sm leading-relaxed text-text-secondary">
          {{ t('developer.apps.connect.subtitle') }}
        </p>
      </div>
    </header>

    <dl class="m-0 mt-4 grid gap-2">
      <div
        v-for="endpoint in connectEndpoints"
        :key="endpoint.key"
        class="rounded-md border border-border bg-bg-elevated p-3"
      >
        <dt class="text-xs font-medium uppercase text-text-muted">
          {{ connectEndpointLabel(endpoint.key) }}
        </dt>
        <dd class="m-0 mt-2 flex min-w-0 items-start gap-2">
          <code class="min-w-0 flex-1 break-all rounded bg-bg-card px-2 py-1.5 text-xs text-text-secondary">
            {{ endpoint.value }}
          </code>
          <button
            type="button"
            class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md border border-border text-text-secondary transition-colors hover:border-primary/40 hover:text-primary"
            :aria-label="t('developer.apps.connect.copyLabel', { endpoint: connectEndpointLabel(endpoint.key) })"
            @click="copyConnectEndpoint(endpoint.key, endpoint.value)"
          >
            <Check v-if="copiedEndpoint === endpoint.key" class="h-4 w-4 text-success" aria-hidden="true" />
            <Copy v-else class="h-4 w-4" aria-hidden="true" />
          </button>
        </dd>
      </div>
    </dl>
  </section>
</template>

<script setup lang="ts">
import { computed, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Check, Copy, Network } from 'lucide-vue-next'
import { useToast } from '@/composables/useToast'
import {
  buildConnectEndpoints,
  normalizeSsoIssuer,
  type ConnectEndpointKey,
} from '../connectEndpoints'

const { t } = useI18n()
const toast = useToast()

const copiedEndpoint = ref<ConnectEndpointKey | null>(null)
let copiedEndpointResetTimer: ReturnType<typeof setTimeout> | null = null

const identityIssuer = computed(() =>
  normalizeSsoIssuer(
    import.meta.env.VITE_SSO_URL,
    typeof window === 'undefined' ? undefined : window.location.origin,
  ),
)
const connectEndpoints = computed(() => buildConnectEndpoints(identityIssuer.value))

function connectEndpointLabel(key: ConnectEndpointKey) {
  return t(`developer.apps.connect.endpoints.${key}`)
}

async function copyConnectEndpoint(key: ConnectEndpointKey, value: string) {
  try {
    if (!navigator.clipboard?.writeText) {
      throw new Error('clipboard is unavailable')
    }
    await navigator.clipboard.writeText(value)
    copiedEndpoint.value = key
    if (copiedEndpointResetTimer) {
      clearTimeout(copiedEndpointResetTimer)
    }
    copiedEndpointResetTimer = setTimeout(() => {
      if (copiedEndpoint.value === key) {
        copiedEndpoint.value = null
      }
    }, 1600)
    toast.success(t('developer.apps.connect.copied'))
  } catch {
    toast.error(t('developer.apps.connect.copyFailed'))
  }
}

onUnmounted(() => {
  if (copiedEndpointResetTimer) {
    clearTimeout(copiedEndpointResetTimer)
  }
})
</script>
