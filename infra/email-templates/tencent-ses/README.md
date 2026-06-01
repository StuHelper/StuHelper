# Tencent Cloud SES Templates

This directory stores Tencent Cloud SES templates used by StuHelper.

## stuhelper-school-email-otp

Purpose: transactional OTP email for school email verification in the StuHelper
student verification and admission flows.

Recommended Tencent Cloud SES template name:

```text
stuhelper-school-email-otp
```

Recommended subject:

```text
学生认证验证码
```

Production configuration:

```dotenv
EMAIL_ENABLED=true
EMAIL_DRIVER=multi
EMAIL_STUDENT_VERIFICATION_SUBJECT=学生认证验证码
EMAIL_FROM=noreply@notify.stuhelper.com
EMAIL_FROM_NAME=StuHelper 系统邮件
EMAIL_TENCENT_REGION=ap-guangzhou
EMAIL_TENCENT_ENDPOINT=ses.tencentcloudapi.com
EMAIL_TENCENT_TEMPLATE_ID=49779
EMAIL_TENCENT_TEMPLATE_PURPOSE=学校邮箱认证
EMAIL_TENCENT_TEMPLATE_SCHOOL_NAME=北京航空航天大学
EMAIL_TENCENT_TEMPLATE_EXPIRE_MINUTES=5
EMAIL_RESEND_ENDPOINT=https://api.resend.com/emails
```

`EMAIL_TENCENT_SECRET_ID` and `EMAIL_TENCENT_SECRET_KEY` are production
secrets. Keep them in `.env.prod.secrets.local` or the deployment secret store;
do not commit real values.

`EMAIL_RESEND_API_KEY` is also a production secret. Resend does not use the
Tencent template ID; StuHelper sends HTML directly through Resend and keeps
Tencent SES as the priority provider by default.

For a real Resend channel smoke, run:

```bash
RESEND_EMAIL_SMOKE_TO=<recipient-email> ./infra/ops/resend-email-channel-smoke.sh
```

The smoke sends `stuhelper-school-email-otp.html` as Resend `html` and
`stuhelper-school-email-otp.txt` as Resend `text`. The generated evidence stores
only the recipient domain, recipient hash prefix, and Resend email ID.

Template variables:

```text
{{code}}
{{expire_minutes}}
{{purpose}}
{{school_name}}
```

Example `TemplateData` for `SendEmail`:

```json
{"code":"123456","expire_minutes":"5","purpose":"学校邮箱认证","school_name":"北京航空航天大学"}
```

Tencent Cloud SES expects `Template.TemplateData` to be a JSON string. Template
variables in the template use `{{key}}`, and each value is supplied by the
matching key in `TemplateData`.

When using the Tencent Cloud API to create or update a template, `Html` and
`Text` in `TemplateContent` must be Base64 encoded:

```bash
base64 -w0 infra/email-templates/tencent-ses/stuhelper-school-email-otp.html
base64 -w0 infra/email-templates/tencent-ses/stuhelper-school-email-otp.txt
```

The template intentionally contains no external images, scripts, forms,
tracking pixels, or marketing copy. It is a transactional verification email.

Relevant Tencent Cloud documentation:

- Template structure and `TemplateData`: https://cloud.tencent.com/document/product/1288/51053
- `SendEmail` template example: https://cloud.tencent.com/document/api/1288/51034
- `GetEmailTemplate` status and Base64 response fields: https://cloud.tencent.com/document/api/1288/51040
- `UpdateEmailTemplate` example: https://cloud.tencent.com/document/api/1288/51038
- Resend send email API: https://resend.com/docs/api-reference/emails/send-email
