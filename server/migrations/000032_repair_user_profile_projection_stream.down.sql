BEGIN;

UPDATE domain_event_outbox
SET stream = 'iam_casdoor_user_projection',
    updated_at = NOW()
WHERE stream = 'iam_openfga_tuple_sync'
  AND job_type = 'user_profile_projection';

COMMIT;
