<script setup lang="ts">
import { computed, ref } from 'vue'
import { onShow } from '@dcloudio/uni-app'
import A11yButton from '@/components/A11yButton.vue'
import { api } from '@/api'
import type { components } from '@/api'
import { unwrapOptionalData } from '@/api/result'
import { getUserMenuItems } from '@/config/featureSurface'
import { setPageTitle, translate } from '@/i18n'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()
const t = translate
const loading = ref(false)
const logoutLoading = ref(false)
const surface = ref<components['schemas']['UserSurface'] | null>(null)
const phoneStatus = ref<components['schemas']['PhoneStatus'] | null>(null)
const menus = computed(() => getUserMenuItems())
let surfaceLoadPromise: Promise<void> | null = null

const verificationSummary = computed(() => {
  if (!surface.value) return t('user.index.verification.unverified')
  if (surface.value.studentVerificationStatus === 'approved') return t('user.index.verification.verified')
  return t('user.index.verification.unverified')
})

const studentVerified = computed(() => surface.value?.studentVerificationStatus === 'approved')
const phoneVerified = computed(() => phoneStatus.value?.publishingRequirementSatisfied === true)

function goVerify(type: 'student' | 'phone') {
  if (!authStore.isAuthenticated) {
    uni.navigateTo({ url: '/pages/auth/login' })
    return
  }
  const webBase = import.meta.env.VITE_WEB_URL || ''
  const path = type === 'phone' ? '/user/phone-binding' : '/user/student-verification'
  if (!webBase) {
    uni.showToast({ title: t('user.index.verifyOnWeb'), icon: 'none' })
    return
  }
  const targetURL = webBase + path
  // #ifdef H5
  window.location.href = targetURL
  // #endif
  // #ifndef H5
  uni.showModal({
    title: t('user.index.verifyInBrowser'),
    content: t('user.index.verifyBrowserHint'),
    confirmText: t('common.confirm'),
    cancelText: t('common.cancel'),
    success: (res) => {
      if (res.confirm) {
        const nativePlus = (globalThis as { plus?: { runtime?: { openURL: (url: string) => void } } }).plus
        if (nativePlus?.runtime) {
          nativePlus.runtime.openURL(targetURL)
        } else {
          uni.showToast({ title: t('user.index.verifyOnWeb'), icon: 'none' })
        }
      }
    }
  })
  // #endif
}

async function performUserSurfaceLoad() {
  loading.value = true
  try {
    await authStore.bootstrapSession()
    if (!authStore.isAuthenticated) {
      surface.value = null
      phoneStatus.value = null
      return
    }
    const [surfaceResult, phoneResult] = await Promise.all([
      api.identity.getUserSurface(),
      api.studentVerification.getPhoneStatus(),
    ])
    surface.value = unwrapOptionalData<components['schemas']['UserSurface']>(surfaceResult)
    phoneStatus.value = unwrapOptionalData<components['schemas']['PhoneStatus']>(phoneResult)
  } catch (error) {
    uni.showToast({
      title: error instanceof Error ? error.message : t('common.retryLater'),
      icon: 'none',
    })
  } finally {
    loading.value = false
  }
}

function loadUserSurface(): Promise<void> {
  if (surfaceLoadPromise) return surfaceLoadPromise
  surfaceLoadPromise = performUserSurfaceLoad().finally(() => {
    surfaceLoadPromise = null
  })
  return surfaceLoadPromise
}

function open(path: string) {
  if (!authStore.isAuthenticated) {
    uni.navigateTo({ url: `/pages/auth/login?redirect=${encodeURIComponent(path)}` })
    return
  }
  uni.navigateTo({ url: path })
}

async function handleLogout() {
  if (logoutLoading.value) return
  logoutLoading.value = true
  try {
    await authStore.logout()
    surface.value = null
    phoneStatus.value = null
    uni.showToast({ title: t('user.index.logoutSuccess'), icon: 'none' })
  } catch (error) {
    uni.showToast({
      title: error instanceof Error ? error.message : t('common.retryLater'),
      icon: 'none',
    })
  } finally {
    logoutLoading.value = false
  }
}

onShow(() => {
  setPageTitle('common.pageTitles.userCenter')
  void loadUserSurface()
})
</script>

