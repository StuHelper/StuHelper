<script setup lang="ts">
import { ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { persistSSOState } from '@/auth/sso-state'
import { api } from '@/api'
import { unwrapData } from '@/api/result'
import { translate } from '@/i18n'

const t = translate
const redirect = ref('/pages/user/index')
const ssoLoading = ref(false)
const supportsSso = typeof window !== 'undefined' && typeof window.location?.href === 'string'
const isNativeApp = typeof plus !== 'undefined'

onLoad((options) => {
  redirect.value = typeof options?.redirect === 'string' && options.redirect ? options.redirect : '/pages/user/index'
})

async function handleSSOLogin() {
  if (ssoLoading.value) return
  ssoLoading.value = true
  try {
    // 原生 App 需要传 platform=native，服务端会标记该 state 走 deep link 回调流程
    const redirectPath = redirect.value || '/pages/user/index'
    const platform = isNativeApp ? 'native' : undefined
    const result = await api.auth.login(redirectPath, platform, 'uniapp')
    const data = unwrapData<{ url: string; state: string }>(result)
    if (!data.url) throw new Error(t('auth.login.ssoInitFailed'))

    if (isNativeApp) {
      // 暂存 state 到本地，deep link 回调时用于校验
      persistSSOState(data.state)
      // 原生：打开系统浏览器完成 SSO（支持密码管理器、已保存的会话）
      plus.runtime.openURL(data.url)
      return
    }
    if (supportsSso) {
      // H5/WebView：直接跳转
      window.location.href = data.url
      return
    }
    // 兜底：SSO 不可用
    uni.showToast({ title: t('auth.login.ssoUnavailable'), icon: 'none' })
  } catch (error) {
    uni.showToast({
      title: error instanceof Error ? error.message : t('auth.login.ssoInitFailed'),
      icon: 'none',
    })
  } finally {
    ssoLoading.value = false
  }
}
</script>

<template>
  <view class="login-page">
    <view class="login-card">
      <text class="logo">🎓</text>
      <text class="title">{{ t('auth.login.title') }}</text>
      <text class="subtitle">{{ t('auth.login.subtitle') }}</text>

      <button class="primary-btn" :disabled="ssoLoading" @tap="handleSSOLogin">
        {{ ssoLoading ? t('auth.login.preparingSso') : t('auth.login.useSso') }}
      </button>

      <text class="tips">{{ t('auth.login.tips') }}</text>
    </view>
  </view>
</template>

<style scoped>
.login-page {
  min-height: 100vh;
  padding: 48rpx;
  background: linear-gradient(180deg, #eef2ff 0%, #f8fafc 50%, #ffffff 100%);
}

.login-card {
  margin-top: 80rpx;
  background: #ffffff;
  border-radius: 32rpx;
  padding: 48rpx 40rpx;
  box-shadow: 0 12rpx 40rpx rgba(15, 23, 42, 0.08);
}

.logo {
  font-size: 72rpx;
  text-align: center;
  display: block;
  margin-bottom: 20rpx;
}

.title {
  display: block;
  text-align: center;
  font-size: 42rpx;
  font-weight: 700;
  color: #0f172a;
}

.subtitle,
.tips {
  display: block;
  text-align: center;
  color: #64748b;
  font-size: 24rpx;
  line-height: 1.6;
}

.subtitle {
  margin-top: 16rpx;
}

.tips {
  margin-top: 24rpx;
}

.primary-btn {
  margin-top: 24rpx;
  height: 88rpx;
  border-radius: 22rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 28rpx;
  font-weight: 600;
}

.primary-btn {
  background: #4f46e5;
  color: #ffffff;
}
</style>
