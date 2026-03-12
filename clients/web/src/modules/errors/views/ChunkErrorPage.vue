<template>
  <div style="min-height:100vh;display:flex;align-items:center;justify-content:center;padding:24px;font-family:system-ui,-apple-system,sans-serif;background:#f8fafc">
    <div style="text-align:center;max-width:400px">
      <svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="#94a3b8" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" style="margin:0 auto 24px">
        <circle cx="12" cy="12" r="10" />
        <line x1="12" y1="8" x2="12" y2="12" />
        <line x1="12" y1="16" x2="12.01" y2="16" />
      </svg>
      <h1 style="font-size:20px;font-weight:700;color:#1e293b;margin:0 0 12px">{{ isZh ? '页面加载失败' : 'Page Load Failed' }}</h1>
      <p style="font-size:14px;color:#64748b;margin:0 0 24px">{{ isZh ? '应用已更新，请刷新页面以加载最新版本。' : 'The app has been updated. Please refresh the page to load the latest version.' }}</p>
      <button
        style="display:inline-flex;align-items:center;gap:8px;padding:10px 24px;background:linear-gradient(135deg,#6366f1,#8b5cf6);color:white;font-size:14px;font-weight:500;border:none;border-radius:9999px;cursor:pointer"
        @click="reload"
      >
        {{ isZh ? '刷新页面' : 'Refresh Page' }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
const isZh = navigator.language.startsWith('zh')

const reload = () => {
  window.location.reload()
}
</script>
