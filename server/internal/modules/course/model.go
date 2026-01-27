package course

import "time"

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

// Stats 学习中心统计
type Stats struct {
	CourseCount     int       `json:"course_count"`
	DepartmentCount int       `json:"department_count"`
	TeacherCount    int       `json:"teacher_count"`
	UpdatedAt       time.Time `json:"updated_at,omitempty"`
}
