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
	// 匹配 data URL
	dataURLRegex = regexp.MustCompile(`(?i)data:`)
	// 匹配多余空白
	multiSpaceRegex = regexp.MustCompile(`\s+`)
)

// SanitizeText 清理纯文本内容，移除所有 HTML 标签并转义特殊字符
func SanitizeText(s string) string {
	// 移除所有 HTML 标签
	s = htmlTagRegex.ReplaceAllString(s, "")
	// HTML 转义特殊字符
	s = html.EscapeString(s)
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
func ContainsDangerousContent(s string) bool {
	lower := strings.ToLower(s)
	// 检查 script 标签
	if strings.Contains(lower, "<script") {
		return true
	}
	// 检查 JavaScript 事件处理器
	if jsEventRegex.MatchString(s) {
		return true
	}
	// 检查 JavaScript URL
	if jsURLRegex.MatchString(s) {
		return true
	}
	// 检查 iframe
	if strings.Contains(lower, "<iframe") {
		return true
	}
	// 检查 object/embed
	if strings.Contains(lower, "<object") || strings.Contains(lower, "<embed") {
		return true
	}
	return false
}
