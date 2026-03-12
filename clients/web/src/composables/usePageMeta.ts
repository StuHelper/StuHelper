/**
 * 页面 meta 标签管理
 * 统一更新 document.title、description、og:* 等 meta 标签
 */

const DEFAULT_TITLE = 'StuHelper'

function setMeta(selector: string, attr: string, value: string) {
  const el = document.querySelector(selector)
  if (el) el.setAttribute(attr, value)
}

/**
 * 更新页面 meta 信息
 */
export function updatePageMeta(options: {
  title?: string
  description?: string
}) {
  const { title, description } = options

  // 更新 document.title
  document.title = title ? `${title} - ${DEFAULT_TITLE}` : DEFAULT_TITLE

  // 更新 og:title
  setMeta('meta[property="og:title"]', 'content', title || DEFAULT_TITLE)

  // 更新 description + og:description
  if (description) {
    setMeta('meta[name="description"]', 'content', description)
    setMeta('meta[property="og:description"]', 'content', description)
  }
}

/**
 * 更新 HTML lang 属性和 locale 相关 meta
 */
export function updateLocaleMeta(locale: string, description: string, ogTitle: string) {
  document.documentElement.lang = locale
  setMeta('meta[name="description"]', 'content', description)
  setMeta('meta[property="og:title"]', 'content', ogTitle)
  setMeta('meta[property="og:description"]', 'content', description)
}
