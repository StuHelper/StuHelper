package review

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/capability"
	"github.com/StuHelper/StuHelper/server/internal/pkg/crypto"
	"github.com/StuHelper/StuHelper/server/internal/pkg/middleware"
	"github.com/StuHelper/StuHelper/server/internal/pkg/reviewaccess"
	"github.com/StuHelper/StuHelper/server/internal/pkg/systemconfig"
)

type fakeAccessReader struct {
	schools []reviewaccess.SchoolConfig
	configs []reviewaccess.SystemConfig
	subject *reviewaccess.Subject
	err     error
}

func (f fakeAccessReader) ListReviewAccessSchoolConfigs(context.Context) ([]reviewaccess.SchoolConfig, error) {
	return f.schools, f.err
}
func (f fakeAccessReader) ListReviewAccessSystemConfigs(context.Context) ([]reviewaccess.SystemConfig, error) {
	return f.configs, f.err
}
func (f fakeAccessReader) GetReviewAccessSubject(context.Context, string) (*reviewaccess.Subject, error) {
	return f.subject, f.err
}

type contextAwareAccessReader struct{}

func (contextAwareAccessReader) ListReviewAccessSchoolConfigs(ctx context.Context) ([]reviewaccess.SchoolConfig, error) {
	return nil, ctx.Err()
}
func (contextAwareAccessReader) ListReviewAccessSystemConfigs(ctx context.Context) ([]reviewaccess.SystemConfig, error) {
	return nil, ctx.Err()
}
func (contextAwareAccessReader) GetReviewAccessSubject(ctx context.Context, _ string) (*reviewaccess.Subject, error) {
	return nil, ctx.Err()
}

func fullReviewWriteAccess(userID int64) ReviewWriteAccess {
	return ReviewWriteAccess{
		InternalUserID: userID,
		CanPostReview:  true,
		CanEditOwn:     true,
		CanDeleteOwn:   true,
	}
}

func TestMaskHash(t *testing.T) {
	assert.Equal(t, "short", maskHash("short"))
	assert.Equal(t, "123456789012...", maskHash("12345678901234567890"))
}

func TestBuildReviewAccessPolicy(t *testing.T) {
	policy, err := buildReviewAccessPolicy(
		[]reviewaccess.SchoolConfig{{SchoolID: 4111010001}, {SchoolID: 4111010002}},
		[]reviewaccess.SystemConfig{
			{Key: systemconfig.ReviewAccessSchoolIDsKey, Value: `4111010002,4111010003`},
			{Key: systemconfig.ReviewGuestPreviewContentCharsKey, Value: `12`},
			{Key: systemconfig.ReviewPreviewContentCharsKey, Value: `80`},
			{Key: systemconfig.ReviewPreviewContentPercentKey, Value: `60`},
		},
	)
	require.NoError(t, err)
	assert.Equal(t, []string{"4111010002", "4111010003"}, policy.AllowedSchoolIDs)
	assert.Equal(t, 12, policy.GuestPreviewContentRunes)
	assert.Equal(t, 80, policy.PreviewContentRunes)
	assert.Equal(t, 60, policy.PreviewContentPct)

	_, err = buildReviewAccessPolicy(nil, []reviewaccess.SystemConfig{{Key: systemconfig.ReviewGuestPreviewContentCharsKey, Value: `999`}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), systemconfig.ReviewGuestPreviewContentCharsKey)
}

func TestParseReviewAccessSchoolIDs(t *testing.T) {
	ids, err := parseReviewAccessSchoolIDs(map[string]string{systemconfig.ReviewAccessSchoolIDsKey: `["4111010001","4111010002"]`}, []string{"4111010004"})
	require.NoError(t, err)
	assert.Equal(t, []string{"4111010001", "4111010002"}, ids)

	ids, err = parseReviewAccessSchoolIDs(map[string]string{}, []string{"4111010004", "4111010004", " 4111010005 "})
	require.NoError(t, err)
	assert.Equal(t, []string{"4111010004", "4111010005"}, ids)
}

