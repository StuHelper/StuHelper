package course

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestCourseService_IntegrationReadPaths(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(repo, zap.NewNop())
	ctx := context.Background()

	deptCS := seedCourseDepartment(t, fixture, 10006, "计算机学院", "science", 1)
	deptMath := seedCourseDepartment(t, fixture, 10006, "数学学院", "science", 2)
	courseDB := seedCourseRecord(t, fixture, 10006, deptCS, "CS101", "数据库系统", 3.0, "通识", 2)
	courseAlgo := seedCourseRecord(t, fixture, 10006, deptCS, "CS102", "算法设计", 4.0, "通识", 1)
	courseMath := seedCourseRecord(t, fixture, 10006, deptMath, "MA101", "高等数学", 5.0, "数学", 0)
	seedCourseFavorite(t, fixture, "11111111-1111-1111-1111-111111111111", "user-hash-1", courseDB)

	departments, err := svc.GetDepartments(ctx, "science")
	require.NoError(t, err)
	require.Len(t, departments, 2)
	assert.Equal(t, "计算机学院", departments[0].Name)

	terms, err := svc.GetTerms(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, terms)
	assert.True(t, terms[0].IsCurrent)

	courses, err := svc.GetCourses(ctx, ListCoursesParams{Query: "系统", DepartmentID: deptCS, Category: "通识", Sort: CourseSortCredits, Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Len(t, courses.List, 1)
	assert.Equal(t, courseDB, courses.List[0].ID)
	assert.Equal(t, 1, courses.Total)

	searched, err := svc.SearchCourses(ctx, SearchCoursesParams{Query: "CS1", Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Len(t, searched.List, 2)
	assert.Equal(t, 2, searched.Total)

	course, err := svc.GetCourse(ctx, courseAlgo)
	require.NoError(t, err)
	assert.Equal(t, "算法设计", course.Name)
	assert.Equal(t, "计算机学院", course.DepartmentName)

	_, err = svc.GetCourse(ctx, 999999)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCourseNotFound)

	categories, err := svc.GetCourseCategories(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, categories)

	groups, err := svc.GetCoursesGrouped(ctx)
	require.NoError(t, err)
	require.Len(t, groups, 2)
	groupSizes := map[int64]int{}
	for _, g := range groups {
		groupSizes[g.DepartmentID] = len(g.Courses)
	}
	assert.Equal(t, 2, groupSizes[deptCS])
	assert.Equal(t, 1, groupSizes[deptMath])

	stats, err := svc.GetStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, stats.CourseCount)
	assert.Equal(t, 2, stats.DepartmentCount)

	exists, err := svc.FavoriteExists(ctx, "user-hash-1", courseDB)
	require.NoError(t, err)
	assert.True(t, exists)

	exists, err = svc.FavoriteExists(ctx, "user-hash-1", courseMath)
	require.NoError(t, err)
	assert.False(t, exists)

	favorited, err := svc.BatchFavoritedCourseIDs(ctx, "user-hash-1", []int64{courseDB, courseAlgo, courseMath})
	require.NoError(t, err)
	assert.True(t, favorited[courseDB])
	assert.False(t, favorited[courseAlgo])
	assert.False(t, favorited[courseMath])
}

func seedCourseDepartment(t *testing.T, fixture *postgresfixture.Fixture, schoolID int64, name, category string, sortOrder int) int64 {
	t.Helper()
	var id int64
	err := fixture.Pool.QueryRow(context.Background(), `
		INSERT INTO departments (school_id, name, short_name, category, sort_order)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, schoolID, name, name, category, sortOrder).Scan(&id)
	require.NoError(t, err)
	return id
}

func seedCourseRecord(t *testing.T, fixture *postgresfixture.Fixture, schoolID, departmentID int64, code, name string, credits float64, category string, reviewCount int) int64 {
	t.Helper()
	var id int64
	err := fixture.Pool.QueryRow(context.Background(), `
		INSERT INTO courses (school_id, department_id, code, name, credits, category, review_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, schoolID, departmentID, code, name, credits, category, reviewCount).Scan(&id)
	require.NoError(t, err)
	return id
}

func seedCourseFavorite(t *testing.T, fixture *postgresfixture.Fixture, id, userHash string, courseID int64) {
	t.Helper()
	_, err := fixture.Pool.Exec(context.Background(), `
		INSERT INTO course_favorites (id, user_hash, course_id)
		VALUES ($1, $2, $3)
	`, id, userHash, courseID)
	require.NoError(t, err)
}

func TestCourseRepositoryAndHandler_FavoritesAndCounts(t *testing.T) {
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(repo, zap.NewNop())
	h := &Handler{service: svc}
	ctx := context.Background()

	deptCS := seedCourseDepartment(t, fixture, 10006, "物理学院", "science", 1)
	deptMath := seedCourseDepartment(t, fixture, 10006, "化学学院", "science", 2)
	courseA := seedCourseRecord(t, fixture, 10006, deptCS, "PH101", "大学物理", 4.0, "通识", 0)
	courseB := seedCourseRecord(t, fixture, 10006, deptMath, "CH101", "大学化学", 3.0, "通识", 0)
	seedCourseFavorite(t, fixture, "22222222-2222-2222-2222-222222222222", "user-hash-batch", courseA)

	courseCount, err := repo.CountCourses(ctx, 0)
	require.NoError(t, err)
	assert.Equal(t, 2, courseCount)

	filteredCount, err := repo.CountCourses(ctx, deptCS)
	require.NoError(t, err)
	assert.Equal(t, 1, filteredCount)

	departmentCount, err := repo.CountDepartments(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, departmentCount)

	courses := []Course{{ID: courseA, Name: "大学物理"}, {ID: courseB, Name: "大学化学"}}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/courses", nil)
	h.annotateFavorites(c, "user-hash-batch", courses)
	require.NotNil(t, courses[0].IsFavorited)
	require.NotNil(t, courses[1].IsFavorited)
	assert.True(t, *courses[0].IsFavorited)
	assert.False(t, *courses[1].IsFavorited)
}
