package review

import (
	"context"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

// Filter 敏感词过滤器
type Filter struct {
	repo        *Repository
	words       []SensitiveWord
	blockWords  map[string]bool
	warnWords   map[string]bool
	mu          sync.RWMutex
	lastRefresh time.Time
	refreshTTL  time.Duration
}

// NewFilter 创建敏感词过滤器
func NewFilter(repo *Repository) *Filter {
	return &Filter{
		repo:       repo,
		blockWords: make(map[string]bool),
		warnWords:  make(map[string]bool),
		refreshTTL: 5 * time.Minute,
	}
}

// Refresh 刷新敏感词列表
func (f *Filter) Refresh(ctx context.Context) error {
	words, err := f.repo.ListActiveSensitiveWords(ctx)
	if err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.words = words
	f.blockWords = make(map[string]bool)
	f.warnWords = make(map[string]bool)

	for _, w := range words {
		lowerWord := strings.ToLower(w.Word)
		if w.Level == "block" {
			f.blockWords[lowerWord] = true
		} else {
			f.warnWords[lowerWord] = true
		}
	}

	f.lastRefresh = time.Now()
	return nil
}

// ensureFresh 确保敏感词列表是最新的
// 使用写锁内二次检查防止并发刷新，在锁内直接执行刷新逻辑
func (f *Filter) ensureFresh(ctx context.Context) {
	f.mu.RLock()
	needRefresh := time.Since(f.lastRefresh) > f.refreshTTL
	f.mu.RUnlock()

	if needRefresh {
		f.mu.Lock()
		defer f.mu.Unlock()
		// 二次检查：获取写锁后再次确认是否仍需刷新
		if time.Since(f.lastRefresh) > f.refreshTTL {
			words, err := f.repo.ListActiveSensitiveWords(ctx)
			if err != nil {
				logger.L().Warn("failed to refresh sensitive words", zap.Error(err))
				return
			}
			f.words = words
			f.blockWords = make(map[string]bool)
			f.warnWords = make(map[string]bool)
			for _, w := range words {
				lowerWord := strings.ToLower(w.Word)
				if w.Level == "block" {
					f.blockWords[lowerWord] = true
				} else {
					f.warnWords[lowerWord] = true
				}
			}
			f.lastRefresh = time.Now()
		}
	}
}

// CheckContent 检查内容是否包含敏感词
func (f *Filter) CheckContent(ctx context.Context, content string) *ContentCheckResult {
	f.ensureFresh(ctx)

	result := &ContentCheckResult{
		IsValid: true,
	}

	lowerContent := strings.ToLower(content)

	f.mu.RLock()
	defer f.mu.RUnlock()

	// 检查阻止级别的敏感词
	for word := range f.blockWords {
		if strings.Contains(lowerContent, word) {
			result.IsValid = false
			result.Level = "block"
			result.MatchCount++
		}
	}

	// 如果已经被阻止，直接返回
	if !result.IsValid {
		return result
	}

	// 检查警告级别的敏感词
	for word := range f.warnWords {
		if strings.Contains(lowerContent, word) {
			result.Level = "warn"
			result.MatchCount++
		}
	}

	return result
}

// ContainsBlockedWord 检查是否包含阻止级别的敏感词
func (f *Filter) ContainsBlockedWord(ctx context.Context, content string) bool {
	result := f.CheckContent(ctx, content)
	return !result.IsValid
}

// QualityCheckResult 内容质量检查结果
type QualityCheckResult struct {
	Score       int      `json:"score"`       // 质量分数 0-100
	Suggestions []string `json:"suggestions"` // 改进建议
}

// CheckQuality 检查内容质量
func (f *Filter) CheckQuality(content string) *QualityCheckResult {
	result := &QualityCheckResult{
		Score:       100,
		Suggestions: []string{},
	}

	// 去除空白字符后的长度
	trimmed := strings.TrimSpace(content)
	length := len([]rune(trimmed))

	// 检查内容长度
	if length < 10 {
		result.Score -= 30
		result.Suggestions = append(result.Suggestions, "content_too_short")
	} else if length < 50 {
		result.Score -= 10
		result.Suggestions = append(result.Suggestions, "content_short")
	}

	// 检查是否包含换行（段落结构）
	if length > 100 && !strings.Contains(content, "\n") {
		result.Score -= 10
		result.Suggestions = append(result.Suggestions, "content_lacks_paragraphs")
	}

	// 检查是否全是标点或特殊字符
	alphaCount := 0
	for _, r := range trimmed {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '\u4e00' && r <= '\u9fff') { // 中文字符
			alphaCount++
		}
	}
	if length > 0 && float64(alphaCount)/float64(length) < 0.5 {
		result.Score -= 20
		result.Suggestions = append(result.Suggestions, "low_meaningful_content_ratio")
	}

	// 检查重复字符
	if hasExcessiveRepetition(trimmed) {
		result.Score -= 15
		result.Suggestions = append(result.Suggestions, "excessive_repetition")
	}

	// 确保分数不低于0
	if result.Score < 0 {
		result.Score = 0
	}

	return result
}

// hasExcessiveRepetition 检查是否有过多重复字符
func hasExcessiveRepetition(s string) bool {
	runes := []rune(s)
	if len(runes) < 5 {
		return false
	}

	repeatCount := 1
	for i := 1; i < len(runes); i++ {
		if runes[i] == runes[i-1] {
			repeatCount++
			if repeatCount >= 5 {
				return true
			}
		} else {
			repeatCount = 1
		}
	}
	return false
}
