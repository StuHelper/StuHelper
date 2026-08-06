package studentverification

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *Repository) ProgressApplicationMethodTx(
	ctx context.Context,
	tx pgx.Tx,
	applicationID string,
	method Method,
	privacyNoticeVersion string,
	now time.Time,
) error {
	result, err := tx.Exec(ctx, `
		UPDATE student_verification_applications
		SET status = 'in_progress', current_method = $2,
		    privacy_notice_version = $3, consented_at = $4,
		    revision = revision + 1, updated_at = $4
		WHERE id = $1
		  AND status IN ('created', 'in_progress', 'pending_manual_review')
		  AND expires_at > $4
	`, applicationID, method, privacyNoticeVersion, now)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return ErrApplicationState
	}
	return nil
}

func (r *Repository) CancelWaitingInboundEmailChallengesTx(
	ctx context.Context,
	tx pgx.Tx,
	applicationID string,
	now time.Time,
) error {
	_, err := tx.Exec(ctx, `
		UPDATE student_email_inbound_challenges
		SET status = 'cancelled', updated_at = $2
		WHERE application_id = $1 AND status = 'waiting'
	`, applicationID, now)
	return err
}

func (r *Repository) InsertInboundEmailChallengeTx(
	ctx context.Context,
	tx pgx.Tx,
	challenge storedInboundEmailChallenge,
) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO student_email_inbound_challenges (
		    id, application_id, user_id, school_id, status,
		    target_address, expected_sender_hash, expected_sender_display,
		    routing_subject, challenge_value_enc, challenge_value_hash,
		    encryption_key_version, hmac_key_version,
		    student_id_hash, name_hash, enrollment_subject_hash,
		    student_id_display, roster_snapshot_id, roster_snapshot_revision,
		    privacy_notice_version, expires_at, created_at, updated_at
		)
		VALUES (
		    $1, $2, $3, $4, 'waiting',
		    $5, $6, $7,
		    $8, $9, $10,
		    $11, $12,
		    $13, $14, $15,
		    $16, $17, $18,
		    $19, $20, $21, $21
		)
	`,
		challenge.ID, challenge.ApplicationID, challenge.UserID, challenge.SchoolID,
		challenge.TargetAddress, challenge.ExpectedSenderHash, challenge.ExpectedSenderMasked,
		challenge.Subject, challenge.ChallengeValueEnc, challenge.ChallengeValueHash,
		challenge.EncryptionKeyVersion, challenge.HMACKeyVersion,
		challenge.StudentIDHash, challenge.NameHash, challenge.EnrollmentSubjectHash,
		challenge.StudentIDDisplay, challenge.SnapshotID, challenge.SnapshotRevision,
		challenge.PrivacyNoticeVersion, challenge.ExpiresAt, challenge.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert inbound email challenge: %w", err)
	}
	return nil
}

func (r *Repository) GetInboundEmailChallenge(
	ctx context.Context,
	applicationID string,
	userID int64,
) (*storedInboundEmailChallenge, error) {
	ctx = withTable(ctx, "student_email_inbound_challenges")
	return scanStoredInboundEmailChallenge(r.db.QueryRow(ctx, inboundEmailChallengeSelectSQL()+`
		WHERE challenge.application_id = $1 AND challenge.user_id = $2
		ORDER BY challenge.created_at DESC, challenge.id DESC
		LIMIT 1
	`, applicationID, userID))
}

func (r *Repository) GetInboundEmailChallengeBySubjectForUpdateTx(
	ctx context.Context,
	tx pgx.Tx,
	routingSubject string,
) (*storedInboundEmailChallenge, error) {
	return scanStoredInboundEmailChallenge(tx.QueryRow(ctx, inboundEmailChallengeSelectSQL()+`
		WHERE challenge.routing_subject = $1
		FOR UPDATE OF challenge
	`, routingSubject))
}

func inboundEmailChallengeSelectSQL() string {
	return `
		SELECT challenge.id, challenge.application_id, challenge.user_id,
		       challenge.school_id, challenge.status, challenge.target_address,
		       challenge.expected_sender_hash, challenge.expected_sender_display,
		       challenge.routing_subject, challenge.challenge_value_enc,
		       challenge.challenge_value_hash, challenge.encryption_key_version,
		       challenge.hmac_key_version, challenge.student_id_hash,
		       challenge.name_hash, challenge.enrollment_subject_hash,
		       challenge.student_id_display, challenge.roster_snapshot_id,
		       challenge.roster_snapshot_revision, challenge.privacy_notice_version,
		       challenge.message_reference_hash, challenge.expires_at,
		       challenge.verified_at, challenge.created_at
		FROM student_email_inbound_challenges challenge
	`
}

func scanStoredInboundEmailChallenge(row rowScanner) (*storedInboundEmailChallenge, error) {
	var challenge storedInboundEmailChallenge
	err := row.Scan(
		&challenge.ID, &challenge.ApplicationID, &challenge.UserID,
		&challenge.SchoolID, &challenge.Status, &challenge.TargetAddress,
		&challenge.ExpectedSenderHash, &challenge.ExpectedSenderMasked,
		&challenge.Subject, &challenge.ChallengeValueEnc,
		&challenge.ChallengeValueHash, &challenge.EncryptionKeyVersion,
		&challenge.HMACKeyVersion, &challenge.StudentIDHash, &challenge.NameHash,
		&challenge.EnrollmentSubjectHash, &challenge.StudentIDDisplay,
		&challenge.SnapshotID, &challenge.SnapshotRevision,
		&challenge.PrivacyNoticeVersion, &challenge.MessageReferenceHash,
		&challenge.ExpiresAt, &challenge.VerifiedAt, &challenge.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &challenge, nil
}

func (r *Repository) InboundEmailEventExistsTx(
	ctx context.Context,
	tx pgx.Tx,
	eventReferenceHash string,
) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS (
		    SELECT 1 FROM student_email_inbound_events
		    WHERE event_reference_hash = $1
		)
	`, eventReferenceHash).Scan(&exists)
	return exists, err
}

