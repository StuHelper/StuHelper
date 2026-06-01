CREATE TABLE public.freshman_camera_handoffs (
    id character varying(36) NOT NULL,
    application_id character varying(36) NOT NULL,
    user_id bigint NOT NULL,
    token_hash character varying(64) NOT NULL,
    status character varying(24) DEFAULT 'pending'::character varying NOT NULL,
    continue_on character varying(16),
    expires_at timestamp with time zone NOT NULL,
    uploaded_at timestamp with time zone,
    chosen_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_freshman_camera_handoffs_continue_on CHECK (
        continue_on IS NULL OR continue_on::text = ANY (ARRAY['desktop'::varchar, 'mobile'::varchar]::text[])
    ),
    CONSTRAINT chk_freshman_camera_handoffs_status CHECK (
        status::text = ANY (ARRAY['pending'::varchar, 'uploaded'::varchar, 'locked'::varchar, 'expired'::varchar]::text[])
    )
);

ALTER TABLE ONLY public.freshman_camera_handoffs
    ADD CONSTRAINT freshman_camera_handoffs_pkey PRIMARY KEY (id);

ALTER TABLE ONLY public.freshman_camera_handoffs
    ADD CONSTRAINT freshman_camera_handoffs_token_hash_key UNIQUE (token_hash);

CREATE INDEX freshman_camera_handoffs_application_idx
    ON public.freshman_camera_handoffs USING btree (application_id, created_at DESC);

CREATE INDEX freshman_camera_handoffs_user_idx
    ON public.freshman_camera_handoffs USING btree (user_id, created_at DESC);

ALTER TABLE ONLY public.freshman_camera_handoffs
    ADD CONSTRAINT freshman_camera_handoffs_application_id_fkey
    FOREIGN KEY (application_id) REFERENCES public.freshman_verification_applications(id) ON DELETE CASCADE;

ALTER TABLE ONLY public.freshman_camera_handoffs
    ADD CONSTRAINT freshman_camera_handoffs_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;
