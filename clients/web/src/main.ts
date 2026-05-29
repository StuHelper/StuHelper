import './styles/main.css'
import 'element-plus/es/components/message/style/css'
import 'element-plus/es/components/config-provider/style/css'
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import router, { hasPendingExternalLocationRedirect } from './router'
import i18n from './i18n'
import enUS from './i18n/locales/en-US'
import zhCN from './i18n/locales/zh-CN'
import App from './App.vue'
import { useAuthStore } from './stores/auth'
import { useLocaleStore } from './stores/locale'
import { vRipple } from './directives/ripple'
import { initObservability } from './utils/observability'
import { hasStoredSessionHint } from './utils/sessionHint'

const app = createApp(App)
const pinia = createPinia()

function renderBootstrapFallback(error: unknown) {
  const root = document.getElementById('app')
  if (!root) return
  const locale = i18n.global.locale.value
  const messages = locale === 'zh-CN' ? zhCN : enUS

  root.replaceChildren()
  const wrapper = document.createElement('div')
  wrapper.style.cssText = 'min-height:100vh;display:flex;align-items:center;justify-content:center;padding:24px;background:#f8fafc;color:#0f172a;font-family:Inter,-apple-system,BlinkMacSystemFont,\'Segoe UI\',sans-serif;'
  const card = document.createElement('div')
  card.style.cssText = 'max-width:480px;border:1px solid rgba(15,23,42,0.08);border-radius:20px;padding:24px;background:#ffffff;box-shadow:0 12px 40px rgba(15,23,42,0.08);'
  const h1 = document.createElement('h1')
  h1.style.cssText = 'margin:0 0 12px;font-size:20px;font-weight:800;'
  h1.textContent = 'StuHelper'
  const p1 = document.createElement('p')
  p1.style.cssText = 'margin:0 0 8px;font-size:14px;line-height:1.7;'
  p1.textContent = messages.common.bootstrap.failed
  const p2 = document.createElement('p')
  p2.style.cssText = 'margin:0;font-size:12px;line-height:1.6;color:#64748b;'
  p2.textContent = messages.common.bootstrap.contactSupport
  card.append(h1, p1, p2)
  wrapper.appendChild(card)
  root.appendChild(wrapper)

  if (import.meta.env.DEV) {
    console.error('[App] bootstrap failed:', error)
  }
}

function isExpectedExternalRedirectAbort(error: unknown) {
  if (!hasPendingExternalLocationRedirect()) {
    return false
  }

  if (import.meta.env.DEV) {
    console.debug('[App] router startup interrupted by external redirect:', error)
  }
  return true
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
if (import.meta.env.VITE_E2E_API_STUB !== '1') {
  initObservability()
}

async function bootstrapApp() {
  useLocaleStore(pinia)
  const authStore = useAuthStore(pinia)
  const hasSessionHint = hasStoredSessionHint()
  if (!hasSessionHint && authStore.isAuthenticated) {
    authStore.clearSession()
  }
  app.use(router)
  try {
    await router.isReady()
  } catch (error) {
    if (isExpectedExternalRedirectAbort(error)) {
      return
    }
    throw error
  }
  app.mount('#app')
  if (hasSessionHint) {
    void authStore.bootstrapSession()
  }
}

bootstrapApp().catch(renderBootstrapFallback)
