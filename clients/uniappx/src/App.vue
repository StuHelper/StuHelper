<script setup lang="ts">
import { onLaunch, onShow } from '@dcloudio/uni-app'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()

onLaunch(() => {
  void authStore.bootstrapSession()
})

onShow(() => {
  if (!authStore.initialized) return
  void authStore.bootstrapSession()
})
</script>

<template>
  <view>
    <slot />
  </view>
</template>

<style>
page {
  background-color: #f8fafc;
  color: #0f172a;
}
</style>