<template>
  <scroll-view class="user-page" scroll-y>
    <view class="hero-card">
      <view class="hero-main">
        <view class="avatar">{{ authStore.isAuthenticated ? '👤' : '🙂' }}</view>
        <view>
          <text class="username">{{ authStore.displayName }}</text>
          <text class="user-subtitle">{{ authStore.user?.email || t('user.index.subtitleGuest') }}</text>
        </view>
      </view>
      <A11yButton
        v-if="!authStore.isAuthenticated"
        class="login-btn"
        data-testid="uni-user-login"
        @tap="open('/pages/user/index')"
      >
        {{ t('user.index.loginNow') }}
      </A11yButton>
      <A11yButton v-else class="logout-btn" data-testid="uni-user-logout" :disabled="logoutLoading" @tap="handleLogout">
        {{ logoutLoading ? t('common.processing') : t('user.index.logout') }}
      </A11yButton>
    </view>

    <view v-if="loading" class="state-card"><text>{{ t('common.loading') }}</text></view>

    <view v-else class="section-card">
      <text class="section-title">{{ t('user.index.authOverview') }}</text>
      <view class="summary-row">
        <A11yButton
          class="summary-item"
          data-testid="uni-user-student-summary"
          :aria-disabled="studentVerified"
          @tap="!studentVerified && goVerify('student')"
        >
          <text class="summary-label">{{ t('user.index.student') }}</text>
          <text class="summary-value" :class="{ 'action-link': !studentVerified }">{{ verificationSummary }}</text>
          <text v-if="!studentVerified" class="action-hint">{{ t('user.index.tapToVerify') }}</text>
        </A11yButton>
        <A11yButton
          class="summary-item"
          data-testid="uni-user-phone-summary"
          :aria-disabled="phoneVerified"
          @tap="!phoneVerified && goVerify('phone')"
        >
          <text class="summary-label">{{ t('user.index.phone') }}</text>
          <text class="summary-value" :class="{ 'action-link': !phoneVerified }">
            {{ phoneVerified ? t('user.index.phoneBound') : t('user.index.phoneUnbound') }}
          </text>
          <text v-if="!phoneVerified" class="action-hint">{{ t('user.index.tapToVerify') }}</text>
        </A11yButton>
      </view>
    </view>

    <view class="section-card">
      <text class="section-title">{{ t('user.index.commonFeatures') }}</text>
      <A11yButton v-for="item in menus" :key="item.path" class="menu-row" @tap="open(item.path)">
        <view class="menu-main">
          <text class="menu-icon">{{ item.icon }}</text>
          <text class="menu-title">{{ item.title }}</text>
        </view>
        <text class="menu-arrow">›</text>
      </A11yButton>
    </view>
  </scroll-view>
</template>

<style scoped>
.user-page {
  min-height: 100vh;
  background: #f8fafc;
}

.hero-card,
.section-card,
.state-card {
  margin: 24rpx;
  background: #ffffff;
  border-radius: 24rpx;
  box-shadow: 0 10rpx 30rpx rgba(15, 23, 42, 0.05);
}

.hero-card {
  padding: 32rpx;
}

.hero-main {
  display: flex;
  align-items: center;
  gap: 24rpx;
}

.avatar {
  width: 108rpx;
  height: 108rpx;
  border-radius: 54rpx;
  background: #eef2ff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 48rpx;
}

.username {
  display: block;
  font-size: 34rpx;
  font-weight: 700;
  color: #0f172a;
}

.user-subtitle {
  display: block;
  margin-top: 8rpx;
  font-size: 22rpx;
  color: #64748b;
}

.login-btn,
.logout-btn {
  margin-top: 24rpx;
  height: 80rpx;
  border-radius: 20rpx;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 28rpx;
  font-weight: 600;
}

.login-btn {
  background: #4f46e5;
  color: #ffffff;
}

.logout-btn {
  background: #f8fafc;
  color: #0f172a;
}

.state-card {
  padding: 40rpx;
  text-align: center;
  color: #64748b;
}

.section-card {
  padding: 30rpx;
}

.section-title {
  display: block;
  font-size: 30rpx;
  font-weight: 700;
  color: #0f172a;
  margin-bottom: 20rpx;
}

.summary-row {
  display: flex;
  flex-direction: column;
  gap: 16rpx;
}

.summary-item,
.menu-row {
  display: flex;
  width: 100%;
  justify-content: space-between;
  align-items: center;
  padding: 22rpx 0;
  border: 0;
  border-bottom: 1rpx solid #e2e8f0;
  background: transparent;
  text-align: left;
  line-height: inherit;
}

.summary-item:last-child,
.menu-row:last-child {
  border-bottom: none;
}

.summary-label,
.summary-value,
.menu-title {
  font-size: 26rpx;
}

.summary-label,
.menu-title {
  color: #334155;
}

.summary-value {
  color: #4f46e5;
}

.summary-value.action-link {
  text-decoration: underline;
}

.action-hint {
  display: block;
  margin-top: 4rpx;
  font-size: 20rpx;
  color: #4f46e5;
}

.menu-main {
  display: flex;
  align-items: center;
  gap: 18rpx;
}

.menu-icon {
  font-size: 34rpx;
}

.menu-arrow {
  font-size: 36rpx;
  color: #94a3b8;
}
</style>
