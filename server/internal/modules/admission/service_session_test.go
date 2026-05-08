package admission

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"git.stuhelper.com/StuHelper/StuHelper/internal/modules/user"
	"git.stuhelper.com/StuHelper/StuHelper/internal/testutil/postgresfixture"
)

func TestAdmissionSessionCreatePreviewAndMismatch(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)

	created, err := svc.CreateBotSession(context.Background(), BotSessionCreateInput{
		Platform: "qq", GuildID: "guild-1", ChannelID: "channel-1", QQID: "10001",
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.Token)
	assert.Equal(t, StatusJoinedMuted, created.Session.Status)
	assert.Equal(t, svc.now().Add(time.Hour), created.Session.LinkWaitDeadlineAt)

	preview, err := svc.PreviewToken(context.Background(), created.Token, "10001")
	require.NoError(t, err)
	assert.Equal(t, created.Session.ID, preview.ID)
	assert.Nil(t, preview.TokenConsumedAt)

	_, err = svc.PreviewToken(context.Background(), created.Token, "99999")
	require.ErrorIs(t, err, ErrAdmissionQQMismatch)
}

func TestAdmissionTokenLinkIsAtomicUnderConcurrency(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	created := createLinkableSession(t, svc)
	firstUser := seedAdmissionUser(t, fixture, "link-first")
	secondUser := seedAdmissionUser(t, fixture, "link-second")

	results := linkTokenConcurrently(t, svc, created.Token, firstUser, secondUser)
	require.Len(t, results, 2)
	assertExactlyOneLinkSuccess(t, results)
}

func TestAdmissionTokenExpiredAndConsumedErrors(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	created := createLinkableSession(t, svc)

	svc.now = func() time.Time { return fixedAdmissionNow().Add(2 * time.Hour) }
	_, err := svc.PreviewToken(context.Background(), created.Token, "10001")
	require.ErrorIs(t, err, ErrAdmissionTokenExpired)

	svc = newSessionTestService(t, fixture)
	linked, err := svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:   created.Token,
		QQQuery: "10001",
		UserID:  seedAdmissionUser(t, fixture, "late-link"),
	})
	require.NoError(t, err)
	assert.Equal(t, StatusLinked, linked.Status)

	_, err = svc.PreviewToken(context.Background(), created.Token, "10001")
	require.ErrorIs(t, err, ErrAdmissionTokenConsumed)
}

func TestAdmissionSessionStatusTransitions(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	created := createLinkableSession(t, svc)
	userID := seedAdmissionUser(t, fixture, "status-link")
	_, err := svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
		Token:   created.Token,
		QQQuery: "10001",
		UserID:  userID,
	})
	require.NoError(t, err)

	submitted, err := svc.MarkMaterialSubmitted(context.Background(), created.Session.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusMaterialSubmitted, submitted.Status)
	require.NotNil(t, submitted.ManualReviewDeadlineAt)

	verified, err := svc.MarkVerified(context.Background(), created.Session.ID)
	require.NoError(t, err)
	assert.Equal(t, StatusVerified, verified.Status)
	require.NotNil(t, verified.VerifiedAt)
}

func TestAdmissionFailureBlacklistFromKickEvent(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	tokenIndex := 0
	svc.generateToken = func() (string, error) {
		tokenIndex++
		return fmt.Sprintf("test-admission-token-%d", tokenIndex), nil
	}

	for i := 0; i < DefaultFailedJoinLimit; i++ {
		created := createLinkableSession(t, svc)
		err := svc.RecordBotEvent(context.Background(), created.Session.ID, BotEventInput{
			Action: BotActionKick, Success: true,
		})
		require.NoError(t, err)
	}

	assertAdmissionFailureBlacklisted(t, fixture, "10001")
}

func TestAdmissionFailureIgnoresDuplicateKickEventForSession(t *testing.T) {
	fixture := postgresfixture.Start(t)
	svc := newSessionTestService(t, fixture)
	insertAdmissionPolicy(t, fixture)
	created := createLinkableSession(t, svc)

	for i := 0; i < 2; i++ {
		err := svc.RecordBotEvent(context.Background(), created.Session.ID, BotEventInput{
			Action: BotActionKick, Success: true,
		})
		require.NoError(t, err)
	}

	var failureCount int
	err := fixture.Pool.QueryRow(context.Background(), `
		SELECT failure_count
		FROM group_admission_failures
		WHERE platform = 'qq' AND guild_id = 'guild-1' AND qq_id = '10001'
	`).Scan(&failureCount)
	require.NoError(t, err)
	assert.Equal(t, 1, failureCount)
}

type linkResult struct {
	session *AdmissionSession
	err     error
}

func linkTokenConcurrently(t *testing.T, svc *Service, token string, users ...int64) []linkResult {
	t.Helper()
	var wg sync.WaitGroup
	results := make([]linkResult, len(users))
	for i, userID := range users {
		wg.Add(1)
		go func(index int, id int64) {
			defer wg.Done()
			session, err := svc.LinkTokenToUser(context.Background(), AdmissionTokenLinkInput{
				Token:   token,
				QQQuery: "10001",
				UserID:  id,
			})
			results[index] = linkResult{session: session, err: err}
		}(i, userID)
	}
	wg.Wait()
	return results
}

func assertExactlyOneLinkSuccess(t *testing.T, results []linkResult) {
	t.Helper()
	var success int
	var consumed int
	for _, result := range results {
		if result.err == nil {
			success++
			assert.Equal(t, StatusLinked, result.session.Status)
			continue
		}
		if errors.Is(result.err, ErrAdmissionTokenConsumed) {
			consumed++
		}
	}
	assert.Equal(t, 1, success)
	assert.Equal(t, 1, consumed)
}

func createLinkableSession(t *testing.T, svc *Service) *CreatedAdmissionSession {
	t.Helper()
	created, err := svc.CreateBotSession(context.Background(), BotSessionCreateInput{
		Platform: "qq", GuildID: "guild-1", ChannelID: "channel-1", QQID: "10001",
	})
	require.NoError(t, err)
	return created
}

func newSessionTestService(t *testing.T, fixture *postgresfixture.Fixture) *Service {
	t.Helper()
	svc, err := NewService(NewRepository(fixture.DB), &testQQBindingGateway{}, []byte("test-admission-hmac-key-32-bytes!"))
	require.NoError(t, err)
	svc.now = fixedAdmissionNow
	svc.generateToken = func() (string, error) { return "test-admission-token", nil }
	return svc
}

func fixedAdmissionNow() time.Time {
	return time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
}

type testQQBindingGateway struct{}

func (g *testQQBindingGateway) EnsureQQBindingForUserTx(
	context.Context,
	pgx.Tx,
	int64,
	string,
	*string,
) (*user.QQBinding, error) {
	return &user.QQBinding{}, nil
}

func assertAdmissionFailureBlacklisted(t *testing.T, fixture *postgresfixture.Fixture, qqID string) {
	t.Helper()
	var failureCount int
	var blacklistedAt *time.Time
	err := fixture.Pool.QueryRow(context.Background(), `
		SELECT failure_count, blacklisted_at
		FROM group_admission_failures
		WHERE platform = 'qq' AND guild_id = 'guild-1' AND qq_id = $1
	`, qqID).Scan(&failureCount, &blacklistedAt)
	require.NoError(t, err)
	assert.Equal(t, DefaultFailedJoinLimit, failureCount)
	assert.NotNil(t, blacklistedAt)
}
