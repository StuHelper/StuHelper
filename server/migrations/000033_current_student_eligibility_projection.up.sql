-- Shared, read-only projection for consumers that need current student facts
-- but must not read legacy user_profiles verification flags. The
-- student-verification service remains the write authority; this view exposes
-- only qualifying credential metadata and no raw identity evidence.

CREATE VIEW public.current_student_qualifying_credentials
WITH (security_barrier = true)
AS
SELECT
    credential.id,
    credential.user_id,
    credential.school_id,
    credential.kind,
    credential.credential_class,
    credential.verified_at,
    credential.expires_at,
    credential.revision
FROM public.user_verification_credentials credential
WHERE credential.status = 'active'
  AND credential.verification_application_id IS NOT NULL
  AND credential.revoked_at IS NULL
  AND (credential.expires_at IS NULL OR credential.expires_at > CURRENT_TIMESTAMP)
  AND NOT EXISTS (
      SELECT 1
      FROM public.student_subject_conflicts conflict
      WHERE conflict.school_id = credential.school_id
        AND conflict.subject_hash = credential.subject_hash
        AND conflict.status IN ('open', 'under_review')
  )
  AND (
      credential.roster_dependency = 'independent'
      OR (
          credential.roster_dependency = 'required'
          AND credential.enrollment_subject_id IS NOT NULL
          AND EXISTS (
              SELECT 1
              FROM public.student_enrollment_subjects subject
              JOIN academic.student_roster_active active
                ON active.school_id = subject.school_id
              JOIN academic.student_roster_snapshots snapshot
                ON snapshot.id = active.snapshot_id
               AND snapshot.school_id = active.school_id
              JOIN academic.student_roster_records record
                ON record.school_id = active.school_id
               AND record.snapshot_id = active.snapshot_id
               AND record.student_id_hash = subject.student_id_hash
              JOIN public.school_verification_profiles profile
                ON profile.school_id = subject.school_id
              WHERE subject.id = credential.enrollment_subject_id
                AND subject.binding_status = 'active'
                AND record.eligibility_status = 'eligible'
                AND snapshot.status = 'active'
                AND snapshot.source_cutoff_at <= CURRENT_TIMESTAMP + INTERVAL '5 minutes'
                AND snapshot.source_cutoff_at
                    + make_interval(secs => profile.snapshot_hard_expiry_seconds)
                    >= CURRENT_TIMESTAMP
          )
      )
      OR (
          credential.roster_dependency = 'conditional'
          AND credential.metadata @> '{"qualification_satisfied": true}'::jsonb
      )
  );

COMMENT ON VIEW public.current_student_qualifying_credentials IS
    'Current qualifying student credentials derived from target-state facts; never a stored verified boolean';

CREATE VIEW public.current_phone_gate_credentials
WITH (security_barrier = true)
AS
SELECT
    credential.id,
    credential.user_id,
    credential.method,
    credential.assurance,
    credential.verified_at,
    credential.expires_at,
    GREATEST(credential.revision, COALESCE(fence.revision, 1)) AS revision
FROM public.phone_verification_credentials credential
JOIN public.users account
  ON account.id = credential.user_id
 AND account.phone_projection_state = 'synced'
 AND account.phone_hash = credential.phone_hash
LEFT JOIN public.phone_eligibility_revisions fence
  ON fence.user_id = credential.user_id
WHERE credential.status = 'active'
  AND credential.revoked_at IS NULL
  AND (credential.expires_at IS NULL OR credential.expires_at > CURRENT_TIMESTAMP);

COMMENT ON VIEW public.current_phone_gate_credentials IS
    'Current publishing-phone gate projection; contains no full or masked phone number';
