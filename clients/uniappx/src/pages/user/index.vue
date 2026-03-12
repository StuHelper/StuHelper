<script setup lang="ts">
import { ref } from 'vue'
import { UNIAPPX_EXPERIMENTAL_NOTICE, USER_MENU_ITEMS } from '../../config/featureSurface'

const userInfo = ref({
  name: '未登录',
  avatar: '',
  isLogin: false
})

const menuItems = USER_MENU_ITEMS

const navigateTo = (path: string) => {
  if (!userInfo.value.isLogin && path !== '/pages/auth/login') {
    uni.navigateTo({ url: '/pages/auth/login' })
    return
  }
  uni.navigateTo({ url: path })
}

const handleLogin = () => {
  uni.navigateTo({ url: '/pages/auth/login' })
}
</script>

<template>
  <view class="user-page">
    <!-- User Header -->
    <view class="user-header">
      <view class="user-info">
        <image v-if="userInfo.avatar" :src="userInfo.avatar" class="avatar" />
        <view v-else class="avatar-placeholder">👤</view>
        <text class="username">{{ userInfo.name }}</text>
      </view>
      <button v-if="!userInfo.isLogin" class="login-btn" @tap="handleLogin">
        登录
      </button>
    </view>

    <view class="notice-banner">
      <text class="notice-banner-text">{{ UNIAPPX_EXPERIMENTAL_NOTICE }}</text>
    </view>

    <!-- Menu List -->
    <view class="menu-list">
      <view
        v-for="item in menuItems"
        :key="item.title"
        class="menu-item"
        @tap="navigateTo(item.path)"
      >
        <view class="menu-left">
          <text class="menu-icon">{{ item.icon }}</text>
          <text class="menu-title">{{ item.title }}</text>
        </view>
        <text class="menu-arrow">›</text>
      </view>
    </view>
  </view>
</template>

<style scoped>
.user-page {
  min-height: 100vh;
  background: #F8F9FA;
}

.user-header {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 80rpx 40rpx 60rpx;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.user-info {
  display: flex;
  align-items: center;
}

.avatar,
.avatar-placeholder {
  width: 120rpx;
  height: 120rpx;
  border-radius: 60rpx;
  margin-right: 24rpx;
}

.avatar-placeholder {
  background: rgba(255, 255, 255, 0.3);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 64rpx;
}

.username {
  font-size: 36rpx;
  font-weight: 600;
  color: #FFFFFF;
}

.login-btn {
  background: rgba(255, 255, 255, 0.2);
  color: #FFFFFF;
  border: 2rpx solid rgba(255, 255, 255, 0.5);
  border-radius: 40rpx;
  padding: 16rpx 48rpx;
  font-size: 28rpx;
}

.notice-banner {
  margin: 24rpx 24rpx 0;
  padding: 20rpx 24rpx;
  background: #FFF7ED;
  border: 2rpx solid #FED7AA;
  border-radius: 16rpx;
}

.notice-banner-text {
  color: #9A3412;
  font-size: 24rpx;
  line-height: 1.6;
}

.menu-list {
  margin-top: 24rpx;
  background: #FFFFFF;
}

.menu-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 32rpx 40rpx;
  border-bottom: 1rpx solid #F3F4F6;
}

.menu-left {
  display: flex;
  align-items: center;
}

.menu-icon {
  font-size: 40rpx;
  margin-right: 24rpx;
}

.menu-title {
  font-size: 32rpx;
  color: #1F2937;
}

.menu-arrow {
  font-size: 48rpx;
  color: #D1D5DB;
}
</style>
