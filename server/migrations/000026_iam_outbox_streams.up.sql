BEGIN;

UPDATE domain_event_outbox
SET stream = 'iam_casdoor_role_sync'
WHERE stream = 'user_external_sync'
  AND job_type = 'verified_student_role';

UPDATE domain_event_outbox
SET stream = 'iam_openfga_tuple_sync'
WHERE stream = 'user_external_sync'
  AND job_type = 'user_profile_projection';

UPDATE domain_event_outbox
SET stream = 'iam_openfga_tuple_sync'
WHERE stream = 'review_fga_sync';

COMMIT;
