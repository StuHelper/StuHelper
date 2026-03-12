import './styles/main.css'
import 'element-plus/es/components/message/style/css'
import 'element-plus/es/components/config-provider/style/css'
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import router from './router'
import i18n from './i18n'
import App from './App.vue'
import { useAuthStore } from './stores/auth'

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(i18n)

async function bootstrapApp() {
  const authStore = useAuthStore(pinia)
  await authStore.bootstrapSession()
  app.use(router)
  await router.isReady()
  app.mount('#app')
}

void bootstrapApp()
