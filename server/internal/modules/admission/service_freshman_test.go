package admission

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestFreshmanApplicationRejectsClosedChannelAndReusesPendingApplication(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newFreshmanTestService(t, fixture)
	userID := seedLinkedAdmissionUser(t, fixture, svc, "freshman-closed")

	closeFreshmanChannel(t, fixture)
	_, err := svc.CreateFreshmanApplication(context.Background(), FreshmanApplicationCreateInput{
		UserID:        userID,
		SchoolID:      4111010006,
		ApplicantName: "Alice Applicant",
		MaterialType:  MaterialAdmissionNotice,
	})
	require.ErrorIs(t, err, ErrAdmissionFreshmanChannelClosed)

	openFreshmanChannel(t, fixture)
	app, err := svc.CreateFreshmanApplication(context.Background(), FreshmanApplicationCreateInput{
		UserID:        userID,
		SchoolID:      4111010006,
		ApplicantName: "Alice Applicant",
		MaterialType:  MaterialAdmissionNotice,
	})
	require.NoError(t, err)
	require.NotEmpty(t, app.ID)

	reused, err := svc.CreateFreshmanApplication(context.Background(), FreshmanApplicationCreateInput{
		UserID:        userID,
		SchoolID:      4111010006,
		ApplicantName: "Alice Applicant",
		MaterialType:  MaterialAdmissionNotice,
	})
	require.NoError(t, err)
	assert.Equal(t, app.ID, reused.ID)
}

func TestFreshmanApplicationReassignsPendingApplicationToCurrentSession(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newFreshmanTestService(t, fixture)
	tokenIndex := 0
	svc.generateToken = func() (string, error) {
		tokenIndex++
		return fmt.Sprintf("freshman-reassign-token-%d", tokenIndex), nil
	}
	userID := seedAdmissionUser(t, fixture, "freshman-reassign-session")
	created := createLinkableSession(t, svc)
	linked, err := svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:  created.Token,
		UserID: userID,
	})
	require.NoError(t, err)

	app, err := svc.CreateFreshmanApplication(context.Background(), FreshmanApplicationCreateInput{
		UserID:        userID,
		SchoolID:      4111010006,
		ApplicantName: "Alice Applicant",
		MaterialType:  MaterialAdmissionNotice,
	})
	require.NoError(t, err)
	require.NotNil(t, app.AdmissionSessionID)
	assert.Equal(t, linked.ID, *app.AdmissionSessionID)

	regenerated, err := svc.RegenerateBotAdmissionSession(context.Background(), BotSessionCreateInput{
		Platform:  "qq",
		BotSelfID: "514",
		GuildID:   "guild-1",
		ChannelID: "channel-1",
		QQID:      "10001",
	})
	require.NoError(t, err)
	relinked, err := svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:  regenerated.Token,
		UserID: userID,
	})
	require.NoError(t, err)

	reused, err := svc.CreateFreshmanApplication(context.Background(), FreshmanApplicationCreateInput{
		UserID:        userID,
		SchoolID:      4111010006,
		ApplicantName: "Alice Applicant",
		MaterialType:  MaterialAdmissionNotice,
	})
	require.NoError(t, err)
	assert.Equal(t, app.ID, reused.ID)
	require.NotNil(t, reused.AdmissionSessionID)
	assert.Equal(t, relinked.ID, *reused.AdmissionSessionID)
}

