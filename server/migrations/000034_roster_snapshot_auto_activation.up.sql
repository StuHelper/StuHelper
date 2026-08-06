ALTER TABLE public.school_verification_profiles
    ADD COLUMN snapshot_auto_activate boolean NOT NULL DEFAULT false;

COMMENT ON COLUMN public.school_verification_profiles.snapshot_auto_activate IS
    'When explicitly enabled, a newly imported full snapshot is activated only after every quality gate passes; defaults fail closed';
