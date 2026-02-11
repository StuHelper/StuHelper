// 评分等级转文字 (1-5 五级制)
export const ratingToText = (value: number): string => {
  const texts: Record<number, string> = {
    5: '强烈推荐',
    4: '推荐',
    3: '一般',
    2: '不推荐',
    1: '强烈不推荐'
  }
  return texts[value] || '未知'
}

// 评分转颜色 (1-5 五级制)
export const ratingToColor = (value: number): string => {
  if (value >= 4) return '#67c23a'
  if (value === 3) return '#909399'
  return '#f56c6c'
}

// 计算综合评分 (1-5 转换为 0-10)
export const normalizeRating = (value: number): number => {
  return ((value - 1) / 4) * 10
}
