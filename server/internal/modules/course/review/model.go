package review

import (
	"encoding/json"
	"fmt"
	"time"
)

// ReviewRatings 评分JSON类型
type ReviewRatings map[string]int

// Scan 实现 sql.Scanner 接口
func (r *ReviewRatings) Scan(value interface{}) error {
	if value == nil {
		*r = make(ReviewRatings)
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("ReviewRatings.Scan: expected []byte, got %T", value)
	}
	return json.Unmarshal(bytes, r)
}

// Review 测评
type Review struct {
	ID           string        `json:"id"`
	CourseID     int64         `json:"courseID"`
	CourseName   string        `json:"courseName,omitempty"`
	TeacherID    *int64        `json:"teacherID,omitempty"`
	TeacherName  string        `json:"teacherName,omitempty"`
	TermID       string        `json:"termID,omitempty"`
	TermName     string        `json:"termName,omitempty"`
	UserHash     string        `json:"-"`
	Title        string        `json:"title,omitempty"`
	Content      string        `json:"content"`
	Grade        string        `json:"grade,omitempty"`
	Ratings      ReviewRatings `json:"ratings"`
	LikeCount    int           `json:"likeCount"`
	DislikeCount int           `json:"dislikeCount"`
	ReplyCount   int           `json:"replyCount"`
	Status       string        `json:"status"`
	CreatedAt    time.Time     `json:"createdAt"`
}

// RatingDimension 评分维度配置
type RatingDimension struct {
	ID          string    `json:"id"`
	SchoolID    int64     `json:"schoolID"`
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	SortOrder   int       `json:"sortOrder"`
	IsActive    bool      `json:"isActive"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// CourseRatingStats 课程评分统计
type CourseRatingStats struct {
	ID           string          `json:"id"`
	CourseID     int64           `json:"courseID"`
	TermID       *string         `json:"termID,omitempty"`
	DimensionKey string          `json:"dimensionKey"`
	AvgRating    float64         `json:"avgRating"`
	RatingCount  int             `json:"ratingCount"`
	RatingDist   json.RawMessage `json:"ratingDist"`
	UpdatedAt    time.Time       `json:"updatedAt"`
}

// DimensionStats 维度统计（用于API响应）
type DimensionStats struct {
	Key          string      `json:"key"`
	Name         string      `json:"name"`
	AvgRating    float64     `json:"avgRating"`
	RatingCount  int         `json:"ratingCount"`
	Distribution map[int]int `json:"distribution"`
}

// TermRatingStats 学期评分统计
type TermRatingStats struct {
	TermID     string           `json:"termID"`
	TermName   string           `json:"termName"`
	Dimensions []DimensionStats `json:"dimensions"`
}

// RadarChartDataset 雷达图数据集
type RadarChartDataset struct {
	Label           string    `json:"label"`
	Data            []float64 `json:"data"`
	BackgroundColor string    `json:"backgroundColor"`
	BorderColor     string    `json:"borderColor"`
}

// RadarChartData 雷达图数据
type RadarChartData struct {
	Labels   []string            `json:"labels"`
	Datasets []RadarChartDataset `json:"datasets"`
}

// CourseRatingStatsResponse 课程评分统计响应
type CourseRatingStatsResponse struct {
	CourseID         int64             `json:"courseID"`
	Overall          TermRatingStats   `json:"overall"`
	ByTerm           []TermRatingStats `json:"byTerm"`
	AllDimensionKeys []string          `json:"allDimensionKeys"`
	RadarChart       RadarChartData    `json:"radarChart"`
}

// ReviewReport 举报记录
type ReviewReport struct {
	ID             string     `json:"id"`
	ReviewID       string     `json:"reviewID"`
	Review         *Review    `json:"review,omitempty"`
	ReporterHash   string     `json:"-"`
	Reason         string     `json:"reason"`
	Description    string     `json:"description,omitempty"`
	Status         string     `json:"status"`
	ResolvedBy     *string    `json:"resolvedBy,omitempty"`
	ResolvedAt     *time.Time `json:"resolvedAt,omitempty"`
	ResolutionNote *string    `json:"resolutionNote,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
}

// AdminStats 管理统计
type AdminStats struct {
	TotalReviews     int `json:"totalReviews"`
	PublishedReviews int `json:"publishedReviews"`
	HiddenReviews    int `json:"hiddenReviews"`
	DeletedReviews   int `json:"deletedReviews"`
	PendingReports   int `json:"pendingReports"`
	TotalReports     int `json:"totalReports"`
	TodayReviews     int `json:"todayReviews"`
	WeekReviews      int `json:"weekReviews"`
}

// CourseFavorite 课程收藏
type CourseFavorite struct {
	ID        string    `json:"id"`
	UserHash  string    `json:"-"`
	CourseID  int64     `json:"courseID"`
	CreatedAt time.Time `json:"createdAt"`
}

