<script setup lang="ts">
import { ref } from 'vue'
import { UNIAPPX_EXPERIMENTAL_NOTICE } from '../../config/featureSurface'

const loading = ref(false)

const handleLogin = async () => {
  if (loading.value) return
  loading.value = true
  try {
    uni.showToast({ title: '实验性脚手架，SSO 暂未开放', icon: 'none' })
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <view class="login-page">
    <view class="login-container">
      <view class="logo-section">
        <text class="logo">🎓</text>
        <text class="app-name">StuHelper</text>
        <text class="slogan">你的智能选课助手</text>
      </view>

      <view class="login-section">
        <view class="notice-card">
          <text class="notice-text">{{ UNIAPPX_EXPERIMENTAL_NOTICE }}</text>
        </view>
        <button class="login-btn" :loading="loading" @tap="handleLogin">
          {{ loading ? '处理中...' : 'SSO 功能暂未开放' }}
        </button>
      </view>
    </view>
  </view>
</template>

<style scoped>
.login-page {
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  display: flex;
  align-items: center;
  justify-content: center;
}

.login-container {
  width: 100%;
  padding: 80rpx 60rpx;
}

.logo-section {
  text-align: center;
  margin-bottom: 120rpx;
}

.logo {
  font-size: 120rpx;
  display: block;
  margin-bottom: 32rpx;
}

.app-name {
  font-size: 64rpx;
  font-weight: 700;
  color: #FFFFFF;
  display: block;
  margin-bottom: 16rpx;
}

.slogan {
  font-size: 28rpx;
  color: rgba(255, 255, 255, 0.9);
  display: block;
}

.login-section {
  padding: 0 40rpx;
}

.notice-card {
  margin-bottom: 32rpx;
  padding: 24rpx 28rpx;
  background: rgba(255, 255, 255, 0.18);
  border: 2rpx solid rgba(255, 255, 255, 0.28);
  border-radius: 20rpx;
}

.notice-text {
  color: #FFFFFF;
  font-size: 24rpx;
  line-height: 1.6;
  text-align: center;
  display: block;
}

.login-btn {
  width: 100%;
  height: 96rpx;
  background: #FFFFFF;
  color: #667eea;
  border-radius: 48rpx;
  font-size: 32rpx;
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 8rpx 24rpx rgba(0, 0, 0, 0.15);
}
</style>
