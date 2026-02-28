// 评分等级转文字 (1-5 五级制)
export const ratingToText = (value: number, t: (key: string) => string): string => {
  const texts: Record<number, string> = {
    5: t('review.rating.level5'),
    4: t('review.rating.level4'),
    3: t('review.rating.level3'),
    2: t('review.rating.level2'),
    1: t('review.rating.level1')
  }
  return texts[value] || t('review.rating.unknown')
}

// 评分转颜色 (1-5 五级制，对齐 CSS 变量 --color-rating-1 ~ --color-rating-5)
// 硬编码颜色值，用于 JS 上下文（如 ECharts）。
// 模板中应使用 CSS 变量 var(--color-rating-*) 替代。
const ratingColors: Record<number, string> = {
  1: '#f56c6c',
  2: '#e6a23c',
  3: '#909399',
  4: '#67c23a',
  5: '#529b2e'
}
export const ratingToColor = (value: number): string => {
  return ratingColors[Math.round(value)] || '#909399'
}

// 计算综合评分 (1-5 转换为 0-10，输入 clamp 到 [1, 5])
export const normalizeRating = (value: number): number => {
  const clamped = Math.max(1, Math.min(5, value))
  return ((clamped - 1) / 4) * 10
}
