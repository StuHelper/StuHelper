import { createSSRApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import { bootstrapLocale } from './i18n'

export function createApp() {
  bootstrapLocale()
  const app = createSSRApp(App)
  const pinia = createPinia()
  app.use(pinia)
  return { app }
}
