package review

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"

	"gitea.stuhelper.com/StuHelper/StuHelper/internal/pkg/logger"
)

// isASCIIWord 判断词是否仅包含 ASCII 字母（英文词需要词边界匹配）
func isASCIIWord(s string) bool {
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')) {
			return false
		}
	}
	return len(s) > 0
}

// wordMatcher 敏感词匹配器，英文词使用 \b 词边界正则，中文词使用子串匹配
type wordMatcher struct {
	word  string
	regex *regexp.Regexp // 非 nil 表示使用正则匹配（英文词）
}

// Filter 敏感词过滤器
type Filter struct {
	repo         *Repository
	words        []SensitiveWord
	blockMatchers []wordMatcher
	warnMatchers  []wordMatcher
	mu           sync.RWMutex
	lastRefresh  time.Time
	refreshTTL   time.Duration
	sf           singleflight.Group // 去重并发刷新调用
}

// NewFilter 创建敏感词过滤器
func NewFilter(repo *Repository) *Filter {
	return &Filter{
		repo:       repo,
		refreshTTL: 5 * time.Minute,
	}
}

// buildMatcher 根据词内容构建匹配器：纯 ASCII 英文词使用 \b 词边界正则，其他使用子串匹配
func buildMatcher(word string) wordMatcher {
	lowerWord := strings.ToLower(word)
	if isASCIIWord(lowerWord) {
		// 英文词使用 \b 词边界，避免 "ass" 匹配 "class"/"assignment"
		re, err := regexp.Compile(`(?i)\b` + regexp.QuoteMeta(lowerWord) + `\b`)
		if err == nil {
			return wordMatcher{word: lowerWord, regex: re}
		}
		// 正则编译失败时降级为子串匹配
		logger.L().Warn("failed to compile word boundary regex, falling back to substring match",
			zap.String("word", lowerWord), zap.Error(err))
	}
	return wordMatcher{word: lowerWord}
}

// Refresh 刷新敏感词列表
func (f *Filter) Refresh(ctx context.Context) error {
	words, err := f.repo.ListActiveSensitiveWords(ctx)
	if err != nil {
		return err
	}

	blockMatchers := make([]wordMatcher, 0, len(words))
	warnMatchers := make([]wordMatcher, 0, len(words))
	for _, w := range words {
		m := buildMatcher(w.Word)
		if w.Level == "block" {
			blockMatchers = append(blockMatchers, m)
		} else {
			warnMatchers = append(warnMatchers, m)
		}
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.words = words
	f.blockMatchers = blockMatchers
	f.warnMatchers = warnMatchers
	f.lastRefresh = time.Now()
	return nil
}

// ensureFresh 确保敏感词列表是最新的
// 使用 singleflight 去重并发刷新，避免多个 goroutine 同时查询 DB
func (f *Filter) ensureFresh(ctx context.Context) {
	f.mu.RLock()
	needRefresh := time.Since(f.lastRefresh) > f.refreshTTL
	f.mu.RUnlock()

	if !needRefresh {
		return
	}

	// singleflight 确保同一时刻只有一个 goroutine 执行 DB 查询
	_, _, _ = f.sf.Do("refresh", func() (interface{}, error) {
		// 二次检查：进入 singleflight 后再次确认是否仍需刷新
		f.mu.RLock()
		if time.Since(f.lastRefresh) <= f.refreshTTL {
			f.mu.RUnlock()
			return nil, nil
		}
		f.mu.RUnlock()

		// 在锁外执行 DB 查询，避免阻塞所有读操作
		words, err := f.repo.ListActiveSensitiveWords(ctx)
		if err != nil {
			logger.L().Warn("failed to refresh sensitive words", zap.Error(err))
			return nil, err
		}

		// 预构建新的匹配器列表，减少写锁持有时间
		newBlock := make([]wordMatcher, 0, len(words))
		newWarn := make([]wordMatcher, 0, len(words))
		for _, w := range words {
			m := buildMatcher(w.Word)
			if w.Level == "block" {
				newBlock = append(newBlock, m)
			} else {
				newWarn = append(newWarn, m)
			}
		}

		f.mu.Lock()
		f.words = words
		f.blockMatchers = newBlock
		f.warnMatchers = newWarn
		f.lastRefresh = time.Now()
		f.mu.Unlock()

		return nil, nil
	})
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
	for _, m := range f.blockMatchers {
		if matchWord(m, lowerContent, content) {
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
	for _, m := range f.warnMatchers {
		if matchWord(m, lowerContent, content) {
			result.Level = "warn"
			result.MatchCount++
		}
	}

	return result
}

// matchWord 根据匹配器类型执行匹配：正则（英文词边界）或子串（中文等）
func matchWord(m wordMatcher, lowerContent, originalContent string) bool {
	if m.regex != nil {
		return m.regex.MatchString(originalContent)
	}
	return strings.Contains(lowerContent, m.word)
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