func TestFreshmanApplicationReassignsSubmittedMaterialToCurrentSession(t *testing.T) {
	fixture := postgresfixture.Start(t)
	store := &testAdmissionMaterialStore{}
	svc := newFreshmanTestService(t, fixture)
	svc.materialStore = store
	tokenIndex := 0
	svc.generateToken = func() (string, error) {
		tokenIndex++
		return fmt.Sprintf("freshman-reassign-submitted-token-%d", tokenIndex), nil
	}
	userID := seedAdmissionUser(t, fixture, "freshman-reassign-submitted")
	created := createLinkableSession(t, svc)
	linked, err := svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:  created.Token,
		UserID: userID,
	})
	require.NoError(t, err)

	app, err := svc.CreateFreshmanApplication(context.Background(), FreshmanApplicationCreateInput{
		UserID:        userID,
		SchoolID:      4111010006,
		ApplicantName: "Alice Applicant",
		MaterialType:  MaterialAdmissionNotice,
	})
	require.NoError(t, err)
	_, err = svc.SubmitCameraCapture(context.Background(), CameraCaptureInput{
		UserID:        userID,
		ApplicationID: app.ID,
		ContentType:   "image/png",
		ImageBase64:   base64.StdEncoding.EncodeToString(validPNGBytes()),
	})
	require.NoError(t, err)
	assertAdmissionSessionStatus(t, fixture, linked.ID, StatusMaterialSubmitted)

	regenerated, err := svc.RegenerateBotAdmissionSession(context.Background(), BotSessionCreateInput{
		Platform:  "qq",
		BotSelfID: "514",
		GuildID:   "guild-1",
		ChannelID: "channel-1",
		QQID:      "10001",
	})
	require.NoError(t, err)
	relinked, err := svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:  regenerated.Token,
		UserID: userID,
	})
	require.NoError(t, err)
	assertAdmissionSessionStatus(t, fixture, relinked.ID, StatusLinked)

	reused, err := svc.CreateFreshmanApplication(context.Background(), FreshmanApplicationCreateInput{
		UserID:        userID,
		SchoolID:      4111010006,
		ApplicantName: "Alice Applicant",
		MaterialType:  MaterialAdmissionNotice,
	})
	require.NoError(t, err)
	assert.Equal(t, app.ID, reused.ID)
	require.NotNil(t, reused.AdmissionSessionID)
	assert.Equal(t, relinked.ID, *reused.AdmissionSessionID)
	assertAdmissionSessionStatus(t, fixture, relinked.ID, StatusMaterialSubmitted)
}

func TestFreshmanApplicationUniqueRaceReusesPendingApplication(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newFreshmanTestService(t, fixture)
	userID := seedLinkedAdmissionUser(t, fixture, svc, "freshman-race")
	session, err := svc.GetBotAdmissionSession(context.Background(), BotSessionSubjectInput{
		Platform: "qq",
		GuildID:  "guild-1",
		QQID:     "10001",
	})
	require.NoError(t, err)
	inserted := false
	svc.beforeFreshmanApplicationCreate = func() {
		if inserted {
			return
		}
		inserted = true
		insertFreshmanPendingApplication(t, fixture, freshmanPendingSeed{
			ID:            "freshman-race-existing",
			UserID:        userID,
			SchoolID:      4111010006,
			SessionID:     session.ID,
			ApplicantName: "Concurrent Applicant",
		})
	}

	app, err := svc.CreateFreshmanApplication(context.Background(), FreshmanApplicationCreateInput{
		UserID:        userID,
		SchoolID:      4111010006,
		ApplicantName: "Alice Applicant",
		MaterialType:  MaterialAdmissionNotice,
	})
	require.NoError(t, err)
	assert.Equal(t, "freshman-race-existing", app.ID)
	require.NotNil(t, app.AdmissionSessionID)
	assert.Equal(t, session.ID, *app.AdmissionSessionID)
}

func TestFreshmanApplicationRejectsExpiredLinkedSession(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newFreshmanTestService(t, fixture)
	userID := seedLinkedAdmissionUser(t, fixture, svc, "freshman-expired")

	svc.now = func() time.Time { return fixedAdmissionNow().Add(25 * time.Hour) }

	_, err := svc.CreateFreshmanApplication(context.Background(), FreshmanApplicationCreateInput{
		UserID:        userID,
		SchoolID:      4111010006,
		ApplicantName: "Alice Applicant",
		MaterialType:  MaterialAdmissionNotice,
	})
	require.ErrorIs(t, err, ErrAdmissionTokenExpired)
}

