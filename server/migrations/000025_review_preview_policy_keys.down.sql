INSERT INTO public.system_configs (key, value, description)
SELECT
    'review_preview_title_chars',
    value,
    '评课标题预览最大字符数'
FROM public.system_configs
WHERE key = 'review_guest_preview_content_chars'
ON CONFLICT (key) DO NOTHING;

DELETE FROM public.system_configs
WHERE key = 'review_guest_preview_content_chars';

UPDATE public.system_configs
SET description = '评课正文预览最大字符数'
WHERE key = 'review_preview_content_chars';

UPDATE public.system_configs
SET description = '评课正文预览最大展示比例（1-100）'
WHERE key = 'review_preview_content_percent';
