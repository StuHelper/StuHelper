import './styles/main.css'
import 'element-plus/es/components/message/style/css'
import 'element-plus/es/components/config-provider/style/css'
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import router from './router'
import i18n from './i18n'
import enUS from './i18n/locales/en-US'
import zhCN from './i18n/locales/zh-CN'
import App from './App.vue'
import { useAuthStore } from './stores/auth'
import { vRipple } from './directives/ripple'
import { initObservability } from './utils/observability'

const app = createApp(App)
const pinia = createPinia()

function renderBootstrapFallback(error: unknown) {
  const root = document.getElementById('app')
  if (!root) return
  const locale = i18n.global.locale.value
  const messages = locale === 'zh-CN' ? zhCN : enUS

  root.innerHTML = `
    <div style="min-height:100vh;display:flex;align-items:center;justify-content:center;padding:24px;background:#f8fafc;color:#0f172a;font-family:Inter,-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;">
      <div style="max-width:480px;border:1px solid rgba(15,23,42,0.08);border-radius:20px;padding:24px;background:#ffffff;box-shadow:0 12px 40px rgba(15,23,42,0.08);">
        <h1 style="margin:0 0 12px;font-size:20px;font-weight:800;">StuHelper</h1>
        <p style="margin:0 0 8px;font-size:14px;line-height:1.7;">${messages.common.bootstrap.failed}</p>
        <p style="margin:0;font-size:12px;line-height:1.6;color:#64748b;">${messages.common.bootstrap.contactSupport}</p>
      </div>
    </div>
  `
  console.error('[App] bootstrap failed:', error)
}

// 全局错误处理：未捕获的组件错误在此统一处理
app.config.errorHandler = (err, instance, info) => {
  const componentName =
    instance?.$options?.name ?? instance?.$options?.__name ?? 'Unknown'
  if (import.meta.env.DEV) {
    console.error(`[Vue Error] ${componentName} — ${info}:`, err)
  }
}

app.use(pinia)
app.use(i18n)
app.directive('ripple', vRipple)
initObservability()

async function bootstrapApp() {
  const authStore = useAuthStore(pinia)
  await authStore.bootstrapSession()
  app.use(router)
  await router.isReady()
  app.mount('#app')
}

bootstrapApp().catch(renderBootstrapFallback)