func TestFreshmanApplicationUsesValidLinkedSessionWhenExpiredLinkedSessionIsNewer(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newFreshmanTestService(t, fixture)
	userID := seedLinkedAdmissionUser(t, fixture, svc, "freshman-ignore-expired-linked")
	current, err := svc.GetBotAdmissionSession(context.Background(), BotSessionSubjectInput{
		Platform: "qq",
		GuildID:  "guild-1",
		QQID:     "10001",
	})
	require.NoError(t, err)
	expiredID := insertExpiredLinkedAdmissionSessionForUser(t, fixture, userID)

	app, err := svc.CreateFreshmanApplication(context.Background(), FreshmanApplicationCreateInput{
		UserID:        userID,
		SchoolID:      4111010006,
		ApplicantName: "Alice Applicant",
		MaterialType:  MaterialAdmissionNotice,
	})

	require.NoError(t, err)
	require.NotNil(t, app.AdmissionSessionID)
	assert.Equal(t, current.ID, *app.AdmissionSessionID)
	assert.NotEqual(t, expiredID, *app.AdmissionSessionID)
}

func TestFreshmanApplicationUsesRequestedAdmissionSession(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newFreshmanTestService(t, fixture)
	userID := seedAdmissionUser(t, fixture, "freshman-current-session")
	first := linkAdmissionSessionForQQ(t, svc, userID, "10011", "freshman-current-session-first")
	second := linkAdmissionSessionForQQ(t, svc, userID, "10012", "freshman-current-session-second")
	setAdmissionSessionUpdatedAt(t, fixture, first.ID, fixedAdmissionNow())
	setAdmissionSessionUpdatedAt(t, fixture, second.ID, fixedAdmissionNow().Add(time.Minute))

	app, err := svc.CreateFreshmanApplication(context.Background(), FreshmanApplicationCreateInput{
		UserID:             userID,
		SchoolID:           4111010006,
		AdmissionSessionID: first.ID,
		ApplicantName:      "Alice Applicant",
		MaterialType:       MaterialAdmissionNotice,
	})

	require.NoError(t, err)
	require.NotNil(t, app.AdmissionSessionID)
	assert.Equal(t, first.ID, *app.AdmissionSessionID)
	assert.NotEqual(t, second.ID, *app.AdmissionSessionID)
}

func TestFreshmanCameraCaptureValidatesAndStoresImage(t *testing.T) {
	fixture := postgresfixture.Start(t)
	store := &testAdmissionMaterialStore{}
	svc := newFreshmanTestService(t, fixture)
	svc.materialStore = store
	userID := seedLinkedAdmissionUser(t, fixture, svc, "freshman-camera")
	app := createFreshmanTestApplication(t, svc, userID)

	_, err := svc.SubmitCameraCapture(context.Background(), CameraCaptureInput{
		UserID:        userID,
		ApplicationID: app.ID,
		ContentType:   "application/pdf",
		ImageBase64:   base64.StdEncoding.EncodeToString([]byte("%PDF")),
	})
	require.ErrorIs(t, err, ErrAdmissionMaterialInvalidType)

	_, err = svc.SubmitCameraCapture(context.Background(), CameraCaptureInput{
		UserID:        userID,
		ApplicationID: app.ID,
		ContentType:   "image/png",
		ImageBase64:   base64.StdEncoding.EncodeToString([]byte("not an image")),
	})
	require.ErrorIs(t, err, ErrAdmissionMaterialInvalidData)

	updated, err := svc.SubmitCameraCapture(context.Background(), CameraCaptureInput{
		UserID:        userID,
		ApplicationID: app.ID,
		ContentType:   "image/png",
		ImageBase64:   base64.StdEncoding.EncodeToString(validPNGBytes()),
	})
	require.NoError(t, err)

	assert.Equal(t, FreshmanApplicationPending, updated.Status)
	assert.Equal(t, "image/png", store.contentType)
	assert.NotEmpty(t, store.objectKey)
	assert.NotEmpty(t, store.content)
}

