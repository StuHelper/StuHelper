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

// 评分转颜色 (1-5 五级制)
// 硬编码颜色值，用于 JS 上下文（如 ECharts）。
// 模板中应使用 CSS 变量 var(--color-rating-*) 替代。
export const ratingToColor = (value: number): string => {
  if (value >= 4) return '#67c23a'
  if (value === 3) return '#909399'
  return '#f56c6c'
}

// 计算综合评分 (1-5 转换为 0-10)
export const normalizeRating = (value: number): number => {
  return ((value - 1) / 4) * 10
}
