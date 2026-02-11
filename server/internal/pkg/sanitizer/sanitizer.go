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
)

// 危险 HTML 标签列表
var dangerousTags = []string{
	"object", "embed", "base", "form", "svg", "math", "link", "meta",
}

// SanitizeText 清理纯文本内容，移除所有 HTML 标签
// 注意：不做 html.EscapeString，因为前端模板（Vue）会自动转义，
// 双重编码会导致用户看到 &amp; 而非 &
func SanitizeText(s string) string {
	// 移除所有 HTML 标签
	s = htmlTagRegex.ReplaceAllString(s, "")
	// 规范化空白字符
	s = multiSpaceRegex.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// SanitizeTitle 清理标题，更严格的清理
func SanitizeTitle(s string) string {
	s = SanitizeText(s)
	// 标题不允许换行
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return strings.TrimSpace(s)
}

// ContainsDangerousContent 检查是否包含危险内容
// 先解码 HTML 实体并移除零宽字符，防止编码绕过
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
	if strings.Contains(lower, "data:") {
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
