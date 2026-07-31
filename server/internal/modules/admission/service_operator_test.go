package admission

import (
	"context"
	"encoding/base64"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/StuHelper/StuHelper/server/internal/pkg/capability"
	"github.com/StuHelper/StuHelper/server/internal/testutil/postgresfixture"
)

func TestOperatorFreshmanReviewAuthorizesOperatorQQ(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newOperatorTestService(t, fixture)
	app := seedReviewableFreshmanApplication(t, fixture, svc)

	_, err := svc.ReviewFreshmanApplicationFromBot(context.Background(), botReviewInput(app.ID, "90001"))
	require.ErrorIs(t, err, ErrAdmissionOperatorUnbound)

	operatorID := seedAdmissionUser(t, fixture, "operator-no-cap")
	bindAdmissionOperatorQQ(t, fixture, operatorBindingSeed{UserID: operatorID, QQID: "90001"})
	svc.operatorAccess = &testOperatorAccessGateway{}
	_, err = svc.ReviewFreshmanApplicationFromBot(context.Background(), botReviewInput(app.ID, "90001"))
	require.ErrorIs(t, err, ErrAdmissionOperatorForbidden)

	svc.operatorAccess = &testOperatorAccessGateway{allowedUserID: operatorID}
	wrongGuild := botReviewInput(app.ID, "90001")
	wrongGuild.GuildID = "mgmt-2"
	_, err = svc.ReviewFreshmanApplicationFromBot(context.Background(), wrongGuild)
	require.ErrorIs(t, err, ErrAdmissionManagementGuildForbidden)
}

func TestFreshmanReviewApprovesAndRejects(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newOperatorTestService(t, fixture)
	operatorID := seedAdmissionUser(t, fixture, "operator-approve")
	bindAdmissionOperatorQQ(t, fixture, operatorBindingSeed{UserID: operatorID, QQID: "90002"})
	svc.operatorAccess = &testOperatorAccessGateway{allowedUserID: operatorID}

	approvedApp := seedReviewableFreshmanApplication(t, fixture, svc)
	expiresInDays := 7
	approved, err := svc.ReviewFreshmanApplicationFromBot(
		context.Background(),
		botReviewInputWithExpiry(approvedApp.ID, "90002", expiresInDays),
	)
	require.NoError(t, err)
	assert.Equal(t, FreshmanApplicationApproved, approved.Status)
	require.NotNil(t, approved.ProvisionalExpiresAt)
	assert.True(t, svc.now().Add(7*admissionDay).Equal(*approved.ProvisionalExpiresAt))
	assertCredentialStored(t, fixture, approved.UserID, CredentialFreshmanMaterialManual, "freshman material A***")
	assertUserSessionVerified(t, fixture, approved.UserID)
	assertUserProfileVerified(t, fixture, approved.UserID, "manual")
	assertApplicationReviewer(t, fixture, reviewerExpectation{
		ApplicationID: approved.ID, UserID: operatorID, QQID: "90002",
	})

	rejectedApp := seedReviewableFreshmanApplication(t, fixture, svc)
	reject := botReviewInput(rejectedApp.ID, "90002")
	reject.Action = FreshmanReviewReject
	rejected, err := svc.ReviewFreshmanApplicationFromBot(context.Background(), reject)
	require.NoError(t, err)
	assert.Equal(t, FreshmanApplicationRejected, rejected.Status)
	assert.Nil(t, rejected.ProvisionalExpiresAt)
}

func TestFreshmanReviewRequiresSubmittedMaterialBeforeDeadline(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newOperatorTestService(t, fixture)
	operatorID := seedAdmissionUser(t, fixture, "operator-material-required")
	bindAdmissionOperatorQQ(t, fixture, operatorBindingSeed{UserID: operatorID, QQID: "90005"})
	svc.operatorAccess = &testOperatorAccessGateway{allowedUserID: operatorID}

	unsubmitted := seedUnsubmittedFreshmanApplication(t, fixture, svc)
	_, err := svc.ReviewFreshmanApplicationFromBot(
		context.Background(),
		botReviewInput(unsubmitted.ID, "90005"),
	)
	require.ErrorIs(t, err, ErrAdmissionInvalidStatus)
	assertNoCredentialStored(t, fixture, unsubmitted.UserID, CredentialFreshmanMaterialManual)

	expired := seedReviewableFreshmanApplication(t, fixture, svc)
	svc.now = func() time.Time { return fixedAdmissionNow().Add(25 * time.Hour) }
	_, err = svc.ReviewFreshmanApplicationFromBot(
		context.Background(),
		botReviewInput(expired.ID, "90005"),
	)
	require.ErrorIs(t, err, ErrAdmissionInvalidStatus)
	assertNoCredentialStored(t, fixture, expired.UserID, CredentialFreshmanMaterialManual)
}

