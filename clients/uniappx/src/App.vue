<script setup lang="ts">
import { onLaunch, onShow } from '@dcloudio/uni-app'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()

function bootstrapAuthSession(): void {
  void authStore.bootstrapSession().catch((error) => {
    console.error('[uniappx] auth bootstrap failed', error)
  })
}

/**
 * 解析 deep link URL 中的查询参数
 * 支持 stuhelper://auth/callback?code=xxx&state=yyy
 */
function parseDeepLinkParams(urlString: string): Record<string, string> {
  const params: Record<string, string> = {}
  const qIndex = urlString.indexOf('?')
  if (qIndex < 0) return params
  const queryString = urlString.slice(qIndex + 1)
  for (const pair of queryString.split('&')) {
    const eqIndex = pair.indexOf('=')
    if (eqIndex < 0) continue
    const key = decodeURIComponent(pair.slice(0, eqIndex))
    const value = decodeURIComponent(pair.slice(eqIndex + 1))
    params[key] = value
  }
  return params
}

/**
 * 处理 SSO deep link 回调
 * 从 URL 中提取 code 和 state，跳转到回调页面完成 token 交换
 */
function handleDeepLink(urlString: string): void {
  if (!urlString || !urlString.startsWith('stuhelper://auth/callback')) return

  const params = parseDeepLinkParams(urlString)
  const code = params.code ?? ''
  const state = params.state ?? ''

  if (!code || !state) return

  // 跳转到回调页面处理 token 交换
  uni.navigateTo({
    url: `/pages/auth/callback?code=${encodeURIComponent(code)}&state=${encodeURIComponent(state)}`,
  })
}

onLaunch((options) => {
  bootstrapAuthSession()

  // 冷启动：通过 scheme 唤起 App 的场景
  const launchUrl = (options as { path?: string; scene?: number })?.path
  if (typeof launchUrl === 'string' && launchUrl.includes('stuhelper://')) {
    handleDeepLink(launchUrl)
    return
  }

  // 原生 App 监听 scheme URL
  if (typeof plus !== 'undefined') {
    plus.runtime.arguments && handleDeepLink(plus.runtime.arguments as string)
  }
})

onShow(() => {
  if (!authStore.initialized) return
  bootstrapAuthSession()

  // 热启动：App 在后台被 scheme 唤起
  if (typeof plus !== 'undefined' && plus.runtime.arguments) {
    const args = plus.runtime.arguments as string
    if (args.startsWith('stuhelper://auth/callback')) {
      handleDeepLink(args)
      // 清除 arguments 防止重复处理
      plus.runtime.arguments = ''
    }
  }
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