func TestFreshmanCameraCaptureRejectsExpiredLinkedSessionBeforeStoringMaterial(t *testing.T) {
	fixture := postgresfixture.Start(t)
	store := &testAdmissionMaterialStore{}
	svc := newFreshmanTestService(t, fixture)
	svc.materialStore = store
	userID := seedLinkedAdmissionUser(t, fixture, svc, "freshman-camera-expired")
	app := createFreshmanTestApplication(t, svc, userID)

	svc.now = func() time.Time { return fixedAdmissionNow().Add(25 * time.Hour) }

	_, err := svc.SubmitCameraCapture(context.Background(), CameraCaptureInput{
		UserID:        userID,
		ApplicationID: app.ID,
		ContentType:   "image/png",
		ImageBase64:   base64.StdEncoding.EncodeToString(validPNGBytes()),
	})
	require.ErrorIs(t, err, ErrAdmissionTokenExpired)
	assert.Empty(t, store.objectKey)
}

func TestFreshmanCameraHandoffUploadsAndLocksContinuation(t *testing.T) {
	fixture := postgresfixture.Start(t)
	store := &testAdmissionMaterialStore{}
	svc := newFreshmanTestService(t, fixture)
	svc.materialStore = store
	tokenIndex := 0
	svc.generateToken = func() (string, error) {
		tokenIndex++
		return fmt.Sprintf("freshman-camera-token-%d", tokenIndex), nil
	}
	userID := seedLinkedAdmissionUser(t, fixture, svc, "freshman-camera-handoff")
	app := createFreshmanTestApplication(t, svc, userID)

	handoff, err := svc.CreateFreshmanCameraHandoff(context.Background(), FreshmanCameraHandoffCreateInput{
		UserID:        userID,
		ApplicationID: app.ID,
	})
	require.NoError(t, err)
	require.NotEmpty(t, handoff.ID)
	assert.Equal(t, FreshmanCameraHandoffPending, handoff.Status)
	assert.Contains(t, handoff.MobileURL, "/admission/freshman/camera/")
	token := handoff.MobileURL[strings.LastIndex(handoff.MobileURL, "/")+1:]

	preview, err := svc.PreviewFreshmanCameraHandoff(
		context.Background(),
		token,
	)
	require.NoError(t, err)
	assert.Equal(t, handoff.ID, preview.ID)

	regenerated, err := svc.CreateFreshmanCameraHandoff(context.Background(), FreshmanCameraHandoffCreateInput{
		UserID:        userID,
		ApplicationID: app.ID,
	})
	require.NoError(t, err)
	assert.NotEqual(t, handoff.ID, regenerated.ID)
	assert.Equal(t, FreshmanCameraHandoffPending, regenerated.Status)
	token = regenerated.MobileURL[strings.LastIndex(regenerated.MobileURL, "/")+1:]

	uploaded, err := svc.SubmitFreshmanCameraHandoffCapture(context.Background(), FreshmanCameraHandoffCaptureInput{
		Token:       token,
		ContentType: "image/png",
		ImageBase64: base64.StdEncoding.EncodeToString(validPNGBytes()),
	})
	require.NoError(t, err)
	assert.Equal(t, FreshmanCameraHandoffUploaded, uploaded.Status)
	assert.NotNil(t, uploaded.UploadedAt)
	assert.NotEmpty(t, store.objectKey)

	_, err = svc.SubmitCameraCapture(context.Background(), CameraCaptureInput{
		UserID:        userID,
		ApplicationID: app.ID,
		ContentType:   "image/png",
		ImageBase64:   base64.StdEncoding.EncodeToString(validPNGBytes()),
	})
	require.ErrorIs(t, err, ErrAdmissionCameraHandoffLocked)

	retried, err := svc.SubmitFreshmanCameraHandoffCapture(context.Background(), FreshmanCameraHandoffCaptureInput{
		Token:       token,
		ContentType: "image/png",
		ImageBase64: base64.StdEncoding.EncodeToString(validPNGBytes()),
	})
	require.NoError(t, err)
	assert.Equal(t, FreshmanCameraHandoffUploaded, retried.Status)
	assert.Equal(t, uploaded.ID, retried.ID)

	locked, err := svc.ChooseFreshmanCameraHandoffContinuation(context.Background(), FreshmanCameraHandoffContinuationInput{
		Token:      token,
		ContinueOn: FreshmanCameraContinueDesktop,
	})
	require.NoError(t, err)
	assert.Equal(t, FreshmanCameraHandoffLocked, locked.Status)
	require.NotNil(t, locked.ContinueOn)
	assert.Equal(t, FreshmanCameraContinueDesktop, *locked.ContinueOn)
	require.NotNil(t, locked.ChosenAt)

	_, err = svc.ChooseFreshmanCameraHandoffContinuation(context.Background(), FreshmanCameraHandoffContinuationInput{
		Token:      token,
		ContinueOn: FreshmanCameraContinueMobile,
	})
	require.ErrorIs(t, err, ErrAdmissionCameraHandoffLocked)
}

