DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM public.campus_connector_nodes LIMIT 1)
       OR EXISTS (SELECT 1 FROM public.campus_connector_requests LIMIT 1)
       OR EXISTS (SELECT 1 FROM public.campus_connector_snapshot_uploads LIMIT 1)
    THEN
        RAISE EXCEPTION
            'refusing destructive rollback: campus connector registry contains configured nodes or audit facts';
    END IF;
END $$;

DROP TABLE IF EXISTS public.campus_connector_node_events;
DROP TABLE IF EXISTS public.campus_connector_snapshot_uploads;
DROP TABLE IF EXISTS public.campus_connector_requests;
DROP TABLE IF EXISTS public.campus_connector_school_operations;
DROP TABLE IF EXISTS public.campus_connector_nodes;
