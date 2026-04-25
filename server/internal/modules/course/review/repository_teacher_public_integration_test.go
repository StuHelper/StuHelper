package review

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestTeacherPublicStats_RefreshAndList(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	ctx := context.Background()

	departmentID := seedDepartment(t, fixture, 10006, "计算机学院")
	teacherA := seedTeacher(t, fixture, 10006, "王老师", departmentID)
	teacherB := seedTeacher(t, fixture, 10006, "李老师", departmentID)
	courseA := seedCourse(t, fixture, 10006, departmentID, "数据库系统")
	courseB := seedCourse(t, fixture, 10006, departmentID, "分布式系统")

	seedReview(t, fixture, "review-a1", courseA, teacherA, "u-1", 4.5, "published")
	seedReview(t, fixture, "review-a2", courseB, teacherA, "u-2", 3.5, "published")
	seedReview(t, fixture, "review-b1", courseA, teacherB, "u-3", 5.0, "hidden")

	require.NoError(t, repo.RefreshTeacherPublicStats(ctx))

	list, total, err := repo.ListPublicTeachers(ctx, "", &departmentID, "reviews", 10, 0)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, 2, total)

	assert.Equal(t, teacherA, list[0].TeacherID)
	assert.Equal(t, "王老师", list[0].TeacherName)
	assert.Equal(t, 2, list[0].ReviewCount)
	assert.Equal(t, 2, list[0].CourseCount)
	require.NotNil(t, list[0].AvgRating)
	assert.InDelta(t, 4.0, *list[0].AvgRating, 0.001)

	assert.Equal(t, teacherB, list[1].TeacherID)
	assert.Equal(t, 0, list[1].ReviewCount)
	assert.Equal(t, 0, list[1].CourseCount)
	assert.Nil(t, list[1].AvgRating)

	hot, err := repo.ListHotTeachers(ctx, 10)
	require.NoError(t, err)
	require.Len(t, hot, 1)
	assert.Equal(t, teacherA, hot[0].TeacherID)
}

func TestTeacherPublicStats_SearchAndDepartmentFilter(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	ctx := context.Background()

	deptCS := seedDepartment(t, fixture, 10006, "计算机学院")
	deptMath := seedDepartment(t, fixture, 10006, "数学学院")
	teacherCS := seedTeacher(t, fixture, 10006, "张三", deptCS)
	teacherMath := seedTeacher(t, fixture, 10006, "李四", deptMath)
	courseCS := seedCourse(t, fixture, 10006, deptCS, "算法设计")
	courseMath := seedCourse(t, fixture, 10006, deptMath, "高等数学")

	seedReview(t, fixture, "review-cs", courseCS, teacherCS, "u-1", 4.2, "published")
	seedReview(t, fixture, "review-math", courseMath, teacherMath, "u-2", 4.9, "published")

	require.NoError(t, repo.RefreshTeacherPublicStats(ctx))

	list, total, err := repo.ListPublicTeachers(ctx, "张", &deptCS, "name", 10, 0)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, 1, total)
	assert.Equal(t, teacherCS, list[0].TeacherID)
	assert.Equal(t, "张三", list[0].TeacherName)

	otherDept, total, err := repo.ListPublicTeachers(ctx, "", &deptMath, "rating", 10, 0)
	require.NoError(t, err)
	require.Len(t, otherDept, 1)
	assert.Equal(t, 1, total)
	assert.Equal(t, teacherMath, otherDept[0].TeacherID)
	assert.Equal(t, "李四", otherDept[0].TeacherName)
}

func seedDepartment(t *testing.T, fixture *postgresfixture.Fixture, schoolID int64, name string) int64 {
	t.Helper()
	var id int64
	err := fixture.Pool.QueryRow(context.Background(), `
		INSERT INTO departments (school_id, name, category, sort_order)
		VALUES ($1, $2, '', 0)
		RETURNING id
	`, schoolID, name).Scan(&id)
	require.NoError(t, err)
	return id
}

func seedTeacher(t *testing.T, fixture *postgresfixture.Fixture, schoolID int64, name string, departmentID int64) int64 {
	t.Helper()
	var id int64
	err := fixture.Pool.QueryRow(context.Background(), `
		INSERT INTO teachers (school_id, name, department_id)
		VALUES ($1, $2, $3)
		RETURNING id
	`, schoolID, name, departmentID).Scan(&id)
	require.NoError(t, err)
	return id
}

func seedCourse(t *testing.T, fixture *postgresfixture.Fixture, schoolID, departmentID int64, name string) int64 {
	t.Helper()
	var id int64
	err := fixture.Pool.QueryRow(context.Background(), `
		INSERT INTO courses (school_id, name, department_id, category, credits)
		VALUES ($1, $2, $3, '', 3)
		RETURNING id
	`, schoolID, name, departmentID).Scan(&id)
	require.NoError(t, err)
	return id
}

func seedReview(t *testing.T, fixture *postgresfixture.Fixture, reviewID string, courseID, teacherID int64, userHash string, avgRating float64, status string) {
	t.Helper()
	_, err := fixture.Pool.Exec(context.Background(), `
		INSERT INTO reviews (
			id, course_id, teacher_id, term_id, user_hash, title, content, ratings, avg_rating, status
		) VALUES (
			$1, $2, $3, '2025-2', $4, 'title', 'content', '{}'::jsonb, $5, $6
		)
	`, reviewID, courseID, teacherID, userHash, avgRating, status)
	require.NoError(t, err)
}
