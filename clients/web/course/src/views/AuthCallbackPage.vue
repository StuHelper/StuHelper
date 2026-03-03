<template>
  <div class="min-h-screen flex items-center justify-center bg-bg-base">
    <div class="bg-bg-card py-12 px-10 border border-border rounded-md text-center max-w-[360px] w-full">
      <!-- 加载中 -->
      <div v-if="loading" class="flex flex-col items-center gap-4 text-text-muted text-sm">
        <div class="w-8 h-8 border-2 border-border border-t-accent rounded-full animate-spin"></div>
        <p>{{ t('errors.authCallback.loading') }}</p>
      </div>

      <!-- 组织不匹配 -->
      <div v-else-if="orgMismatch" class="text-text-primary">
        <p class="text-base font-semibold m-0 mb-2">{{ t('errors.authCallback.orgMismatch') }}</p>
        <p class="text-text-muted text-sm my-3 mb-6">{{ t('errors.authCallback.orgMismatchHint') }}</p>
        <div class="flex flex-col gap-3">
          <button
            class="py-2 px-6 rounded-sm text-sm font-medium cursor-pointer transition-all duration-fast bg-gradient-to-br from-primary to-accent text-white border-none hover:opacity-90"
            :disabled="loggingOut"
            @click="handleSsoLogout"
          >
            {{ loggingOut ? t('errors.authCallback.loading') : t('errors.authCallback.ssoLogout') }}
          </button>
        </div>
      </div>

      <!-- 通用错误 -->
      <div v-else-if="error" class="text-text-primary">
        <p>{{ t('errors.authCallback.error') }}</p>
        <p class="text-text-muted text-sm my-3 mb-6">{{ error }}</p>
        <button
          class="py-2 px-6 bg-text-primary text-bg-base border-none rounded-sm text-sm font-medium cursor-pointer transition-all duration-fast hover:bg-accent hover:text-white"
          @click="goToLogin"
        >
          {{ t('errors.authCallback.backToLogin') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { isApiError } from '@/api/errors'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const { t } = useI18n()

const loading = ref(true)
const error = ref('')
const orgMismatch = ref(false)
const ssoLogoutURL = ref('')
const loggingOut = ref(false)

onMounted(async () => {
  // 刷新页面时恢复 org mismatch 状态（code/state 是一次性的，刷新后无法重新请求）
  const saved = sessionStorage.getItem('org_mismatch')
  if (saved) {
    orgMismatch.value = true
    ssoLogoutURL.value = saved
    loading.value = false
    return
  }

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
    // 如果有草稿重定向，回到原页面；否则回首页
    const draftRedirect = sessionStorage.getItem('draft_redirect')
    if (draftRedirect) {
      sessionStorage.removeItem('draft_redirect')
      router.push(draftRedirect)
    } else {
      router.push('/')
    }
  } catch (err) {
    if (isApiError(err) && err.status === 403) {
      orgMismatch.value = true
      ssoLogoutURL.value = (err.details?.ssoLogoutURL as string) || ''
      sessionStorage.setItem('org_mismatch', ssoLogoutURL.value)
    } else {
      error.value = t('errors.authCallback.loginFailed')
    }
    loading.value = false
  }
})

let logoutTimer: ReturnType<typeof setTimeout> | null = null

onUnmounted(() => {
  if (logoutTimer) clearTimeout(logoutTimer)
})

// 通过弹窗访问 Casdoor 登出端点清除 SSO session cookie
// 必须用顶级导航（window.open），因为 SameSite=Lax cookie 不会随 fetch/iframe 发送
const handleSsoLogout = () => {
  sessionStorage.removeItem('org_mismatch')
  if (!ssoLogoutURL.value) {
    router.push('/login')
    return
  }
  // 验证 URL 协议，防止 javascript:/data: 等 XSS 向量
  try {
    const parsed = new URL(ssoLogoutURL.value)
    if (parsed.protocol !== 'https:' && parsed.protocol !== 'http:') {
      console.warn('[AuthCallback] SSO logout URL has unsafe protocol')
      router.push('/login')
      return
    }
  } catch {
    console.warn('[AuthCallback] SSO logout URL is invalid')
    router.push('/login')
    return
  }
  loggingOut.value = true
  const popup = window.open(
    ssoLogoutURL.value, 'sso_logout',
    'width=1,height=1,left=0,top=0,menubar=no,toolbar=no,status=no'
  )
  if (!popup) {
    // 弹窗被浏览器拦截，直接跳转登录页（SSO session 未清除，但不阻塞用户）
    console.warn('[AuthCallback] SSO logout popup blocked by browser')
    router.push('/login')
    return
  }
  logoutTimer = setTimeout(() => {
    try { popup.close() } catch { /* 跨域忽略 */ }
    router.push('/login')
  }, 500)
}

const goToLogin = () => {
  sessionStorage.removeItem('org_mismatch')
  router.push('/login')
}
</script>
