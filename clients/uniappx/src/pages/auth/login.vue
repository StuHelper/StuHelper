<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { onLoad } from '@dcloudio/uni-app'
import { api } from '@/api'
import { unwrapData } from '@/api/result'
import { translate } from '@/i18n'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()
const t = translate
const redirect = ref('/pages/user/index')
const phone = ref('')
const code = ref('')
const requestingCode = ref(false)
const verifying = ref(false)
const ssoLoading = ref(false)
const cooldown = ref(0)
const supportsSso = typeof window !== 'undefined' && typeof window.location?.href === 'string'
const isNativeApp = typeof plus !== 'undefined'
let cooldownTimer: ReturnType<typeof setInterval> | null = null
const phonePattern = /^1[3-9]\d{9}$/

onLoad((options) => {
  redirect.value = typeof options?.redirect === 'string' && options.redirect ? options.redirect : '/pages/user/index'
})

const canRequestCode = computed(() => phonePattern.test(phone.value.trim()) && cooldown.value === 0)
const canVerify = computed(() => phonePattern.test(phone.value.trim()) && /^\d{6}$/.test(code.value.trim()))

function startCooldown(seconds: number) {
  cooldown.value = seconds
  if (cooldownTimer) clearInterval(cooldownTimer)
  cooldownTimer = setInterval(() => {
    cooldown.value = Math.max(0, cooldown.value - 1)
    if (cooldown.value === 0 && cooldownTimer) {
      clearInterval(cooldownTimer)
      cooldownTimer = null
    }
  }, 1000)
}

async function handleRequestCode() {
  if (!canRequestCode.value || requestingCode.value) return
  requestingCode.value = true
  try {
    const data = await authStore.requestPhoneOTP(phone.value.trim())
    startCooldown(data.cooldown)
    uni.showToast({ title: data.message || t('auth.login.codeSent'), icon: 'none' })
  } catch (error) {
    uni.showToast({
      title: error instanceof Error ? error.message : t('auth.login.codeSendFailed'),
      icon: 'none',
    })
  } finally {
    requestingCode.value = false
  }
}

async function handleVerify() {
  if (!canVerify.value || verifying.value) return
  verifying.value = true
  try {
    await authStore.verifyPhoneOTP(phone.value.trim(), code.value.trim())
    uni.showToast({ title: t('auth.login.success'), icon: 'success' })
    setTimeout(() => {
      if (redirect.value.startsWith('/pages/')) {
        uni.reLaunch({ url: redirect.value })
        return
      }
      uni.switchTab({ url: '/pages/user/index' })
    }, 250)
  } catch (error) {
    uni.showToast({
      title: error instanceof Error ? error.message : t('auth.login.verifyFailed'),
      icon: 'none',
    })
  } finally {
    verifying.value = false
  }
}

async function handleSSOLogin() {
  if (ssoLoading.value) return
  ssoLoading.value = true
  try {
    const result = await api.auth.login(redirect.value || '/pages/user/index')
    const data = unwrapData<{ url: string; state: string }>(result)
    if (!data.url) throw new Error(t('auth.login.ssoInitFailed'))

    if (isNativeApp) {
      // Native: open system browser for SSO (supports password managers, saved sessions)
      plus.runtime.openURL(data.url)
      return
    }
    if (supportsSso) {
      // H5/WebView: direct redirect
      window.location.href = data.url
      return
    }
    // Fallback: SSO unavailable
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

onBeforeUnmount(() => {
  if (cooldownTimer) clearInterval(cooldownTimer)
})
</script>

<template>
  <view class="login-page">
    <view class="login-card">
      <text class="logo">🎓</text>
      <text class="title">{{ t('auth.login.title') }}</text>
      <text class="subtitle">{{ t('auth.login.subtitle') }}</text>

      <view class="field-group">
        <text class="field-label">{{ t('auth.login.phone') }}</text>
        <input
          v-model="phone"
          class="field-input"
          maxlength="11"
          :placeholder="t('auth.login.phonePlaceholder')"
          type="number"
        />
      </view>

      <view class="field-group">
        <text class="field-label">{{ t('auth.login.code') }}</text>
        <view class="code-row">
          <input
            v-model="code"
            class="field-input code-input"
            maxlength="6"
            :placeholder="t('auth.login.codePlaceholder')"
            type="number"
          />
          <button
            class="secondary-btn code-btn"
            :disabled="!canRequestCode || requestingCode"
            @tap="handleRequestCode"
          >
            {{ cooldown > 0 ? `${cooldown}s` : requestingCode ? t('auth.login.sending') : t('auth.login.requestCode') }}
          </button>
        </view>
      </view>

      <button class="primary-btn" :disabled="!canVerify || verifying" @tap="handleVerify">
        {{ verifying ? t('auth.login.loggingIn') : t('auth.login.otpLogin') }}
      </button>

      <view class="divider"><text>{{ t('common.or') }}</text></view>

      <button class="secondary-btn" :disabled="ssoLoading" @tap="handleSSOLogin">
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

.field-group {
  margin-top: 28rpx;
}

.field-label {
  display: block;
  margin-bottom: 12rpx;
  color: #334155;
  font-size: 26rpx;
  font-weight: 600;
}

.field-input {
  width: 100%;
  height: 88rpx;
  border-radius: 22rpx;
  background: #f8fafc;
  border: 2rpx solid #e2e8f0;
  padding: 0 24rpx;
  box-sizing: border-box;
  font-size: 28rpx;
}

.code-row {
  display: flex;
  gap: 16rpx;
}

.code-input {
  flex: 1;
}

.code-btn {
  width: 220rpx;
  min-width: 220rpx;
}

.primary-btn,
.secondary-btn {
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

.secondary-btn {
  background: #eef2ff;
  color: #4338ca;
}

.divider {
  margin-top: 28rpx;
  text-align: center;
  color: #94a3b8;
  font-size: 24rpx;
}
</style>
