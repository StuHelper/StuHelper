WITH ranked AS (
    SELECT
        id,
        row_number() OVER (
            PARTITION BY application_id
            ORDER BY
                CASE status WHEN 'uploaded' THEN 0 ELSE 1 END,
                created_at DESC,
                id DESC
        ) AS rn
    FROM public.freshman_camera_handoffs
    WHERE status IN ('pending', 'uploaded')
)
UPDATE public.freshman_camera_handoffs handoff
SET status = 'expired',
    updated_at = now()
FROM ranked
WHERE handoff.id = ranked.id
  AND ranked.rn > 1;

CREATE UNIQUE INDEX freshman_camera_handoffs_active_application_idx
    ON public.freshman_camera_handoffs USING btree (application_id)
    WHERE status IN ('pending', 'uploaded');
