<script setup lang="ts">
import { ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { SSO_STATE_STORAGE_KEY, validateStoredSSOState } from '@/auth/sso-state'
import { useAuthStore } from '@/stores/auth'
import { translate } from '@/i18n'

const authStore = useAuthStore()
const t = translate
const status = ref<'loading' | 'success' | 'error'>('loading')
const errorMessage = ref('')

onLoad(async (options) => {
  const code = typeof options?.code === 'string' ? options.code : ''
  const state = typeof options?.state === 'string' ? options.state : ''

  if (!code || !state) {
    status.value = 'error'
    errorMessage.value = t('auth.callback.missingParams')
    return
  }

  // 校验 state 与发起 SSO 时存储的一致
  let savedState: unknown = ''
  try {
    savedState = uni.getStorageSync(SSO_STATE_STORAGE_KEY)
  } catch (_error) { void _error;
    savedState = ''
  }

  try {
    uni.removeStorageSync(SSO_STATE_STORAGE_KEY)
  } catch (_error) { void _error;
    // best-effort cleanup; validation above already fails closed
  }

  if (!validateStoredSSOState(savedState, state).ok) {
    status.value = 'error'
    errorMessage.value = t('auth.callback.stateMismatch')
    return
  }

  try {
    // 用授权码换取 token 并持久化
    await authStore.exchangeNativeCode(code, state)

    status.value = 'success'
    uni.showToast({ title: t('auth.login.success'), icon: 'success' })

    // 短暂延迟后跳转首页
    setTimeout(() => {
      uni.reLaunch({ url: '/pages/user/index' })
    }, 500)
  } catch (error) {
    status.value = 'error'
    errorMessage.value = error instanceof Error ? error.message : t('auth.callback.exchangeFailed')
  }
})

function handleRetry() {
  uni.reLaunch({ url: '/pages/auth/login' })
}
</script>

<template>
  <view class="callback-page">
    <view class="callback-card">
      <!-- 加载中 -->
      <view v-if="status === 'loading'" class="status-block">
        <text class="spinner">⏳</text>
        <text class="status-text">{{ t('auth.callback.processing') }}</text>
      </view>

      <!-- 成功 -->
      <view v-if="status === 'success'" class="status-block">
        <text class="spinner">✅</text>
        <text class="status-text">{{ t('auth.callback.success') }}</text>
      </view>

      <!-- 失败 -->
      <view v-if="status === 'error'" class="status-block">
        <text class="spinner">❌</text>
        <text class="status-text">{{ t('auth.callback.failed') }}</text>
        <text class="error-detail">{{ errorMessage }}</text>
        <button class="retry-btn" @tap="handleRetry">
          {{ t('auth.callback.retry') }}
        </button>
      </view>
    </view>
  </view>
</template>

<style scoped>
.callback-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48rpx;
  background: linear-gradient(180deg, #eef2ff 0%, #f8fafc 50%, #ffffff 100%);
}

.callback-card {
  width: 100%;
  max-width: 600rpx;
  background: #ffffff;
  border-radius: 32rpx;
  padding: 64rpx 48rpx;
  box-shadow: 0 12rpx 40rpx rgba(15, 23, 42, 0.08);
}

.status-block {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 24rpx;
}

.spinner {
  font-size: 72rpx;
}

.status-text {
  font-size: 32rpx;
  font-weight: 600;
  color: #0f172a;
  text-align: center;
}

.error-detail {
  font-size: 24rpx;
  color: #64748b;
  text-align: center;
  line-height: 1.6;
}

.retry-btn {
  margin-top: 32rpx;
  width: 320rpx;
  height: 80rpx;
  border-radius: 22rpx;
  background: #4f46e5;
  color: #ffffff;
  font-size: 28rpx;
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: center;
}
</style>