func (r *Repository) InsertInboundEmailEventTx(
	ctx context.Context,
	tx pgx.Tx,
	eventReferenceHash string,
	challengeID *string,
	senderVerified bool,
	mailAuthenticationVerified bool,
	challengeVerified bool,
	resultCode string,
	receivedAt time.Time,
	processedAt time.Time,
) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO student_email_inbound_events (
		    event_reference_hash, challenge_id, signature_verified,
		    sender_alignment_verified, mail_authentication_verified,
		    challenge_verified, result_code, received_at, processed_at, created_at
		)
		VALUES ($1, $2, true, $3, $4, $5, $6, $7, $8, $8)
		ON CONFLICT (event_reference_hash) DO NOTHING
	`, eventReferenceHash, challengeID, senderVerified, mailAuthenticationVerified,
		challengeVerified, resultCode, receivedAt, processedAt)
	return err
}

func (r *Repository) CompleteInboundEmailChallengeTx(
	ctx context.Context,
	tx pgx.Tx,
	challengeID string,
	messageReferenceHash string,
	now time.Time,
) error {
	result, err := tx.Exec(ctx, `
		UPDATE student_email_inbound_challenges
		SET status = 'verified', message_reference_hash = $2,
		    verified_at = $3, updated_at = $3
		WHERE id = $1 AND status = 'waiting' AND expires_at > $3
	`, challengeID, messageReferenceHash, now)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return ErrApplicationState
	}
	return nil
}

func (r *Repository) ExpireInboundEmailChallengeTx(
	ctx context.Context,
	tx pgx.Tx,
	challengeID string,
	now time.Time,
) error {
	_, err := tx.Exec(ctx, `
		UPDATE student_email_inbound_challenges
		SET status = 'expired', updated_at = $2
		WHERE id = $1 AND status = 'waiting'
	`, challengeID, now)
	return err
}
