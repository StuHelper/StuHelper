INSERT INTO public.system_configs (key, value, description)
VALUES
    (
        'email.delivery_policy',
        '{"mode":"priority","maxAttempts":2,"providers":[{"name":"tencent_ses","enabled":true,"priority":10,"weight":100},{"name":"resend","enabled":true,"priority":20,"weight":100}]}',
        '邮件发送提供商策略；mode=priority/weighted，priority 越小越优先，weight 用于同优先级负载均衡'
    )
ON CONFLICT (key) DO UPDATE
SET
    value = EXCLUDED.value,
    description = EXCLUDED.description,
    updated_at = NOW();