func TestFreshmanCameraHandoffContinuationRaceReturnsLocked(t *testing.T) {
	fixture := postgresfixture.Start(t)
	store := &testAdmissionMaterialStore{}
	svc := newFreshmanTestService(t, fixture)
	svc.materialStore = store
	svc.generateToken = func() (string, error) {
		return "freshman-camera-continuation-race-token", nil
	}
	userID := seedLinkedAdmissionUser(t, fixture, svc, "freshman-camera-continuation-race")
	app := createFreshmanTestApplication(t, svc, userID)

	handoff, err := svc.CreateFreshmanCameraHandoff(context.Background(), FreshmanCameraHandoffCreateInput{
		UserID:        userID,
		ApplicationID: app.ID,
	})
	require.NoError(t, err)
	token := handoff.MobileURL[strings.LastIndex(handoff.MobileURL, "/")+1:]
	uploaded, err := svc.SubmitFreshmanCameraHandoffCapture(context.Background(), FreshmanCameraHandoffCaptureInput{
		Token:       token,
		ContentType: "image/png",
		ImageBase64: base64.StdEncoding.EncodeToString(validPNGBytes()),
	})
	require.NoError(t, err)

	raced := false
	svc.beforeFreshmanCameraHandoffContinuationChoose = func() {
		if raced {
			return
		}
		raced = true
		locked, lockErr := svc.repo.ChooseFreshmanCameraHandoffContinuation(
			context.Background(),
			uploaded.ID,
			FreshmanCameraContinueDesktop,
			fixedAdmissionNow(),
		)
		require.NoError(t, lockErr)
		assert.Equal(t, FreshmanCameraHandoffLocked, locked.Status)
	}

	_, err = svc.ChooseFreshmanCameraHandoffContinuation(context.Background(), FreshmanCameraHandoffContinuationInput{
		Token:      token,
		ContinueOn: FreshmanCameraContinueMobile,
	})

	require.ErrorIs(t, err, ErrAdmissionCameraHandoffLocked)
	assert.True(t, raced)
}

func TestFreshmanCameraHandoffUploadRaceRecoversUploadedState(t *testing.T) {
	fixture := postgresfixture.Start(t)
	store := &testAdmissionMaterialStore{}
	svc := newFreshmanTestService(t, fixture)
	svc.materialStore = store
	svc.generateToken = func() (string, error) {
		return "freshman-camera-upload-race-token", nil
	}
	userID := seedLinkedAdmissionUser(t, fixture, svc, "freshman-camera-upload-race")
	app := createFreshmanTestApplication(t, svc, userID)

	handoff, err := svc.CreateFreshmanCameraHandoff(context.Background(), FreshmanCameraHandoffCreateInput{
		UserID:        userID,
		ApplicationID: app.ID,
	})
	require.NoError(t, err)
	token := handoff.MobileURL[strings.LastIndex(handoff.MobileURL, "/")+1:]
	raced := false
	svc.beforeFreshmanCameraHandoffMarkUploaded = func() {
		if raced {
			return
		}
		raced = true
		require.NoError(
			t,
			svc.repo.MarkFreshmanCameraHandoffUploaded(
				context.Background(),
				handoff.ID,
				fixedAdmissionNow(),
			),
		)
	}

	uploaded, err := svc.SubmitFreshmanCameraHandoffCapture(context.Background(), FreshmanCameraHandoffCaptureInput{
		Token:       token,
		ContentType: "image/png",
		ImageBase64: base64.StdEncoding.EncodeToString(validPNGBytes()),
	})

	require.NoError(t, err)
	assert.True(t, raced)
	assert.Equal(t, handoff.ID, uploaded.ID)
	assert.Equal(t, FreshmanCameraHandoffUploaded, uploaded.Status)
	assert.NotNil(t, uploaded.UploadedAt)
}

