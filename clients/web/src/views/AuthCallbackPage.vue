<template>
  <div class="callback-page">
    <div class="loading-card">
      <div v-if="loading" class="loading">
        <div class="spinner"></div>
        <p>{{ t('errors.authCallback.loading') }}</p>
      </div>

      <div v-else-if="error" class="error">
        <p>{{ t('errors.authCallback.error') }}</p>
        <p class="error-message">{{ error }}</p>
        <button @click="goToLogin">{{ t('errors.authCallback.backToLogin') }}</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const { t } = useI18n()

const loading = ref(true)
const error = ref('')

onMounted(async () => {
  const code = route.query.code as string
  const state = route.query.state as string

  if (!code) {
    error.value = t('errors.authCallback.missingCode')
    loading.value = false
    return
  }

  if (!state) {
    error.value = t('errors.authCallback.missingState')
    loading.value = false
    return
  }

  try {
    await authStore.handleCallback(code, state)
    router.push('/')
  } catch {
    error.value = t('errors.authCallback.loginFailed')
    loading.value = false
  }
})

const goToLogin = () => {
  router.push('/login')
}
</script>

<style scoped>
.callback-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-base);
}

.loading-card {
  background: var(--bg-card);
  padding: var(--space-12) var(--space-10);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  text-align: center;
  max-width: 360px;
  width: 100%;
}

.loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-4);
  color: var(--text-muted);
  font-size: var(--text-sm);
}

.spinner {
  width: 32px;
  height: 32px;
  border: 2px solid var(--border);
  border-top-color: var(--accent);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.error {
  color: var(--text-primary);
}

.error-message {
  color: var(--text-muted);
  font-size: var(--text-sm);
  margin: var(--space-3) 0 var(--space-6);
}

button {
  padding: var(--space-2) var(--space-6);
  background: var(--text-primary);
  color: var(--bg-base);
  border: none;
  border-radius: var(--radius-sm);
  font-size: var(--text-sm);
  font-weight: var(--weight-medium);
  cursor: pointer;
  transition: all var(--duration-fast);
}

button:hover {
  background: var(--accent);
  color: white;
}
</style>