// FavoriteCourse 收藏的课程（包含课程信息，字段对齐前端 Course 类型）
type FavoriteCourse struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	Code           *string   `json:"code,omitempty"`
	Credits        float64   `json:"credits"`
	DepartmentID   int64     `json:"departmentID"`
	DepartmentName *string   `json:"departmentName,omitempty"`
	ReviewCount    int       `json:"reviewCount"`
	FavoritedAt    time.Time `json:"favoritedAt"`
}

// ReviewDraft 评论草稿
type ReviewDraft struct {
	ID        string        `json:"id"`
	UserHash  string        `json:"-"`
	CourseID  int64         `json:"courseID"`
	TeacherID *int64        `json:"teacherID,omitempty"`
	TermID    string        `json:"termID,omitempty"`
	Title     string        `json:"title,omitempty"`
	Content   string        `json:"content,omitempty"`
	Grade     string        `json:"grade,omitempty"`
	Ratings   ReviewRatings `json:"ratings"`
	UpdatedAt time.Time     `json:"updatedAt"`
}

// Reply 评论回复
type Reply struct {
	ID        string    `json:"id"`
	ReviewID  string    `json:"reviewID"`
	ParentID  *string   `json:"parentID,omitempty"`
	UserHash  string    `json:"-"`
	Content   string    `json:"content"`
	LikeCount int       `json:"likeCount"`
	Status    string    `json:"status"`
	IsOwner   bool      `json:"isOwner"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Notification 通知
type Notification struct {
	ID          string    `json:"id"`
	UserHash    string    `json:"-"`
	Type        string    `json:"type"`
	Title       string    `json:"title"`
	Content     string    `json:"content,omitempty"`
	RelatedType string    `json:"relatedType,omitempty"`
	RelatedID   string    `json:"relatedID,omitempty"`
	IsRead      bool      `json:"isRead"`
	CreatedAt   time.Time `json:"createdAt"`
}

// TeacherRatingStats 教师评分统计
type TeacherRatingStats struct {
	ID           string          `json:"id"`
	TeacherID    int64           `json:"teacherID"`
	TermID       *string         `json:"termID,omitempty"`
	DimensionKey string          `json:"dimensionKey"`
	AvgRating    float64         `json:"avgRating"`
	RatingCount  int             `json:"ratingCount"`
	RatingDist   json.RawMessage `json:"ratingDist"`
	UpdatedAt    time.Time       `json:"updatedAt"`
}

// TeacherRatingStatsResponse 教师评分统计响应
type TeacherRatingStatsResponse struct {
	TeacherID      int64              `json:"teacherID"`
	TeacherName    string             `json:"teacherName"`
	DepartmentName string             `json:"departmentName"`
	AvgRating      *float64           `json:"avgRating"`
	CourseCount    int                `json:"courseCount"`
	ReviewCount    int                `json:"reviewCount"`
	Courses        []TeacherCourse    `json:"courses"`
	RatingTrend    []RatingTrendItem `json:"ratingTrend"`
	Overall        TermRatingStats   `json:"overall"`
	ByTerm         []TermRatingStats  `json:"byTerm"`
	RadarChart     RadarChartData     `json:"radarChart"`
}

// CourseTeacherStats 课程详情页教师统计卡片数据
type CourseTeacherStats struct {
	TeacherID      int64    `json:"teacherID"`
	TeacherName    string   `json:"teacherName"`
	DepartmentName string   `json:"departmentName"`
	AvgRating      *float64 `json:"avgRating"`
	CourseCount    int      `json:"courseCount"`
	ReviewCount    int      `json:"reviewCount"`
}

// TeacherCourse 教师授课课程
type TeacherCourse struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	AvgRating   *float64 `json:"avgRating"`
	ReviewCount int      `json:"reviewCount"`
}

// SensitiveWord 敏感词
type SensitiveWord struct {
	ID        string    `json:"id"`
	Word      string    `json:"word"`
	Category  string    `json:"category"`
	Level     string    `json:"level"` // block, warn
	IsActive  bool      `json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
}

// ContentCheckResult 内容检查结果
type ContentCheckResult struct {
	IsValid    bool   `json:"isValid"`
	Level      string `json:"level,omitempty"`
	MatchCount int    `json:"matchCount,omitempty"`
}

// ContentCheckResponse 内容检查综合响应（敏感词 + 质量）
type ContentCheckResponse struct {
	Sensitive *ContentCheckResult `json:"sensitive"`
	Quality   *QualityCheckResult `json:"quality"`
}

// AdminOperationLog 管理操作日志
type AdminOperationLog struct {
	ID            string          `json:"id"`
	AdminUserID   string          `json:"adminUserID"`
	AdminUsername string          `json:"adminUsername"`
	Action        string          `json:"action"`
	ResourceType  string          `json:"resourceType"`
	ResourceID    string          `json:"resourceID"`
	OldValue      json.RawMessage `json:"oldValue,omitempty"`
	NewValue      json.RawMessage `json:"newValue,omitempty"`
	IPAddress     string          `json:"ipAddress,omitempty"`
	UserAgent     string          `json:"userAgent,omitempty"`
	CreatedAt     time.Time       `json:"createdAt"`
}

