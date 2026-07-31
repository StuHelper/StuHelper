package outbox

import "time"

const (
	StreamIAMCasdoorRoleSync              = "iam_casdoor_role_sync"
	StreamIAMCasdoorUserProjection        = "iam_casdoor_user_projection"
	StreamIAMOpenFGATupleSync             = "iam_openfga_tuple_sync"
	StreamIAMAuthorizationGrantProjection = "iam_authorization_grant_projection"
	StreamReviewNotification              = "review_notification"
	StreamReviewProjection                = "review_projection"

	IAMWorkerBatchSize      = 16
	IAMWorkerPollInterval   = 2 * time.Second
	IAMWorkerLockStaleAfter = 2 * time.Minute
	IAMWorkerRetryBase      = 5 * time.Second
	IAMWorkerMaxBackoff     = 5 * time.Minute
	IAMWorkerMaxAttempts    = 5
)

func IAMWorkerConfig(name string) WorkerConfig {
	return WorkerConfig{
		Name:             name,
		BatchSize:        IAMWorkerBatchSize,
		PollInterval:     IAMWorkerPollInterval,
		LockStaleAfter:   IAMWorkerLockStaleAfter,
		RetryBaseBackoff: IAMWorkerRetryBase,
		MaxBackoff:       IAMWorkerMaxBackoff,
		MaxAttempts:      IAMWorkerMaxAttempts,
	}
}
