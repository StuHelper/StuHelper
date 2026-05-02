BEGIN;

UPDATE domain_event_outbox
SET stream = 'user_external_sync'
WHERE stream IN ('iam_casdoor_role_sync', 'iam_casdoor_user_projection');

UPDATE domain_event_outbox
SET stream = 'user_external_sync'
WHERE stream = 'iam_openfga_tuple_sync'
  AND job_type = 'user_profile_projection';

UPDATE domain_event_outbox
SET stream = 'review_fga_sync'
WHERE stream = 'iam_openfga_tuple_sync'
  AND job_type <> 'user_profile_projection';

COMMIT;
