ALTER TABLE public.group_admission_policies
  ADD COLUMN guard_enabled boolean NOT NULL DEFAULT TRUE;

COMMENT ON COLUMN public.group_admission_policies.guard_enabled IS 'Whether this admission policy should be synchronized as an active Koishi guard target.';