func TestNormalizeAuthorizationProvider(t *testing.T) {
	provider := normalizeAuthorizationProvider(nil)
	require.NotNil(t, provider)
	assert.ErrorIs(t,
		provider.WriteReviewRelations(context.Background(), "review-1", "user-1", "4111010006"),
		errAuthorizationProviderNotConfigured,
	)
	assert.ErrorIs(t,
		provider.WriteReportRelations(context.Background(), "report-1", "4111010006"),
		errAuthorizationProviderNotConfigured,
	)
}

func TestNormalizeReviewAccessReader(t *testing.T) {
	reader := normalizeReviewAccessReader(nil)
	_, err := reader.ListReviewAccessSchoolConfigs(context.Background())
	require.ErrorIs(t, err, errReviewAccessReaderNotConfigured)
}

func TestResolveAccessFacts(t *testing.T) {
	systemconfig.InvalidateReviewAccessPolicySnapshot()
	t.Cleanup(systemconfig.InvalidateReviewAccessPolicySnapshot)

	schoolID := int64(4111010001)
	svc := &Service{accessReader: fakeAccessReader{
		schools: []reviewaccess.SchoolConfig{{SchoolID: 4111010001}},
		configs: []reviewaccess.SystemConfig{{Key: systemconfig.ReviewGuestPreviewContentCharsKey, Value: `16`}},
		subject: &reviewaccess.Subject{InternalUserID: 42, SchoolID: &schoolID, StudentVerified: true, IdentityVerified: true},
	}}

	facts, err := svc.ResolveAccessFacts(context.Background(), "external-1", []string{
		capability.ReviewListFull,
		capability.ReviewCreate,
		capability.ReviewEditOwn,
		capability.ReviewDeleteOwn,
	}, nil)
	require.NoError(t, err)
	assert.True(t, facts.Authenticated)
	assert.True(t, facts.StudentVerified)
	assert.True(t, facts.IdentityVerified)
	assert.True(t, facts.CanViewFull)
	assert.True(t, facts.CanPostReview)
	assert.True(t, facts.CanEditOwn)
	assert.True(t, facts.CanDeleteOwn)
	assert.Equal(t, int64(42), facts.InternalUserID)
	assert.Equal(t, 16, facts.GuestPreviewContentRunes)
	require.NotNil(t, facts.SchoolID)
	assert.Equal(t, "4111010001", *facts.SchoolID)
}

func TestResolveAccessFacts_AnonymousAndCacheFresh(t *testing.T) {
	systemconfig.SetReviewAccessPolicySnapshot(systemconfig.ReviewAccessPolicySnapshot{
		AllowedSchoolIDs:         []string{"4111010001"},
		GuestPreviewContentRunes: 18,
		PreviewContentRunes:      90,
		PreviewContentPct:        75,
		LoadedAt:                 time.Now(),
	})
	t.Cleanup(systemconfig.InvalidateReviewAccessPolicySnapshot)

	svc := &Service{accessReader: fakeAccessReader{err: assert.AnError}}
	facts, err := svc.ResolveAccessFacts(context.Background(), "", nil, nil)
	require.NoError(t, err)
	assert.False(t, facts.Authenticated)
	assert.Equal(t, 18, facts.GuestPreviewContentRunes)
	assert.Equal(t, 90, facts.PreviewContentRunes)
	assert.Equal(t, 75, facts.PreviewContentPct)
}

func TestResolveAccessFacts_UsesServiceLifecycleForPolicyRefresh(t *testing.T) {
	systemconfig.InvalidateReviewAccessPolicySnapshot()
	t.Cleanup(systemconfig.InvalidateReviewAccessPolicySnapshot)

	lifecycleCtx, cancelLifecycle := context.WithCancel(context.Background())
	cancelLifecycle()

	svc := &Service{
		accessReader:  contextAwareAccessReader{},
		backgroundCtx: lifecycleCtx,
	}

	_, err := svc.ResolveAccessFacts(context.Background(), "", nil, nil)
	require.ErrorIs(t, err, context.Canceled)
}

