package app

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	apigen "github.com/StuHelper/StuHelper/server/internal/api/gen"
)

func TestOpenAPIRateLimitedOperationsDeclareLimiterResponses(t *testing.T) {
	spec, err := apigen.GetSpec()
	require.NoError(t, err)

	for _, route := range []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/auth/exchange-native"},
		{http.MethodPost, "/api/v1/auth/refresh"},
		{http.MethodPost, "/api/v1/user/identity/uploads"},
		{http.MethodPost, "/api/v1/user/profile/verify"},
		{http.MethodPost, "/api/v1/user/profile/school-email/academic-match"},
		{http.MethodPost, "/api/v1/user/profile/school-email/request-otp"},
		{http.MethodPost, "/api/v1/user/profile/school-email/verify-otp"},
		{http.MethodPost, "/api/v1/user/profile/bind-phone/otp"},
		{http.MethodGet, "/api/v1/course/review/reviews/search"},
		{http.MethodGet, "/api/v1/course/review/reviews/batch"},
		{http.MethodPost, "/api/v1/course/review/reviews"},
		{http.MethodPut, "/api/v1/course/review/reviews/{reviewID}"},
		{http.MethodDelete, "/api/v1/course/review/reviews/{reviewID}"},
		{http.MethodPost, "/api/v1/course/review/reviews/{reviewID}/votes"},
		{http.MethodPost, "/api/v1/course/review/reviews/{reviewID}/reports"},
		{http.MethodPost, "/api/v1/course/review/reviews/{reviewID}/replies"},
		{http.MethodDelete, "/api/v1/course/review/replies/{replyID}"},
		{http.MethodPost, "/api/v1/admission/school-email/academic-match"},
		{http.MethodPost, "/api/v1/admission/school-email/request-otp"},
	} {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			path := spec.Paths.Value(route.path)
			require.NotNil(t, path, "rate-limited runtime route is missing from OpenAPI")
			operation := path.Operations()[route.method]
			require.NotNil(t, operation, "rate-limited runtime operation is missing from OpenAPI")
			require.NotNil(t, operation.Responses)
			for _, status := range []string{"429", "503"} {
				_, declared := operation.Responses.Map()[status]
				require.Truef(t, declared, "%s %s must declare limiter response %s", route.method, route.path, status)
			}
		})
	}
}
