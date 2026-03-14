/**
 * Web Vitals 性能监控
 * 收集 Core Web Vitals 指标（CLS、INP、LCP）及辅助指标（FCP、TTFB）
 */
import type { Metric } from 'web-vitals'

function sendMetric(metric: Metric) {
  if (import.meta.env.DEV) {
    const label = metric.rating === 'good' ? '✅' : metric.rating === 'needs-improvement' ? '⚠️' : '❌'
    console.info(`[WebVitals] ${label} ${metric.name}: ${Math.round(metric.value)}ms (${metric.rating})`)
    return
  }

  // 生产环境：通过 sendBeacon 异步上报，不阻塞页面卸载
  const body = JSON.stringify({
    name: metric.name,
    value: metric.value,
    rating: metric.rating,
    delta: metric.delta,
    id: metric.id,
    navigationType: metric.navigationType
  })

  if (navigator.sendBeacon) {
    navigator.sendBeacon('/api/v1/metrics/vitals', body)
  }
}

/**
 * 初始化 Web Vitals 监控
 * 动态导入 web-vitals 避免阻塞首屏渲染
 */
export function reportWebVitals() {
  import('web-vitals').then(({ onCLS, onINP, onLCP, onFCP, onTTFB }) => {
    onCLS(sendMetric)
    onINP(sendMetric)
    onLCP(sendMetric)
    onFCP(sendMetric)
    onTTFB(sendMetric)
  })
}
