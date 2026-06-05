package review

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/capability"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/crypto"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/middleware"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/reviewaccess"
	"git.stuhelper.com/StuHelper/StuHelper/internal/pkg/systemconfig"
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

func TestMaskHash(t *testing.T) {
	assert.Equal(t, "short", maskHash("short"))
	assert.Equal(t, "123456789012...", maskHash("12345678901234567890"))
}

func TestBuildReviewAccessPolicy(t *testing.T) {
	policy, err := buildReviewAccessPolicy(
		[]reviewaccess.SchoolConfig{{SchoolID: 4111010001}, {SchoolID: 4111010002}},
		[]reviewaccess.SystemConfig{
			{Key: systemconfig.ReviewAccessSchoolIDsKey, Value: `4111010002,4111010003`},
			{Key: systemconfig.ReviewPreviewTitleCharsKey, Value: `12`},
			{Key: systemconfig.ReviewPreviewContentCharsKey, Value: `80`},
			{Key: systemconfig.ReviewPreviewContentPercentKey, Value: `60`},
		},
	)
	require.NoError(t, err)
	assert.Equal(t, []string{"4111010002", "4111010003"}, policy.AllowedSchoolIDs)
	assert.Equal(t, 12, policy.PreviewTitleRunes)
	assert.Equal(t, 80, policy.PreviewContentRunes)
	assert.Equal(t, 60, policy.PreviewContentPct)

	_, err = buildReviewAccessPolicy(nil, []reviewaccess.SystemConfig{{Key: systemconfig.ReviewPreviewTitleCharsKey, Value: `999`}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), systemconfig.ReviewPreviewTitleCharsKey)
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
		configs: []reviewaccess.SystemConfig{{Key: systemconfig.ReviewPreviewTitleCharsKey, Value: `16`}},
		subject: &reviewaccess.Subject{InternalUserID: 42, SchoolID: &schoolID, StudentVerified: true, IdentityVerified: true},
	}}

	facts, err := svc.ResolveAccessFacts(context.Background(), "external-1", []string{
		capability.ReviewListFull,
		capability.ReviewCreate,
		capability.ReviewEditOwn,
		capability.ReviewDeleteOwn,
	})
	require.NoError(t, err)
	assert.True(t, facts.Authenticated)
	assert.True(t, facts.StudentVerified)
	assert.True(t, facts.IdentityVerified)
	assert.True(t, facts.CanViewFull)
	assert.True(t, facts.CanPostReview)
	assert.True(t, facts.CanEditOwn)
	assert.True(t, facts.CanDeleteOwn)
	assert.Equal(t, int64(42), facts.InternalUserID)
	assert.Equal(t, 16, facts.PreviewTitleRunes)
	require.NotNil(t, facts.SchoolID)
	assert.Equal(t, "4111010001", *facts.SchoolID)
}

func TestResolveAccessFacts_AnonymousAndCacheFresh(t *testing.T) {
	systemconfig.SetReviewAccessPolicySnapshot(systemconfig.ReviewAccessPolicySnapshot{
		AllowedSchoolIDs:    []string{"4111010001"},
		PreviewTitleRunes:   18,
		PreviewContentRunes: 90,
		PreviewContentPct:   75,
		LoadedAt:            time.Now(),
	})
	t.Cleanup(systemconfig.InvalidateReviewAccessPolicySnapshot)

	svc := &Service{accessReader: fakeAccessReader{err: assert.AnError}}
	facts, err := svc.ResolveAccessFacts(context.Background(), "", nil)
	require.NoError(t, err)
	assert.False(t, facts.Authenticated)
	assert.Equal(t, 18, facts.PreviewTitleRunes)
	assert.Equal(t, 90, facts.PreviewContentRunes)
	assert.Equal(t, 75, facts.PreviewContentPct)
}

func TestResolveAccessFacts_UsesServiceLifecycleForPolicyRefresh(t *testing.T) {
	systemconfig.InvalidateReviewAccessPolicySnapshot()
	t.Cleanup(systemconfig.InvalidateReviewAccessPolicySnapshot)

	lifecycleCtx, cancelLifecycle := context.WithCancel(context.Background())
	cancelLifecycle()

	svc := &Service{
		accessReader: contextAwareAccessReader{},
		asyncCtx:     lifecycleCtx,
	}

	_, err := svc.ResolveAccessFacts(context.Background(), "", nil)
	require.ErrorIs(t, err, context.Canceled)
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
