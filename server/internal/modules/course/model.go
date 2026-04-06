package course

import "time"

// Department 院系
type Department struct {
	ID        int64  `json:"id"`
	SchoolID  int64  `json:"schoolID"`
	Name      string `json:"name"`
	ShortName string `json:"shortName,omitempty"`
	Category  string `json:"category"`
	SortOrder int    `json:"sortOrder"`
}

// Course 课程
type Course struct {
	ID             int64   `json:"id"`
	SchoolID       int64   `json:"schoolID"`
	DepartmentID   int64   `json:"departmentID"`
	DepartmentName string  `json:"departmentName,omitempty"`
	Code           string  `json:"code,omitempty"`
	Name           string  `json:"name"`
	Credits        float64 `json:"credits,omitempty"`
	Category       string  `json:"category"`
	ReviewCount    int     `json:"reviewCount"`
	IsFavorited    *bool   `json:"isFavorited,omitempty"`
}

// CourseCategory 课程分类（后台可配置）
type CourseCategory struct {
	ID        int64  `json:"id"`
	SchoolID  int64  `json:"schoolID"`
	Name      string `json:"name"`
	SortOrder int    `json:"sortOrder"`
}

// Teacher 教师
type Teacher struct {
	ID           int64  `json:"id"`
	SchoolID     int64  `json:"schoolID"`
	Name         string `json:"name"`
	DepartmentID int64  `json:"departmentID,omitempty"`
}

// Term 学期
type Term struct {
	ID        string `json:"id"`
	SchoolID  int64  `json:"schoolID"`
	Name      string `json:"name"`
	IsCurrent bool   `json:"isCurrent"`
}

// Stats 学习中心统计
type Stats struct {
	CourseCount     int       `json:"courseCount"`
	DepartmentCount int       `json:"departmentCount"`
	TeacherCount    int       `json:"teacherCount"`
	UpdatedAt       time.Time `json:"updatedAt,omitempty"`
}
