<template>
  <div class="min-h-screen flex items-center justify-center bg-gray-100">
    <div class="w-full max-w-sm p-8 bg-white rounded-xl shadow-lg">
      <div class="text-center mb-8">
        <div class="mx-auto w-12 h-12 rounded-xl bg-gradient-to-br from-blue-500 to-indigo-600 flex items-center justify-center mb-4">
          <span class="text-white text-xl font-bold">S</span>
        </div>
        <h1 class="text-xl font-bold text-gray-900">StuHelper Admin</h1>
        <p class="text-sm text-gray-500 mt-1">管理后台需要管理员权限</p>
      </div>

      <div v-if="error" class="mb-4">
        <el-alert :title="error" type="error" show-icon :closable="false" />
      </div>

      <el-button type="primary" size="large" class="w-full" :loading="loading" @click="handleLogin">
        SSO 登录
      </el-button>

      <div class="mt-4 text-center">
        <a href="/" class="text-sm text-gray-400 hover:text-gray-600 transition-colors">返回首页</a>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useRoute } from 'vue-router'

const authStore = useAuthStore()
const route = useRoute()
const loading = ref(false)
const error = ref('')

async function handleLogin() {
  loading.value = true
  error.value = ''
  try {
    // Redirect to SSO login - admin uses the same SSO as the main app
    const loginURL = authStore.getSSOLoginURL()
    window.location.href = loginURL
  } catch {
    error.value = '登录跳转失败，请稍后重试'
    loading.value = false
  }
}

// Check for error in query params
if (route.query.error) {
  error.value = route.query.error === 'not_admin' ? '您没有管理员权限' : '登录失败'
}
</script>
