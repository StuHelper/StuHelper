BEGIN;

UPDATE domain_event_outbox
SET stream = 'iam_openfga_tuple_sync',
    updated_at = NOW()
WHERE stream = 'iam_casdoor_user_projection'
  AND job_type = 'user_profile_projection';

COMMIT;
