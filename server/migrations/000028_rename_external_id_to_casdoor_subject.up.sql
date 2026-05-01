ALTER TABLE users RENAME COLUMN external_id TO casdoor_subject;
ALTER TABLE users RENAME CONSTRAINT users_external_id_key TO users_casdoor_subject_key;
