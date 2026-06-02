package course

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/cache"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/httputil"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/redisfixture"
)

type courseHandlerEnvelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
}

func newCourseHandlerForTest(t *testing.T, svc *Service) *Handler {
	t.Helper()
	fixture := redisfixture.Start(t)

	return &Handler{
		cache:   cache.NewHelper(fixture.Client),
		service: svc,
	}
}

func decodeCourseEnvelope(t *testing.T, w *httptest.ResponseRecorder) courseHandlerEnvelope {
	t.Helper()
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var env courseHandlerEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &env))
	require.True(t, env.Success, w.Body.String())
	return env
}

func TestCourseHandler_IntegrationSuccessPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fixture := postgresfixture.Start(t)
	repo := NewRepository(fixture.DB)
	svc := NewService(repo, zap.NewNop())
	h := newCourseHandlerForTest(t, svc)
	ctx := context.Background()

	require.NoError(t, crypto.InitHMACKey("test-course-handler-secret-32-bytes!!", false))
	userID := "course-handler-user"
	userHash, err := httputil.HashUserID(userID)
	require.NoError(t, err)

	deptCS := seedCourseDepartment(t, fixture, 4111010006, "计算机学院", "science", 1)
	deptMath := seedCourseDepartment(t, fixture, 4111010006, "数学学院", "science", 2)
	courseDB := seedCourseRecord(t, fixture, 4111010006, deptCS, "CS101", "数据库系统", 3.0, "通识", 2)
	courseAlgo := seedCourseRecord(t, fixture, 4111010006, deptCS, "CS102", "算法设计", 4.0, "通识", 1)
	seedCourseRecord(t, fixture, 4111010006, deptMath, "MA101", "高等数学", 5.0, "数学", 0)
	seedCourseFavorite(t, fixture, "33333333-3333-3333-3333-333333333333", userHash, courseDB)

	// GetCourseCategories + cached path
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/course/categories", nil)
	h.GetCourseCategories(c)
	env := decodeCourseEnvelope(t, w)
	assert.Contains(t, string(env.Data), "通识")
	assert.Contains(t, string(env.Data), "体育")

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/course/categories", nil)
	h.GetCourseCategories(c)
	decodeCourseEnvelope(t, w)

	// GetDepartments
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/course/departments?category=science", nil)
	h.GetDepartments(c)
	env = decodeCourseEnvelope(t, w)
	assert.Contains(t, string(env.Data), "计算机学院")
	assert.Contains(t, string(env.Data), "数学学院")

	// GetTerms
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/course/terms", nil)
	h.GetTerms(c)
	env = decodeCourseEnvelope(t, w)
	assert.Contains(t, string(env.Data), "2025")

	// GetCourses logged-in branch with favorite annotation
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/course/courses?q=系统&departmentID="+json.Number("0").String()+"&category=通识&sort=credits&page=1&pageSize=10", nil)
	c.Request.URL.RawQuery = "q=系统&departmentID=" + strconv.FormatInt(deptCS, 10) + "&category=通识&sort=credits&page=1&pageSize=10"
	c.Set(middleware.CtxKeyUserID, userID)
	h.GetCourses(c)
	env = decodeCourseEnvelope(t, w)
	assert.Contains(t, string(env.Data), "数据库系统")
	assert.Contains(t, string(env.Data), "isFavorited")
	assert.Contains(t, string(env.Data), "true")

	// GetCoursesGrouped
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/course/courses/grouped", nil)
	h.GetCoursesGrouped(c)
	env = decodeCourseEnvelope(t, w)
	assert.Contains(t, string(env.Data), "groups")
	assert.Contains(t, string(env.Data), "计算机学院")

	// SearchCourses logged-in branch
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/course/courses/search?q=CS&page=1&pageSize=10", nil)
	c.Set(middleware.CtxKeyUserID, userID)
	h.SearchCourses(c)
	env = decodeCourseEnvelope(t, w)
	assert.Contains(t, string(env.Data), "CS101")
	assert.Contains(t, string(env.Data), "CS102")

	// GetCourse logged-in branch with favorite annotation
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "courseID", Value: strconv.FormatInt(courseDB, 10)}}
	c.Request = httptest.NewRequest(http.MethodGet, "/course/courses/"+strconv.FormatInt(courseDB, 10), nil)
	c.Set(middleware.CtxKeyUserID, userID)
	h.GetCourse(c)
	env = decodeCourseEnvelope(t, w)
	assert.Contains(t, string(env.Data), "数据库系统")
	assert.Contains(t, string(env.Data), "计算机学院")
	assert.Contains(t, string(env.Data), "isFavorited")

	// GetStats
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/course/stats", nil)
	h.GetStats(c)
	env = decodeCourseEnvelope(t, w)
	assert.Contains(t, string(env.Data), "courseCount")
	assert.Contains(t, string(env.Data), "departmentCount")

	// direct service invariants still sane
	total, err := repo.CountCourses(ctx, 0)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	_ = courseAlgo
}
