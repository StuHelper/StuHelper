// 评分等级转文字
export const ratingToText = (value: number): string => {
  const texts: Record<number, string> = {
    2: '强烈推荐',
    1: '推荐',
    0: '一般',
    [-1]: '不推荐',
    [-2]: '强烈不推荐'
  }
  return texts[value] || '未知'
}

// 评分转颜色
export const ratingToColor = (value: number): string => {
  if (value >= 1) return '#67c23a'
  if (value === 0) return '#909399'
  return '#f56c6c'
}

// 计算综合评分 (-2 ~ 2 转换为 0 ~ 10)
export const normalizeRating = (value: number): number => {
  return ((value + 2) / 4) * 10
}
