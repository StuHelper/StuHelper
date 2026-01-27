package course

import (
	"encoding/json"
	"fmt"
	"time"
)

// RatingDimension 评分维度配置
type RatingDimension struct {
	ID          int64     `json:"id"`
	SchoolID    int64     `json:"school_id"`
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	SortOrder   int       `json:"sort_order"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Department 院系
type Department struct {
	ID        int64  `json:"id"`
	SchoolID  int64  `json:"school_id"`
	Name      string `json:"name"`
	ShortName string `json:"short_name,omitempty"`
	Category  string `json:"category"`
	SortOrder int    `json:"sort_order"`
}

// Course 课程
type Course struct {
	ID             int64   `json:"id"`
	SchoolID       int64   `json:"school_id"`
	DepartmentID   int64   `json:"department_id"`
	DepartmentName string  `json:"department_name,omitempty"`
	Code           string  `json:"code,omitempty"`
	Name           string  `json:"name"`
	Credits        float64 `json:"credits,omitempty"`
	ReviewCount    int     `json:"review_count"`
}

// Teacher 教师
type Teacher struct {
	ID           int64  `json:"id"`
	SchoolID     int64  `json:"school_id"`
	Name         string `json:"name"`
	DepartmentID int64  `json:"department_id,omitempty"`
}

// Term 学期
type Term struct {
	ID        string `json:"id"`
	SchoolID  int64  `json:"school_id"`
	Name      string `json:"name"`
	IsCurrent bool   `json:"is_current"`
}

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
	CourseID     int64         `json:"course_id"`
	CourseName   string        `json:"course_name,omitempty"`
	TeacherID    *int64        `json:"teacher_id,omitempty"`
	TeacherName  string        `json:"teacher_name,omitempty"`
	TermID       string        `json:"term_id,omitempty"`
	TermName     string        `json:"term_name,omitempty"`
	UserHash     string        `json:"-"`
	Title        string        `json:"title,omitempty"`
	Content      string        `json:"content"`
	Grade        string        `json:"grade,omitempty"`
	Ratings      ReviewRatings `json:"ratings"`
	LikeCount    int           `json:"like_count"`
	DislikeCount int           `json:"dislike_count"`
	Status       string        `json:"status"`
	CreatedAt    time.Time     `json:"created_at"`
}

// CourseRatingStats 课程评分统计
type CourseRatingStats struct {
	ID           int64           `json:"id"`
	CourseID     int64           `json:"course_id"`
	TermID       *string         `json:"term_id,omitempty"`
	DimensionKey string          `json:"dimension_key"`
	AvgRating    float64         `json:"avg_rating"`
	RatingCount  int             `json:"rating_count"`
	RatingDist   json.RawMessage `json:"rating_dist"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// DimensionStats 维度统计（用于API响应）
type DimensionStats struct {
	Key          string      `json:"key"`
	Name         string      `json:"name"`
	AvgRating    float64     `json:"avg_rating"`
	RatingCount  int         `json:"rating_count"`
	Distribution map[int]int `json:"distribution"`
}

// TermRatingStats 学期评分统计
type TermRatingStats struct {
	TermID     string           `json:"term_id"`
	TermName   string           `json:"term_name"`
	Dimensions []DimensionStats `json:"dimensions"`
}

// RadarChartDataset 雷达图数据集
type RadarChartDataset struct {
	Label           string    `json:"label"`
	Data            []float64 `json:"data"`
	BackgroundColor string    `json:"background_color"`
	BorderColor     string    `json:"border_color"`
}

// RadarChartData 雷达图数据
type RadarChartData struct {
	Labels   []string            `json:"labels"`
	Datasets []RadarChartDataset `json:"datasets"`
}

// CourseRatingStatsResponse 课程评分统计响应
type CourseRatingStatsResponse struct {
	CourseID         int64             `json:"course_id"`
	Overall          TermRatingStats   `json:"overall"`
	ByTerm           []TermRatingStats `json:"by_term"`
	AllDimensionKeys []string          `json:"all_dimension_keys"`
	RadarChart       RadarChartData    `json:"radar_chart"`
}
