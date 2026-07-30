package review

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"

	"github.com/StuHelper/StuHelper/server/internal/pkg/logger"
	"github.com/StuHelper/StuHelper/server/internal/pkg/singleflightx"
)

// isASCIIWord 判断词是否仅包含 ASCII 字母（英文词需要词边界匹配）
func isASCIIWord(s string) bool {
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
			return false
		}
	}
	return s != ""
}

// wordMatcher 敏感词匹配器，英文词使用 \b 词边界正则，中文词使用子串匹配
type wordMatcher struct {
	word  string
	regex *regexp.Regexp // 非 nil 表示使用正则匹配（英文词）
}

// Filter 敏感词过滤器
type Filter struct {
	repo           *Repository
	words          []SensitiveWord
	blockMatchers  []wordMatcher
	warnMatchers   []wordMatcher
	reviewMatchers []wordMatcher
	mu             sync.RWMutex
	refreshMu      sync.Mutex
	lastRefresh    time.Time
	refreshTTL     time.Duration
	sf             singleflight.Group // 去重并发刷新调用
}

// NewFilter 创建敏感词过滤器
func NewFilter(repo *Repository) *Filter {
	if repo == nil {
		panic("review.NewFilter: repo must not be nil")
	}
	return &Filter{
		repo:       repo,
		refreshTTL: 5 * time.Minute,
	}
}

// buildMatcher 根据词内容构建匹配器：纯 ASCII 英文词使用 \b 词边界正则，其他使用子串匹配
func buildMatcher(word string) wordMatcher {
	lowerWord := strings.ToLower(word)
	if !isASCIIWord(lowerWord) {
		return wordMatcher{word: lowerWord}
	}
	// 模式完全由固定前缀和 QuoteMeta 输出组成，这里应始终可编译。
	return wordMatcher{
		word:  lowerWord,
		regex: regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(lowerWord) + `\b`),
	}
}

// Refresh 刷新敏感词列表
func (f *Filter) Refresh(ctx context.Context) error {
	f.refreshMu.Lock()
	defer f.refreshMu.Unlock()

	words, err := f.repo.ListActiveSensitiveWords(ctx)
	if err != nil {
		return err
	}

	f.applyWords(words)
	return nil
}

// Invalidate 将当前进程的敏感词快照标记为过期。
//
// 与 Refresh 共用 refreshMu，避免管理端 mutation 与正在进行的刷新交错时，
// 旧查询结果重新把快照标记为 5 分钟有效。已经开始的内容检查允许完成；
// mutation 返回后发起的下一次检查会重新加载数据库词表。
func (f *Filter) Invalidate() {
	f.refreshMu.Lock()
	defer f.refreshMu.Unlock()

	f.mu.Lock()
	f.lastRefresh = time.Time{}
	f.mu.Unlock()
}

func (f *Filter) applyWords(words []SensitiveWord) {
	blockMatchers := make([]wordMatcher, 0, len(words))
	warnMatchers := make([]wordMatcher, 0, len(words))
	reviewMatchers := make([]wordMatcher, 0, len(words))
	for _, w := range words {
		m := buildMatcher(w.Word)
		switch w.Level {
		case "block":
			blockMatchers = append(blockMatchers, m)
		case "review":
			reviewMatchers = append(reviewMatchers, m)
		default:
			warnMatchers = append(warnMatchers, m)
		}
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.words = words
	f.blockMatchers = blockMatchers
	f.warnMatchers = warnMatchers
	f.reviewMatchers = reviewMatchers
	f.lastRefresh = time.Now()
}

// ensureFresh 确保敏感词列表是最新的
// 使用 singleflight 去重并发刷新，避免多个 goroutine 同时查询 DB
func (f *Filter) ensureFresh(ctx context.Context) error {
	f.mu.RLock()
	needRefresh := time.Since(f.lastRefresh) > f.refreshTTL
	f.mu.RUnlock()

	if !needRefresh {
		return nil
	}

	// singleflight 确保同一时刻只有一个 goroutine 执行 DB 查询
	// 使用独立 context 避免单个请求取消导致所有并发等待者的刷新失败
	err := singleflightx.Do(&f.sf, "refresh", func() error {
		// 二次检查：进入 singleflight 后再次确认是否仍需刷新
		f.mu.RLock()
		if time.Since(f.lastRefresh) <= f.refreshTTL {
			f.mu.RUnlock()
			return nil
		}
		f.mu.RUnlock()

		// 独立 context：刷新操作不应因某个请求取消而中断
		refreshCtx, cancel := detachedRefreshContext(ctx, 10*time.Second)
		defer cancel()

		// Refresh 仅与管理端 invalidation 串行；数据库查询期间不会持有
		// matcher 读写锁，因此已加载快照的并发读不会被阻塞。
		if err := f.Refresh(refreshCtx); err != nil {
			logger.L().Warn("failed to refresh sensitive words", zap.Error(err))
			return err
		}

		return nil
	})
	if err != nil {
		return errors.Join(ErrModerationUnavailable, err)
	}
	return nil
}

// CheckContent 检查内容是否包含敏感词
func (f *Filter) CheckContent(ctx context.Context, content string) (*ContentCheckResult, error) {
	if err := f.ensureFresh(ctx); err != nil {
		return nil, err
	}

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
		return result, nil
	}

	// 检查警告级别的敏感词
	for _, m := range f.warnMatchers {
		if matchWord(m, lowerContent, content) {
			result.Level = "warn"
			result.MatchCount++
		}
	}

	if result.Level == "warn" {
		return result, nil
	}

	for _, m := range f.reviewMatchers {
		if matchWord(m, lowerContent, content) {
			result.Level = "review"
			result.MatchCount++
		}
	}

	return result, nil
}

// matchWord 根据匹配器类型执行匹配：正则（英文词边界）或子串（中文等）
func matchWord(m wordMatcher, lowerContent, originalContent string) bool {
	if m.regex != nil {
		return m.regex.MatchString(originalContent)
	}
	return strings.Contains(lowerContent, m.word)
}

// ContainsBlockedWord 检查是否包含阻止级别的敏感词
func (f *Filter) ContainsBlockedWord(ctx context.Context, content string) (bool, error) {
	result, err := f.CheckContent(ctx, content)
	if err != nil {
		return false, err
	}
	return !result.IsValid, nil
}