func TestFreshmanReviewEnforcesExtensionLimit(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newOperatorTestService(t, fixture)
	operatorID := seedAdmissionUser(t, fixture, "operator-extension")
	bindAdmissionOperatorQQ(t, fixture, operatorBindingSeed{UserID: operatorID, QQID: "90003"})
	svc.operatorAccess = &testOperatorAccessGateway{allowedUserID: operatorID}

	cases := []struct {
		name          string
		expiresInDays int
		wantErr       error
	}{
		{name: "negative", expiresInDays: -1, wantErr: ErrAdmissionInvalidInput},
		{name: "zero", expiresInDays: 0, wantErr: ErrAdmissionInvalidInput},
		{name: "too long", expiresInDays: DefaultMaxExtensionDays + 1, wantErr: ErrAdmissionReviewExtensionTooLong},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := seedReviewableFreshmanApplication(t, fixture, svc)

			_, err := svc.ReviewFreshmanApplicationFromBot(
				context.Background(),
				botReviewInputWithExpiry(app.ID, "90003", tc.expiresInDays),
			)

			require.ErrorIs(t, err, tc.wantErr)
			assertNoCredentialStored(t, fixture, app.UserID, CredentialFreshmanMaterialManual)
		})
	}
}

func newOperatorTestService(t *testing.T, fixture *postgresfixture.Fixture) *Service {
	t.Helper()
	svc := newFreshmanTestService(t, fixture)
	svc.materialStore = &testAdmissionMaterialStore{}
	return svc
}

func seedReviewableFreshmanApplication(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	svc *Service,
) *FreshmanApplication {
	t.Helper()
	app := seedUnsubmittedFreshmanApplication(t, fixture, svc)
	_, err := svc.SubmitCameraCapture(context.Background(), CameraCaptureInput{
		UserID:        app.UserID,
		ApplicationID: app.ID,
		ContentType:   "image/png",
		ImageBase64:   base64.StdEncoding.EncodeToString(validPNGBytes()),
	})
	require.NoError(t, err)
	return app
}

func seedUnsubmittedFreshmanApplication(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	svc *Service,
) *FreshmanApplication {
	t.Helper()
	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	userID := seedAdmissionUser(t, fixture, "freshman-review-"+suffix)
	linkFreshmanReviewSession(t, svc, freshmanReviewSessionSeed{
		UserID: userID, QQID: "10" + suffix, Token: "review-token-" + suffix,
	})
	return createFreshmanTestApplication(t, svc, userID)
}

func linkFreshmanReviewSession(t *testing.T, svc *Service, seed freshmanReviewSessionSeed) {
	t.Helper()
	svc.generateToken = func() (string, error) { return seed.Token, nil }
	created, err := svc.CreateBotSession(context.Background(), BotSessionCreateInput{
		Platform: "qq", GuildID: "guild-1", ChannelID: "channel-1", QQID: seed.QQID, BotSelfID: "514",
	})
	require.NoError(t, err)
	_, err = svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:  created.Token,
		UserID: seed.UserID,
	})
	require.NoError(t, err)
}

type freshmanReviewSessionSeed struct {
	UserID int64
	QQID   string
	Token  string
}

func botReviewInput(applicationID string, operatorQQID string) BotFreshmanReviewInput {
	return BotFreshmanReviewInput{
		ApplicationID: applicationID,
		Action:        FreshmanReviewApprove,
		OperatorQQID:  operatorQQID,
		GuildID:       "mgmt-1",
		RawCommand:    "新生审核通过 " + applicationID,
	}
}

func botReviewInputWithExpiry(
	applicationID string,
	operatorQQID string,
	expiresInDays int,
) BotFreshmanReviewInput {
	input := botReviewInput(applicationID, operatorQQID)
	input.ExpiresInDays = &expiresInDays
	return input
}

func bindAdmissionOperatorQQ(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	seed operatorBindingSeed,
) {
	t.Helper()
	_, err := fixture.Pool.Exec(context.Background(), `
		INSERT INTO user_qq_bindings (user_id, qq_id, bound_at)
		VALUES ($1, $2, NOW())
	`, seed.UserID, seed.QQID)
	require.NoError(t, err)
}

type operatorBindingSeed struct {
	UserID int64
	QQID   string
}

func assertApplicationReviewer(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	want reviewerExpectation,
) {
	t.Helper()
	var reviewedBy int64
	var reviewedQQ string
	err := fixture.Pool.QueryRow(context.Background(), `
		SELECT reviewed_by_user_id, reviewed_by_operator_qq_id
		FROM freshman_verification_applications
		WHERE id = $1
	`, want.ApplicationID).Scan(&reviewedBy, &reviewedQQ)
	require.NoError(t, err)
	assert.Equal(t, want.UserID, reviewedBy)
	assert.Equal(t, want.QQID, reviewedQQ)
}

type reviewerExpectation struct {
	ApplicationID string
	UserID        int64
	QQID          string
}

type testOperatorAccessGateway struct {
	allowedUserID   int64
	allowedSchoolID int64
}

func (g *testOperatorAccessGateway) UserHasCapabilityInSchool(
	_ context.Context,
	userID int64,
	capName string,
	schoolID int64,
) (bool, error) {
	schoolAllowed := g.allowedSchoolID == 0 || schoolID == g.allowedSchoolID
	return userID == g.allowedUserID &&
		capName == capability.AdmissionFreshmanReview &&
		schoolAllowed, nil
}