func TestFreshmanCameraHandoffReusesConcurrentActiveHandoff(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newFreshmanTestService(t, fixture)
	tokenIndex := 0
	svc.generateToken = func() (string, error) {
		tokenIndex++
		return fmt.Sprintf("freshman-camera-race-token-%d", tokenIndex), nil
	}
	userID := seedLinkedAdmissionUser(t, fixture, svc, "freshman-camera-race")
	app := createFreshmanTestApplication(t, svc, userID)
	inserted := false
	svc.beforeFreshmanCameraHandoffCreate = func() {
		if inserted {
			return
		}
		inserted = true
		insertFreshmanCameraHandoff(t, fixture, freshmanCameraHandoffSeed{
			ID:            "freshman-camera-race-existing",
			ApplicationID: app.ID,
			UserID:        userID,
			TokenHash:     svc.hashToken("freshman-camera-race-existing-token"),
			Status:        FreshmanCameraHandoffPending,
			ExpiresAt:     fixedAdmissionNow().Add(freshmanCameraHandoffTTL),
		})
	}

	handoff, err := svc.CreateFreshmanCameraHandoff(context.Background(), FreshmanCameraHandoffCreateInput{
		UserID:        userID,
		ApplicationID: app.ID,
	})

	require.NoError(t, err)
	require.NotNil(t, handoff)
	assert.Equal(t, "freshman-camera-race-existing", handoff.ID)
	assert.Equal(t, FreshmanCameraHandoffPending, handoff.Status)
}

func TestFreshmanDesktopCaptureExpiresPendingMobileHandoff(t *testing.T) {
	fixture := postgresfixture.Start(t)
	store := &testAdmissionMaterialStore{}
	svc := newFreshmanTestService(t, fixture)
	svc.materialStore = store
	svc.generateToken = func() (string, error) {
		return "freshman-camera-token", nil
	}
	userID := seedLinkedAdmissionUser(t, fixture, svc, "freshman-camera-desktop-after-handoff")
	app := createFreshmanTestApplication(t, svc, userID)

	handoff, err := svc.CreateFreshmanCameraHandoff(context.Background(), FreshmanCameraHandoffCreateInput{
		UserID:        userID,
		ApplicationID: app.ID,
	})
	require.NoError(t, err)
	token := handoff.MobileURL[strings.LastIndex(handoff.MobileURL, "/")+1:]

	updated, err := svc.SubmitCameraCapture(context.Background(), CameraCaptureInput{
		UserID:        userID,
		ApplicationID: app.ID,
		ContentType:   "image/png",
		ImageBase64:   base64.StdEncoding.EncodeToString(validPNGBytes()),
	})
	require.NoError(t, err)
	assert.Equal(t, FreshmanApplicationPending, updated.Status)
	assert.NotEmpty(t, store.objectKey)

	preview, err := svc.PreviewFreshmanCameraHandoff(context.Background(), token)
	require.NoError(t, err)
	assert.Equal(t, FreshmanCameraHandoffExpired, preview.Status)
}

