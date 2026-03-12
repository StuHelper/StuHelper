<script setup lang="ts">
import { onLaunch } from '@dcloudio/uni-app'
import { useAuthStore } from '@/stores/auth'

onLaunch(() => {
  const authStore = useAuthStore()
  authStore.initAuth()
})
</script>

<template>
  <view>
    <slot />
  </view>
</template>

<style>
page {
  background-color: #F8F9FA;
}
</style>
