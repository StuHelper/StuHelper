ALTER TABLE users RENAME CONSTRAINT users_casdoor_subject_key TO users_external_id_key;
ALTER TABLE users RENAME COLUMN casdoor_subject TO external_id;
