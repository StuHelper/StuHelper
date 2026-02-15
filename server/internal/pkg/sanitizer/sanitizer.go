package sanitizer

import (
	"html"
	"regexp"
	"strings"
)

var (
	// 匹配 HTML 标签
	htmlTagRegex = regexp.MustCompile(`<[^>]*>`)
	// 匹配 JavaScript 事件处理器
	jsEventRegex = regexp.MustCompile(`(?i)on\w+\s*=`)
	// 匹配 JavaScript URL
	jsURLRegex = regexp.MustCompile(`(?i)javascript:`)
	// 匹配多余空白
	multiSpaceRegex = regexp.MustCompile(`\s+`)
	// 匹配零宽字符（防止绕过检测）
	zeroWidthRegex = regexp.MustCompile(`[\x{200B}\x{200C}\x{200D}\x{FEFF}\x{00AD}]`)
	// 匹配 data: URI（仅匹配 data: 后跟 MIME 类型模式，避免误报 "the data: shows..."）
	dataURIRegex = regexp.MustCompile(`(?i)data:\s*\w+/\w+`)
)

// 危险 HTML 标签列表
var dangerousTags = []string{
	"object", "embed", "base", "form", "svg", "math", "link", "meta",
}

// SanitizeText 清理纯文本内容，移除所有 HTML 标签并解码实体。
// 先移除标签，再解码实体（防止 &lt;script&gt; 等编码绕过），最后二次移除标签。
//
// 行为说明：
//   - 空字符串输入返回空字符串
//   - 不限制输出长度（调用方需自行截断，或使用 SanitizeTextWithLimit）
//   - 保留换行符（\n），但连续空白被合并为单个空格
//   - 输出已 TrimSpace，首尾无空白
func SanitizeText(s string) string {
	// 第一遍：移除明文 HTML 标签
	s = htmlTagRegex.ReplaceAllString(s, "")
	// 解码 HTML 实体（&lt; → <, &amp; → & 等），确保非 Vue 上下文安全
	s = html.UnescapeString(s)
	// 第二遍：移除实体解码后可能出现的标签
	s = htmlTagRegex.ReplaceAllString(s, "")
	// 规范化空白字符
	s = multiSpaceRegex.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// SanitizeTextWithLimit 清理纯文本内容并截断到指定最大长度（按 rune 计数，保证 UTF-8 安全）。
// maxLen <= 0 时不截断，等同于 SanitizeText。
func SanitizeTextWithLimit(s string, maxLen int) string {
	s = SanitizeText(s)
	if maxLen <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) > maxLen {
		return string(runes[:maxLen])
	}
	return s
}

// SanitizeTitle 清理标题，比 SanitizeText 更严格。
//
// 行为说明：
//   - 在 SanitizeText 基础上额外移除换行符（\n \r → 空格）
//   - 不限制输出长度（调用方需自行截断，如 100 字符）
//   - 输出已 TrimSpace，首尾无空白
func SanitizeTitle(s string) string {
	s = SanitizeText(s)
	// 标题不允许换行
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return strings.TrimSpace(s)
}

// ContainsDangerousContent 检查是否包含危险内容（XSS 向量）。
// 先解码 HTML 实体并移除零宽字符，防止编码绕过。
//
// 检测范围：
//   - <script>、<iframe>、<object>、<embed>、<base>、<form>、<svg>、<math>、<link>、<meta> 标签
//   - JavaScript 事件处理器（onclick= 等）
//   - javascript: URL scheme
//   - data: URI（data:text/html 等 MIME 类型模式）
//
// 返回 true 表示内容包含危险向量，应拒绝写入。
func ContainsDangerousContent(s string) bool {
	// 解码 HTML 实体（防止 &#60;script 等绕过）
	decoded := html.UnescapeString(s)
	// 移除零宽字符（防止 ja\u200Bvascript: 等绕过）
	decoded = zeroWidthRegex.ReplaceAllString(decoded, "")
	lower := strings.ToLower(decoded)

	// 检查 script 标签
	if strings.Contains(lower, "<script") {
		return true
	}
	// 检查 JavaScript 事件处理器
	if jsEventRegex.MatchString(decoded) {
		return true
	}
	// 检查 JavaScript URL
	if jsURLRegex.MatchString(decoded) {
		return true
	}
	// 检查 iframe
	if strings.Contains(lower, "<iframe") {
		return true
	}
	// 检查 data: URI（防止 data:text/html,<script>... 等 XSS 向量）
	if dataURIRegex.MatchString(decoded) {
		return true
	}
	// 检查 object/embed/base/form/svg
	for _, tag := range dangerousTags {
		if strings.Contains(lower, "<"+tag) {
			return true
		}
	}
	return false
}
