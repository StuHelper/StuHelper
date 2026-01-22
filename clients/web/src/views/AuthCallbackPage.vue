<template>
  <div class="callback-page">
    <div class="loading-card">
      <div v-if="loading" class="loading">
        <div class="spinner"></div>
        <p>正在登录中...</p>
      </div>

      <div v-else-if="error" class="error">
        <p>登录失败</p>
        <p class="error-message">{{ error }}</p>
        <button @click="goToLogin">返回登录</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const loading = ref(true)
const error = ref('')

onMounted(async () => {
  const code = route.query.code as string
  const state = route.query.state as string

  if (!code) {
    error.value = '缺少授权码'
    loading.value = false
    return
  }

  if (!state) {
    error.value = '缺少 state 参数'
    loading.value = false
    return
  }

  try {
    await authStore.handleCallback(code, state)
    router.push('/')
  } catch (err) {
    error.value = err instanceof Error ? err.message : '登录失败，请重试'
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
  background: #f5f5f5;
}

.loading-card {
  background: white;
  padding: 3rem;
  border-radius: 1rem;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
  text-align: center;
}

.loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
}

.spinner {
  width: 40px;
  height: 40px;
  border: 3px solid #f3f3f3;
  border-top: 3px solid #667eea;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.error {
  color: #e74c3c;
}

.error-message {
  color: #666;
  font-size: 0.875rem;
  margin: 1rem 0;
}

button {
  padding: 0.75rem 1.5rem;
  background: #667eea;
  color: white;
  border: none;
  border-radius: 0.5rem;
  cursor: pointer;
}
</style>
