import { createApp } from 'vue'
import { createPinia } from 'pinia'
import 'element-plus/es/components/message/style/css'
import 'element-plus/es/components/config-provider/style/css'
import App from './App.vue'
import router from './router'
import i18n from './i18n'

const app = createApp(App)

app.use(createPinia())
app.use(router)
app.use(i18n)

app.mount('#app')

// Web Vitals 性能监控（异步加载，不阻塞首屏）
import { reportWebVitals } from '@/utils/webVitals'
reportWebVitals()