func TestAdmissionMVPFreshmanMaterialFlowReleasesVerifiedMember(t *testing.T) {
	fixture := postgresfixture.Start(t)
	store := &testAdmissionMaterialStore{}
	svc := newOperatorTestService(t, fixture)
	svc.materialStore = store
	operatorID := seedAdmissionUser(t, fixture, "mvp-freshman-operator")
	bindAdmissionOperatorQQ(t, fixture, operatorBindingSeed{UserID: operatorID, QQID: "90004"})
	svc.operatorAccess = &testOperatorAccessGateway{allowedUserID: operatorID}

	created, err := svc.CreateBotSession(context.Background(), BotSessionCreateInput{
		Platform:  "qq",
		BotSelfID: "514",
		GuildID:   "guild-1",
		ChannelID: "channel-1",
		QQID:      "10001",
	})
	require.NoError(t, err)
	assert.Equal(t, "https://join.stuhelper.com/verify/test-admission-token", created.AuthURL)

	userID := seedAdmissionUser(t, fixture, "mvp-freshman")
	_, err = svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:  created.Token,
		UserID: userID,
	})
	require.NoError(t, err)
	app, err := svc.CreateFreshmanApplication(context.Background(), FreshmanApplicationCreateInput{
		UserID:        userID,
		SchoolID:      4111010006,
		ApplicantName: "Alice Applicant",
		MaterialType:  MaterialAdmissionNotice,
	})
	require.NoError(t, err)
	_, err = svc.SubmitCameraCapture(context.Background(), CameraCaptureInput{
		UserID:        userID,
		ApplicationID: app.ID,
		ContentType:   "image/png",
		ImageBase64:   base64.StdEncoding.EncodeToString(validPNGBytes()),
	})
	require.NoError(t, err)

	reviewed, err := svc.ReviewFreshmanApplicationFromBot(
		context.Background(),
		botReviewInput(app.ID, "90004"),
	)
	require.NoError(t, err)
	assert.Equal(t, FreshmanApplicationApproved, reviewed.Status)

	actions, err := svc.ListPendingAdmissionActions(context.Background(), AdmissionPendingActionFilter{
		Platform:  "qq",
		BotSelfID: "514",
	})
	require.NoError(t, err)
	require.Len(t, actions, 1)
	assert.Equal(t, created.Session.ID, actions[0].SessionID)
	assert.Equal(t, BotActionRelease, actions[0].Action)
	assert.Equal(t, created.AuthURL, actions[0].AuthURL)

	err = svc.RecordBotEvent(context.Background(), actions[0].SessionID, BotEventInput{
		Action:  actions[0].Action,
		Success: true,
	})
	require.NoError(t, err)
	assertAdmissionSessionCancelled(t, fixture, created.Session.ID)
}

func newFreshmanTestService(t *testing.T, fixture *postgresfixture.Fixture) *Service {
	t.Helper()
	svc := newSessionTestService(t, fixture)
	insertAdmissionSchoolConfig(t, fixture)
	insertAdmissionPolicy(t, fixture)
	return svc
}

func seedLinkedAdmissionUser(t *testing.T, fixture *postgresfixture.Fixture, svc *Service, suffix string) int64 {
	t.Helper()
	userID := seedAdmissionUser(t, fixture, suffix)
	created := createLinkableSession(t, svc)
	_, err := svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:  created.Token,
		UserID: userID,
	})
	require.NoError(t, err)
	return userID
}

func createFreshmanTestApplication(t *testing.T, svc *Service, userID int64) *FreshmanApplication {
	t.Helper()
	app, err := svc.CreateFreshmanApplication(context.Background(), FreshmanApplicationCreateInput{
		UserID:        userID,
		SchoolID:      4111010006,
		ApplicantName: "Alice Applicant",
		MaterialType:  MaterialAdmissionNotice,
	})
	require.NoError(t, err)
	return app
}

func closeFreshmanChannel(t *testing.T, fixture *postgresfixture.Fixture) {
	t.Helper()
	_, err := fixture.Pool.Exec(context.Background(), `
		UPDATE group_admission_policies
		SET freshman_channel_closes_at = $1
		WHERE platform = 'qq' AND guild_id = 'guild-1'
	`, fixedAdmissionNow().Add(-time.Hour))
	require.NoError(t, err)
}

