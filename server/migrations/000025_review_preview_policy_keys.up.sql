INSERT INTO public.system_configs (key, value, description)
SELECT
    'review_guest_preview_content_chars',
    value,
    '游客评课正文单行预览最大字符数'
FROM public.system_configs
WHERE key = 'review_preview_title_chars'
ON CONFLICT (key) DO NOTHING;

INSERT INTO public.system_configs (key, value, description)
VALUES (
    'review_guest_preview_content_chars',
    '24',
    '游客评课正文单行预览最大字符数'
)
ON CONFLICT (key) DO NOTHING;

DELETE FROM public.system_configs
WHERE key = 'review_preview_title_chars';

UPDATE public.system_configs
SET description = '已登录但无完整查看权限时，评课正文单行预览最大字符数'
WHERE key = 'review_preview_content_chars';

UPDATE public.system_configs
SET description = '已登录但无完整查看权限时，评课正文单行预览最大展示比例（1-100）'
WHERE key = 'review_preview_content_percent';
