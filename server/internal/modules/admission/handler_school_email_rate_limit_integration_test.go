package admission

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
	"github.com/StuHelper/StuHelper/server/internal/testutil/redisfixture"
)

const schoolEmailRateLimitTestBody = `{
	"schoolCode":"4111010006",
	"studentID":"20250001",
	"studentName":"张三"
}`

type countingAcademicLookupGateway struct {
	calls atomic.Int64
}

func (g *countingAcademicLookupGateway) GetAcademicInfo(
	context.Context,
	int64,
	string,
) (*AcademicStudent, error) {
	g.calls.Add(1)
	return &AcademicStudent{StudentID: "20250001", Name: stringPtr("张三")}, nil
}

func TestAdmissionSchoolEmailLookupRateLimitIsSharedAndUserScoped(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pg := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	svc := newFreshmanTestService(t, pg)
	svc.redisClient = redis.Client
	svc.emailSender = &testSchoolEmailSender{}
	lookup := &countingAcademicLookupGateway{}
	svc.academicLookup = lookup
	enableAcademicStudentEmailPolicy(t, pg)

	firstUserID := seedAdmissionUser(t, pg, "school-email-limit-first")
	secondUserID := seedAdmissionUser(t, pg, "school-email-limit-second")
	linkAdmissionSessionForQQ(t, svc, firstUserID, "rate-limit-first-qq", "rate-limit-first-token")
	linkAdmissionSessionForQQ(t, svc, secondUserID, "rate-limit-second-qq", "rate-limit-second-token")
	router := newAdmissionSchoolEmailRateLimitTestRouter(t, svc, redis, map[string]int64{
		"rate-limit-user-first":  firstUserID,
		"rate-limit-user-second": secondUserID,
	})

	for range schoolEmailLookupRateLimitPerMinute - 1 {
		recorder := performAdmissionSchoolEmailRateLimitRequest(
			router,
			"rate-limit-user-first",
			"/api/v1/admission/school-email/academic-match",
		)
		require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	}

	otpRecorder := performAdmissionSchoolEmailRateLimitRequest(
		router,
		"rate-limit-user-first",
		"/api/v1/admission/school-email/request-otp",
	)
	require.Equal(t, http.StatusOK, otpRecorder.Code, otpRecorder.Body.String())
	require.EqualValues(t, schoolEmailLookupRateLimitPerMinute, lookup.calls.Load())

	limitedRecorder := performAdmissionSchoolEmailRateLimitRequest(
		router,
		"rate-limit-user-first",
		"/api/v1/admission/school-email/academic-match",
	)
	require.Equal(t, http.StatusTooManyRequests, limitedRecorder.Code, limitedRecorder.Body.String())
	assert.Equal(t, "60", limitedRecorder.Header().Get("Retry-After"))
	assert.EqualValues(t, schoolEmailLookupRateLimitPerMinute, lookup.calls.Load())

	isolatedRecorder := performAdmissionSchoolEmailRateLimitRequest(
		router,
		"rate-limit-user-second",
		"/api/v1/admission/school-email/academic-match",
	)
	require.Equal(t, http.StatusOK, isolatedRecorder.Code, isolatedRecorder.Body.String())
	assert.EqualValues(t, schoolEmailLookupRateLimitPerMinute+1, lookup.calls.Load())
}

func TestAdmissionSchoolEmailLookupRateLimitFailsClosedAfterAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pg := postgresfixture.Start(t)
	redis := redisfixture.Start(t)
	svc := newFreshmanTestService(t, pg)
	lookup := &countingAcademicLookupGateway{}
	svc.academicLookup = lookup
	userID := seedLinkedAdmissionUser(t, pg, svc, "school-email-limit-outage")
	router := newAdmissionSchoolEmailRateLimitTestRouter(t, svc, redis, map[string]int64{
		"rate-limit-outage-user": userID,
	})
	redis.Server.Close()

	unauthenticated := performAdmissionSchoolEmailRateLimitRequest(
		router,
		"",
		"/api/v1/admission/school-email/academic-match",
	)
	require.Equal(t, http.StatusUnauthorized, unauthenticated.Code, unauthenticated.Body.String())

	unavailable := performAdmissionSchoolEmailRateLimitRequest(
		router,
		"rate-limit-outage-user",
		"/api/v1/admission/school-email/academic-match",
	)
	require.Equal(t, http.StatusServiceUnavailable, unavailable.Code, unavailable.Body.String())
	assert.Zero(t, lookup.calls.Load())
}

func newAdmissionSchoolEmailRateLimitTestRouter(
	t *testing.T,
	svc *Service,
	redis *redisfixture.Fixture,
	userIDs map[string]int64,
) *gin.Engine {
	t.Helper()
	handler := NewHandler(
		svc,
		func(_ context.Context, subject string) (int64, error) {
			userID, exists := userIDs[subject]
			require.True(t, exists)
			return userID, nil
		},
		nil,
		WithSchoolEmailRateLimiter(redis.Client),
	)
	router := gin.New()
	handler.RegisterRoutes(router.Group("/api/v1"), func(c *gin.Context) {
		subject := c.GetHeader("X-Test-User")
		if subject == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Set(middleware.CtxKeyUserID, subject)
		c.Next()
	})
	return router
}

func performAdmissionSchoolEmailRateLimitRequest(
	router http.Handler,
	subject string,
	target string,
) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, target, strings.NewReader(schoolEmailRateLimitTestBody))
	request.Header.Set("Content-Type", "application/json")
	if subject != "" {
		request.Header.Set("X-Test-User", subject)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func enableAcademicStudentEmailPolicy(t *testing.T, fixture *postgresfixture.Fixture) {
	t.Helper()
	_, err := fixture.Pool.Exec(context.Background(), `
		UPDATE school_configs
		SET enabled = true,
		    manual_form_fields = '{
		      "admission": {
		        "emailDomains": ["buaa.edu.cn"],
		        "emailIdentityPolicy": {
		          "type": "academic_student_email",
		          "studentIDEmailDomain": "buaa.edu.cn",
		          "requireStudentName": true
		        }
		      }
		    }'::jsonb
		WHERE school_id = 4111010006
	`)
	require.NoError(t, err)
}