func openFreshmanChannel(t *testing.T, fixture *postgresfixture.Fixture) {
	t.Helper()
	_, err := fixture.Pool.Exec(context.Background(), `
		UPDATE group_admission_policies
		SET freshman_channel_closes_at = $1
		WHERE platform = 'qq' AND guild_id = 'guild-1'
	`, futureTime(30))
	require.NoError(t, err)
}

type freshmanPendingSeed struct {
	ID            string
	UserID        int64
	SchoolID      int64
	SessionID     string
	ApplicantName string
}

type freshmanCameraHandoffSeed struct {
	ID            string
	ApplicationID string
	UserID        int64
	TokenHash     string
	Status        FreshmanCameraHandoffStatus
	ExpiresAt     time.Time
}

func insertFreshmanCameraHandoff(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	seed freshmanCameraHandoffSeed,
) {
	t.Helper()
	_, err := fixture.Pool.Exec(context.Background(), `
		INSERT INTO freshman_camera_handoffs (
			id, application_id, user_id, token_hash, status, expires_at
		)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, seed.ID, seed.ApplicationID, seed.UserID, seed.TokenHash, seed.Status, seed.ExpiresAt)
	require.NoError(t, err)
}

func insertFreshmanPendingApplication(
	t *testing.T,
	fixture *postgresfixture.Fixture,
	seed freshmanPendingSeed,
) {
	t.Helper()
	_, err := fixture.Pool.Exec(context.Background(), `
		INSERT INTO freshman_verification_applications (
			id, user_id, school_id, admission_session_id, applicant_name,
			applicant_name_masked, material_type, status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, seed.ID, seed.UserID, seed.SchoolID, seed.SessionID, seed.ApplicantName,
		maskAdmissionName(seed.ApplicantName), MaterialAdmissionNotice, FreshmanApplicationPending)
	require.NoError(t, err)
}

type testAdmissionMaterialStore struct {
	objectKey   string
	content     []byte
	contentType string
}

func (s *testAdmissionMaterialStore) PutAdmissionMaterial(
	_ context.Context,
	objectKey string,
	content []byte,
	contentType string,
) error {
	s.objectKey = objectKey
	s.content = content
	s.contentType = contentType
	return nil
}

func (s *testAdmissionMaterialStore) DeleteAdmissionMaterial(_ context.Context, objectKey string) error {
	if s.objectKey == objectKey {
		s.objectKey = ""
		s.content = nil
		s.contentType = ""
	}
	return nil
}

func (s *testAdmissionMaterialStore) GetAdmissionMaterialURL(_ context.Context, objectKey string) (string, error) {
	return "https://materials.example/" + objectKey, nil
}

func insertAdmissionSchoolConfig(t *testing.T, fixture *postgresfixture.Fixture) {
	t.Helper()
	_, err := fixture.Pool.Exec(context.Background(), `
		INSERT INTO school_configs (
			school_id, school_name, verification_method, approval_policy, manual_form_fields, enabled
		)
		VALUES (
			4111010006, '北京航空航天大学', 'manual', 'auto',
			'{"admission":{"emailDomains":["buaa.edu.cn"],"ssoLoginURL":"https://sso.school.example/login"}}',
			true
		)
		ON CONFLICT (school_id) DO UPDATE
		SET manual_form_fields = EXCLUDED.manual_form_fields,
		    enabled = EXCLUDED.enabled,
		    updated_at = NOW()
	`)
	require.NoError(t, err)
}

func validPNGBytes() []byte {
	return []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
		0x89, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9c, 0x62, 0xf8, 0xcf, 0xc0, 0x00,
		0x00, 0x03, 0x01, 0x01, 0x00, 0x18, 0xdd, 0x8d,
		0xb0, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e,
		0x44, 0xae, 0x42, 0x60, 0x82,
	}
}
