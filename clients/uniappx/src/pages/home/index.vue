<script setup lang="ts">
import { HOME_FEATURES, UNIAPPX_EXPERIMENTAL_NOTICE } from '../../config/featureSurface'

const features = HOME_FEATURES

const navigateTo = (path: string) => {
  uni.navigateTo({ url: path })
}
</script>

<template>
  <view class="home-page">
    <!-- Hero Section -->
    <view class="hero">
      <view class="hero-content">
        <text class="hero-title">StuHelper</text>
        <text class="hero-subtitle">你的智能选课助手</text>
      </view>
    </view>

    <view class="notice">
      <text class="notice-text">{{ UNIAPPX_EXPERIMENTAL_NOTICE }}</text>
    </view>

    <!-- Features Grid -->
    <view class="features">
      <view
        v-for="feature in features"
        :key="feature.title"
        class="feature-card"
        @tap="navigateTo(feature.path)"
      >
        <text class="feature-icon">{{ feature.icon }}</text>
        <text class="feature-title">{{ feature.title }}</text>
        <text class="feature-desc">{{ feature.desc }}</text>
      </view>
    </view>
  </view>
</template>

<style scoped>
.home-page {
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.hero {
  padding: 120rpx 40rpx 80rpx;
  text-align: center;
}

.hero-content {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.hero-title {
  font-size: 64rpx;
  font-weight: 700;
  color: #FFFFFF;
  margin-bottom: 20rpx;
}

.hero-subtitle {
  font-size: 32rpx;
  color: rgba(255, 255, 255, 0.9);
}

.notice {
  margin: 0 40rpx 24rpx;
  padding: 24rpx 28rpx;
  background: rgba(255, 255, 255, 0.18);
  border: 2rpx solid rgba(255, 255, 255, 0.25);
  border-radius: 20rpx;
}

.notice-text {
  color: #FFFFFF;
  font-size: 24rpx;
  line-height: 1.6;
}

.features {
  padding: 40rpx;
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 24rpx;
}

.feature-card {
  background: rgba(255, 255, 255, 0.95);
  border-radius: 24rpx;
  padding: 40rpx 24rpx;
  display: flex;
  flex-direction: column;
  align-items: center;
  box-shadow: 0 8rpx 24rpx rgba(0, 0, 0, 0.1);
}

.feature-icon {
  font-size: 64rpx;
  margin-bottom: 16rpx;
}

.feature-title {
  font-size: 32rpx;
  font-weight: 600;
  color: #1F2937;
  margin-bottom: 8rpx;
}

.feature-desc {
  font-size: 24rpx;
  color: #6B7280;
  text-align: center;
}
</style>