func TestResolveReviewAccessFactsForRequestRequiresGlobalManageCapability(t *testing.T) {
	gin.SetMode(gin.TestMode)
	systemconfig.SetReviewAccessPolicySnapshot(systemconfig.ReviewAccessPolicySnapshot{
		GuestPreviewContentRunes: 8,
		PreviewContentRunes:      80,
		PreviewContentPct:        100,
		LoadedAt:                 time.Now(),
	})
	t.Cleanup(systemconfig.InvalidateReviewAccessPolicySnapshot)

	handler := &Handler{service: &Service{accessReader: fakeAccessReader{
		subject: &reviewaccess.Subject{InternalUserID: 42},
	}}}
	resolve := func(t *testing.T, snapshot capability.UserAccessSnapshot) ReviewAccessFacts {
		t.Helper()
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Set(middleware.CtxKeyUserID, "admin-subject")
		c.Set(middleware.CtxKeyCapabilities, snapshot.Capabilities)
		c.Set(middleware.CtxKeyGlobalCapabilities, snapshot.GlobalCapabilities)

		facts, ok := handler.resolveReviewAccessFactsForRequest(c)
		require.True(t, ok)
		require.Equal(t, http.StatusOK, w.Code)
		return facts
	}

	t.Run("scoped grant stays preview-only on public reads", func(t *testing.T) {
		testCases := []struct {
			name  string
			grant capability.Grant
		}{
			{
				name: "school scope",
				grant: capability.Grant{
					Name:           capability.AdminReviewsManage,
					ScopeSchoolIDs: []string{"4111010006"},
				},
			},
			{
				name: "section scope",
				grant: capability.Grant{
					Name:            capability.AdminReviewsManage,
					ScopeSectionIDs: []string{"school_4111010006_review_moderation"},
				},
			},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				snapshot := capability.BuildUserAccessSnapshot([]capability.Grant{tc.grant})
				require.Contains(t, snapshot.Capabilities, capability.AdminReviewsManage)
				require.Empty(t, snapshot.GlobalCapabilities)

				facts := resolve(t, snapshot)
				assert.False(t, facts.CanManageReviews)
				assert.False(t, facts.CanViewFull)

				result := stripReviewsForResponse([]Review{{
					Status:  StatusPublished,
					Content: "范围外正文第一行很长，不能因为 scoped grant 返回完整内容\n第二行敏感正文",
				}}, facts)
				require.Len(t, result, 1)
				assert.NotContains(t, result[0].Content, "第二行敏感正文")
			})
		}
	})

	t.Run("global grant keeps platform-wide management access", func(t *testing.T) {
		snapshot := capability.BuildUserAccessSnapshot([]capability.Grant{{
			Name: capability.AdminReviewsManage,
		}})
		require.Contains(t, snapshot.GlobalCapabilities, capability.AdminReviewsManage)

		facts := resolve(t, snapshot)
		assert.True(t, facts.CanManageReviews)
		assert.True(t, facts.CanViewFull)

		const content = "全局管理员可见完整第一行\n完整第二行"
		result := stripReviewsForResponse([]Review{{
			Status:  StatusPublished,
			Content: content,
		}}, facts)
		require.Len(t, result, 1)
		assert.Equal(t, content, result[0].Content)
	})
}

func TestResolveUserHashHelpers(t *testing.T) {
	require.NoError(t, crypto.InitHMACKey("test-review-hash-secret-32-bytes!!", false))
	gin.SetMode(gin.TestMode)

	h := &Handler{}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(middleware.CtxKeyUserID, "user-123")
	userID, userHash, ok := h.resolveRequiredUserHash(c)
	require.True(t, ok)
	assert.Equal(t, "user-123", userID)
	assert.NotEmpty(t, userHash)

	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	hash, ok := h.resolveOptionalUserHash(c2)
	require.True(t, ok)
	assert.Empty(t, hash)
}
