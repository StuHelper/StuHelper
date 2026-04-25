/**
 * 颜色工具函数
 */

/**
 * 为颜色值添加透明度。
 * 支持 hex (#rrggbb) 和 rgb(...) 格式，返回 rgba 字符串。
 * @param color - CSS 颜色值（hex 或 rgb）
 * @param alpha - 透明度 0~1
 */
export function withAlpha(color: string, alpha: number): string {
  const trimmed = color.trim()

  // hex 格式只接受合法的 3/4/6/8 位十六进制字符
  const hexMatch = trimmed.match(/^#([0-9a-fA-F]{3}|[0-9a-fA-F]{4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$/)
  const matchedHex = hexMatch?.[1]
  if (matchedHex) {
    let hex = matchedHex
    // 短格式 #rgb → #rrggbb, #rgba → #rrggbbaa
    if (hex.length === 3 || hex.length === 4) {
      hex =
        hex.charAt(0) +
        hex.charAt(0) +
        hex.charAt(1) +
        hex.charAt(1) +
        hex.charAt(2) +
        hex.charAt(2)
    }
    // 取前 6 位（忽略已有 alpha）
    const r = parseInt(hex.substring(0, 2), 16)
    const g = parseInt(hex.substring(2, 4), 16)
    const b = parseInt(hex.substring(4, 6), 16)
    return `rgba(${r}, ${g}, ${b}, ${alpha})`
  }

  // rgb(...) 格式
  const rgbMatch = trimmed.match(/^rgb\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)\s*\)$/)
  if (rgbMatch) {
    return `rgba(${rgbMatch[1]}, ${rgbMatch[2]}, ${rgbMatch[3]}, ${alpha})`
  }

  // rgba(...) 格式 — 替换 alpha
  const rgbaMatch = trimmed.match(/^rgba\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)\s*,\s*[\d.]+\s*\)$/)
  if (rgbaMatch) {
    return `rgba(${rgbaMatch[1]}, ${rgbaMatch[2]}, ${rgbaMatch[3]}, ${alpha})`
  }

  // 无法解析时原样返回，避免拼接出无效 CSS 值
  return trimmed
}
