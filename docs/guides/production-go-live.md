---
type: guide
audience: ops
status: current
authoritative-source: docker-compose.prod.yml + infra/ops/*.sh + infra/nginx/baota-stuhelper.conf
last-verified: 2026-05-30
---

# 生产上线缺漏清单与执行指导

本文用于当前 B/2B 架构的生产上线：本地仓库、部署脚本、配置模板和 runbook 是唯一事实来源；生产环境只做可复现部署、smoke 和日志确认，不把生产手工修改当最终状态。

## 当前入口约定

- `stuhelper.com`：主站、后台、API、账号中心、学生认证、QQ 绑定、授权应用和开发者应用。
- `join.stuhelper.com`：加群验证业务域，唯一公开验证链接是 `https://join.stuhelper.com/verify/<token>?qq=<qq>`。
- `sso.stuhelper.com`：Casdoor，唯一公开登录认证系统和 OIDC issuer。

主站生产 Compose 只把业务服务绑定到回环地址，公网 `80/443` 只由宝塔 Nginx 监听。

默认回环端口：

```text
backend 127.0.0.1:18080
web     127.0.0.1:18000
admin   127.0.0.1:18001
```

## 上线前阻断项

| 项目 | 要求 |
|------|------|
| DNS | `stuhelper.com`、`www.stuhelper.com`、`join.stuhelper.com`、`sso.stuhelper.com` 指向对应公网入口；不再配置独立身份入口域名 |
| 宝塔 Nginx | 主站机合并 `infra/nginx/baota-stuhelper.conf`；Casdoor 入口按 `infra/nginx/baota-casdoor-sso.conf` 或等价配置反代 |
| 生产 env | 使用 `infra/ops/init-prod-env.sh` 生成模板，替换占位符；不得提交真实 `.env.prod.*` |
| Secret backend | 生产使用非 repo 的 secret backend；真实 token、密码、对象存储密钥不写入仓库 |
| PostgreSQL / Redis | 生产 PostgreSQL TLS 默认开启；Redis 使用 StuHelper 独立实例，不复用全局 Redis |
| 对象存储与备份 | 配置 HTTPS S3 兼容对象存储和备份对象存储，至少完成一次取回校验 |
| Admission 数据 | `admission-bootstrap-production-data.sh` 可重复执行并准备 BUAA 和目标 QQ 群策略 |
| Koishi/NapCat | 独立节点部署，插件包来自本地当前代码构建产物，不在生产 `node_modules` 手改源码 |

## 生产环境核心变量

```env
WEB_PUBLIC_URL=https://stuhelper.com
ADMIN_PUBLIC_URL=https://stuhelper.com/admin/
ADMISSION_PUBLIC_BASE_URL=https://join.stuhelper.com
WEB_VITE_WEB_URL=https://stuhelper.com
WEB_VITE_SSO_URL=https://sso.stuhelper.com

SSO_PUBLIC_SMOKE_ENABLED=true

CASDOOR_ISSUER=https://sso.stuhelper.com
CASDOOR_PUBLIC_AUTH_BASE_URL=https://sso.stuhelper.com
CASDOOR_REDIRECT_URI=https://stuhelper.com/api/v1/auth/callback
CASDOOR_ADMIN_REDIRECT_URI=https://stuhelper.com/api/v1/auth/callback
CASDOOR_UNIAPP_REDIRECT_URI=https://stuhelper.com/api/v1/auth/callback

CORS_ORIGINS=https://stuhelper.com,https://join.stuhelper.com,https://sso.stuhelper.com
TOKEN_COOKIE_SECURE=true
TOKEN_COOKIE_DOMAIN=.stuhelper.com

ADMISSION_PRODUCTION_READINESS_ENABLED=true
ADMISSION_READINESS_REQUIRED_PLATFORM=qq
ADMISSION_READINESS_REQUIRED_GUILD_IDS=178037297
ADMISSION_READINESS_REQUIRED_SCHOOL_CODES=4111010006
ADMISSION_READINESS_REQUIRED_SCHOOL_IDS=
ADMISSION_READINESS_REQUIRED_BOT_CREDENTIAL_NAME=koishi-runtime
ADMISSION_READINESS_REQUIRED_BOT_CREDENTIAL_AUDIENCE=/api/v1/bot/*
ADMISSION_READINESS_REQUIRED_BOT_CREDENTIAL_SCOPES=bot.qq_binding.consume,bot.qq_verification.read,bot.admission.session,bot.admission.event,bot.admission.review,bot.admission.forward,bot.member_blacklist.read,bot.member_blacklist.manage
ADMISSION_PUBLIC_SMOKE_ENABLED=true
ADMISSION_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS=false
ADMISSION_PUBLIC_SMOKE_CURL_INSECURE=false
PUBLIC_WEB_AUTH_BROWSER_SMOKE_ENABLED=true
PUBLIC_WEB_AUTH_BROWSER_SMOKE_ALLOW_LOCAL_TARGETS=false

STUHELPER_FRESHMAN_MATERIAL_HOSTS=stuhelper.com,join.stuhelper.com
```

Koishi 独立节点必须注入：

```env
STUHELPER_PLATFORM_BASE_URL=https://stuhelper.com
STUHELPER_PLATFORM_SERVICE_TOKEN=<redacted>
STUHELPER_FRESHMAN_MATERIAL_HOSTS=stuhelper.com,join.stuhelper.com
```

`STUHELPER_PLATFORM_SERVICE_TOKEN` 的值必须对应后端 `BOT_SERVICE_TOKEN` bootstrap 出来的 bot service credential。不要把真实值写入 runbook、脚本或 git。

## Nginx 入口契约

主站入口：

```text
stuhelper.com /api/*      -> http://127.0.0.1:18080
stuhelper.com /health/*   -> http://127.0.0.1:18080
stuhelper.com /admin/*    -> http://127.0.0.1:18001
stuhelper.com /verify     -> 404
stuhelper.com /verify/*   -> 404
stuhelper.com /*          -> http://127.0.0.1:18000
```

Join 入口：

```text
join.stuhelper.com /verify/<token>?qq=<qq> -> http://127.0.0.1:18000
join.stuhelper.com /verify                 -> 404
join.stuhelper.com /api/*                  -> http://127.0.0.1:18080
join.stuhelper.com /health/*               -> http://127.0.0.1:18080
```

SSO 入口：

```text
sso.stuhelper.com /.well-known/openid-configuration -> Casdoor
sso.stuhelper.com /.well-known/jwks                 -> Casdoor
sso.stuhelper.com /login/*                          -> Casdoor
sso.stuhelper.com /api/*                            -> Casdoor
```

执行审计：

```bash
NGINX_PUBLIC_INGRESS_PROFILE=stuhelper ./infra/ops/nginx-public-ingress-preflight.sh
NGINX_PUBLIC_INGRESS_PROFILE=sso ./infra/ops/nginx-public-ingress-preflight.sh
```

应用或恢复宝塔 vhost 模板：

```bash
# 默认只 dry-run，不写文件
./infra/ops/apply-baota-nginx-templates.sh --profile all

# 主站和 SSO 同机时可一次应用；脚本会先备份目标文件，再 nginx -t，最后按需 reload
sudo ./infra/ops/apply-baota-nginx-templates.sh --profile all --apply --reload --preflight

# 如果宝塔面板重写了 sso.stuhelper.com.conf，导致 discovery 变成 404，优先只恢复 SSO vhost
sudo ./infra/ops/apply-baota-nginx-templates.sh --profile sso --apply --reload --preflight
./infra/ops/sso-public-smoke.sh
```

注意：`apply-baota-nginx-templates.sh` 是仓库事实来源的一部分，生产不应直接手改 vhost 后停留在漂移状态。`--profile sso` 会同时安装 `infra/nginx/baota-casdoor-sso.conf` 和 `infra/nginx/baota-casdoor-sso-well-known-extension.conf`。后者目标路径是宝塔扩展目录 `/www/server/panel/vhost/nginx/extension/sso.stuhelper.com/stuhelper-sso-well-known.conf`，用于在宝塔重写主 vhost 但保留 extension include 时继续让 OIDC discovery/JWKS 走 Casdoor。如果主站和 SSO 不在同一台机器，只在对应机器执行对应 profile；不要把另一台机器的 vhost 目标路径作为临时手改。宝塔面板保存站点配置后可能重写 vhost。若 `sso-public-smoke.sh` 报 discovery 404，先用上面的 `--profile sso` 恢复，再审计 `NGINX_PUBLIC_INGRESS_PROFILE=sso ./infra/ops/nginx-public-ingress-preflight.sh`。

## Admission 生产数据

幂等准备：

```bash
./infra/ops/admission-bootstrap-production-data.sh
./infra/ops/admission-production-readiness.sh
```

验收条件：学校目录中存在 `code=4111010006` 的北京航空航天大学；管理后台学校配置页以 `schools` 目录为基表展示所有已录入学校，缺少 `school_configs` 的学校按默认停用配置展示，只有 `school_configs.enabled=true` 才进入学生认证和 admission 白名单。当前 admission 白名单只开放北航，对外、前端表单和运维检查使用学校代码 `4111010006`，不得再把旧五位学校 ID 作为业务事实或配置入口。公开学生认证、admission 邮箱 OTP、新生材料申请和学校 SSO 路径都应以 `schoolCode` 为主识别字段。`manual_form_fields.admission.emailDomains` 只有 `buaa.edu.cn`，且 `emailIdentityPolicy.type=academic_student_email`。`group_admission_policies` 至少包含 `platform=qq, guild_id=178037297, forward_raw_material_to_qq=false`，除非对象存储公开材料下载链路已完成并单独验收。手机拍照接力桌面端优先使用 `/api/v1/admission/freshman/camera-handoffs/{id}/events` SSE 获取实时状态，失败时才回退短轮询；上传后 continuation 必须锁定另一端，防止重复提交。`bot_service_credentials` 中存在 `koishi-runtime`，未吊销、未过期，audience 包含 `/api/v1/bot/*`，scopes 覆盖 QQ 绑定、admission session/event/review/forward 和 member blacklist。

北航老生学号邮箱 OTP 的生产推荐来源是外部只读 Oracle 学籍源。生产 secret env 中启用 `EXTERNAL_STUDENT_SOURCE_ENABLED=true`，并配置 `EXTERNAL_STUDENT_SOURCE_PROVIDER=oracle`、`EXTERNAL_STUDENT_SOURCE_SCHOOL_CODE=4111010006`、Oracle host/service/user/password/schema/table/column 等变量；真实密码只写 secret env 或 secret backend，不写入仓库。该源只读取 `USR_JWBIZ.T_XS_JBXX` 的 `XH` 和 `XM`，按学号查询并用于“学号 + 姓名”即时匹配。`./infra/ops/admission-production-readiness.sh` 会检查外部源配置完整性，并在 summary 中标记 `buaa_student_source=external_oracle`。真实连通验收还必须运行 `./infra/ops/external-student-source-smoke.sh`，留档 `infra/generated/external-student-source-smoke.json`；该 smoke 通过后表示 Oracle 可连接、配置表可读取且存在至少一条 `XH/XM` 非空记录。需要抽样校验时，在 secret env 中设置 `EXTERNAL_STUDENT_SOURCE_SMOKE_REQUIRE_SAMPLE=true`、`EXTERNAL_STUDENT_SOURCE_SMOKE_STUDENT_ID` 和可选 `EXTERNAL_STUDENT_SOURCE_SMOKE_EXPECTED_NAME`，evidence 只记录学号哈希前缀和匹配布尔值，不记录原始学号、姓名或密码。

如果暂时没有可用 Oracle 外部源，才使用本地 fallback 表 `academic.buaa_students`。拿到真实 TSV 后，先用 `BUAA_ACADEMIC_VALIDATE_ONLY=true BUAA_ACADEMIC_STUDENTS_TSV=/path/to/buaa-students.tsv ./infra/ops/import-buaa-academic-students.sh` 做离线校验，再用 `BUAA_ACADEMIC_STUDENTS_TSV=/path/to/buaa-students.tsv ./infra/ops/import-buaa-academic-students.sh` 导入。TSV 至少包含 `xh` 和 `xm` 列，也接受 `学号` 和 `姓名` 列名；可选列包括 `sfzjlxdm`、`sfzjh_hash`、`yxdm`、`zydm`、`bjdm`、`xznj`、`rxnj`、`pyccdm`、`xslbdm`、`sjh`、`dzxx`、`xjztdm`、`sfzx`、`sfzj`。脚本只做幂等 upsert，不清空旧数据，不打印学生明细。

## Koishi Admission MVP 配置

生产 `stuhelper-group-guard` 的 admission 配置必须限制职责边界：

```yaml
stuhelper-group-guard:admission:
  platform:
    baseUrl: ${{ env.STUHELPER_PLATFORM_BASE_URL }}
    serviceToken: ${{ env.STUHELPER_PLATFORM_SERVICE_TOKEN }}
  guard:
    targetGroups:
      - '178037297'
  commands:
    enabled: false
  admissionCommands:
    enabled: true
    minAuthority: 4
  moderation:
    enabled: false
  freshmanForward:
    enabled: false
```

约束：生产 NapCat / OneBot runtime platform 可以是 `onebot`；后端 admission policy/session/action 的 subject platform 仍是 `qq`。禁言、踢人、发消息使用当前 Koishi runtime bot，不切换适配器。`student-query.enableGroupVerify` 不因 admission 上线被整体关闭；冲突应通过新插件 `commands.enabled=false`、`moderation.enabled=false` 和旧插件自己的目标群范围处理。`admissionCommands.enabled=true` 只保留入群认证管理员命令，用于“查询入群认证 / 重发认证链接 / 重新生成认证链接”。当前 MVP 默认关闭 `freshmanForward.enabled`，避免未启用材料转发时每分钟请求 `/api/v1/bot/admission/freshman/applications/pending-forward` 并报错。

部署包必须从本地当前代码构建：

```bash
./infra/ops/package-koishi-stuhelper-packages.sh
```

若宝塔 Compose 环境只能手工覆盖包，也必须记录包 sha256、覆盖路径和重启步骤；源码修复必须回写到仓库。

## 发布流程

1. 在本地确认工作树中要发布的代码脚本模板和 runbook 已完整。
2. 生成并填写生产 env，不提交真实 secret：

   ```bash
   ENV_FILE=.env.prod.shared \
   SECRETS_ENV_FILE=.env.prod.secrets \
   GENERATED_ENV_FILE=.env.prod.generated \
   GENERATED_SECRET_ENV_FILE=.env.prod.generated.secrets \
   ./infra/ops/init-prod-env.sh
   ```

3. 本地执行契约和构建验证：

   ```bash
   make check-admission-mvp
   make check-infra-contracts
   cd server && make test && make build
   cd clients && pnpm type-check:all && pnpm build:web && pnpm build:admin
   cd bots/koishi && corepack yarn test && corepack yarn build
   ```

4. 远端发布：

   ```bash
   ./infra/ops/remote-prod-deploy.sh
   ```

   或在生产机已加载 `.deploy/remote.env` 后：

   ```bash
   make prod-deploy
   ```

5. 发布后执行 smoke 和日志确认。

## 发布后 smoke

公网入口：

```bash
curl -fsSI https://stuhelper.com/
curl -fsS https://stuhelper.com/health/live
curl -fsS https://stuhelper.com/health/ready
curl -fsSI https://stuhelper.com/admin/
curl -fsS https://sso.stuhelper.com/.well-known/openid-configuration | head
curl -fsSI 'https://join.stuhelper.com/verify/__manual_probe__?qq=10000'
curl -fsSI https://join.stuhelper.com/verify                  # 应 404
curl -fsSI 'https://stuhelper.com/verify/__manual_probe__?qq=10000' # 应 404
```

`sso.stuhelper.com` 的 OIDC discovery `issuer`、authorize、token 和 JWKS endpoint 必须全部是 `https://sso.stuhelper.com`。如果独立 Casdoor 宝塔 Compose 的 `conf/app.conf` 已经写入 `origin = https://sso.stuhelper.com`，但 discovery 仍返回 `http://sso.stuhelper.com`，不要只修改文件后结束；重启该 Compose 项目的 `casdoor` 容器，再运行 `./infra/ops/sso-public-smoke.sh` 留档。

如果发布时替换了宝塔 `source/` 目录或重建了基础服务容器，必须先运行 `./infra/ops/ensure-baota-runtime-permissions.sh --apply` 归一化 bind mount 权限，再重建 Postgres、Redis、app、frontend、admin。否则 `postgres:*-alpine` 可能读不到 `source/infra/generated/postgres/server.key`，Redis 可能读不到 `source/infra/generated/redis/users.acl`。如果同机还有独立 Casdoor 宝塔 Compose，该脚本也会修复 Casdoor `conf/app.conf` 和 `logs/` 的 UID 1000 权限，避免 SSO 502 或 Casdoor 重启循环。该步骤只改权限和 owner，不输出 secret 内容。

`sso-public-smoke.sh` 的 evidence 会记录每个公网请求的 `remoteIP`，生产模式会拒绝本机、私网、链路本地和保留网段解析结果。如果运维机 `/etc/hosts`、代理或诊断用 `SSO_PUBLIC_SMOKE_RESOLVE_IP` 把 `sso.stuhelper.com` 指向非公网地址，本 smoke 必须失败；只有本地契约测试或明确的本地验证才允许设置 `SSO_PUBLIC_SMOKE_ALLOW_LOCAL_TARGETS=true`。

自动 smoke：

```bash
./infra/ops/sso-public-smoke.sh
./infra/ops/admission-public-smoke.sh
./infra/ops/public-web-auth-browser-smoke.mjs
./infra/ops/admission-production-readiness.sh
./infra/ops/admission-reviewer-readiness.sh
./infra/ops/admission-mvp-production-evidence.sh
./infra/ops/tencent-ses-template-smoke.sh
RESEND_EMAIL_SMOKE_TO=<recipient-email> ./infra/ops/resend-email-channel-smoke.sh
./infra/ops/smoke-check.sh
./infra/ops/open-platform-production-evidence.sh
./infra/ops/observability-smoke-check.sh
```

学校邮箱 OTP 邮件使用多提供商驱动。生产配置应保持 `EMAIL_DRIVER=multi`、`EMAIL_FROM=noreply@notify.stuhelper.com`、`EMAIL_FROM_NAME=StuHelper 系统邮件`、`EMAIL_STUDENT_VERIFICATION_SUBJECT=学生认证验证码`、`EMAIL_TENCENT_TEMPLATE_ID=49779`、`EMAIL_RESEND_ENDPOINT=https://api.resend.com/emails`；`EMAIL_TENCENT_SECRET_ID`、`EMAIL_TENCENT_SECRET_KEY` 与 `EMAIL_RESEND_API_KEY` 只能放在 `.env.prod.secrets.local` 或部署 secret store。默认发送策略来自 `system_configs.email.delivery_policy`：腾讯云 SES `priority=10` 先发，Resend `priority=20` 作为故障兜底；如需负载均衡，把策略 `mode` 改为 `weighted` 并把多个 provider 设为同一优先级。`tencent-ses-template-smoke.sh` 不发送邮件，只调用 `GetEmailTemplate` 确认腾讯云凭据、地域、模板 ID 和审核状态，证据写入 `infra/generated/tencent-ses-template-smoke.json`。`resend-email-channel-smoke.sh` 会真实发送一封测试邮件，必须显式传入 `RESEND_EMAIL_SMOKE_TO`；它复用 `infra/email-templates/tencent-ses/stuhelper-school-email-otp.html` 和 `.txt` 渲染 Resend `html`/`text` 字段，证据只记录收件域名、收件地址哈希前缀和 Resend email ID，不输出 API key 或完整收件地址。

`sso-public-smoke.sh` 不只检查 discovery/JWKS/authorize 路由，也会读取公开 Casdoor application 元数据并断言 `admin/stuhelper-web` 仍为 `organization=stuhelper`、`enableSignUp=true`、`Password` 登录方式为 `All`、`Face ID` 为 `None`，且注册项包含必填的 `Password` 与 `Confirm password`。如果这里失败，先运行仓库内 `bootstrap-platform.sh prod` 修复 Casdoor 配置漂移，不要在 Casdoor 控制台手工改完就结束。

`admission-public-smoke.sh` 不只检查 `join.stuhelper.com/verify/<token>?qq=<qq>` 和旧入口 404，还会从 join 域向 `/api/v1/metrics/vitals`、`/api/v1/metrics/frontend-errors` 发送同源 beacon，要求返回 204，避免真实页面加载时 F12 出现红色 metrics 请求却被上线 smoke 漏掉。脚本还会无登录探测 `/api/v1/admission/freshman/camera-handoffs/<probe>/events`，要求走到后端返回 401 且 `X-Accel-Buffering: no`，防止手机拍照接力 SSE 被 Nginx 误转给 SPA 或被缓冲。

`public-web-auth-browser-smoke.mjs` 会用真实浏览器确认主站登录按钮进入 `sso.stuhelper.com/login/oauth/authorize` 后仍有账号密码登录和 `/signup/oauth/authorize` 注册入口，确认主站“注册账号”进入 `sso.stuhelper.com/signup/oauth/authorize` 的账号密码注册表单，并确认 `join.stuhelper.com/login?redirect=/verify/<token>?qq=<qq>` 的登录/注册按钮也分别进入 SSO 登录和注册授权页。这样 Casdoor 配置漂移成“只剩 Face ID”、注册按钮走错授权路径，或 join 入群链路的登录入口漂移时，公网浏览器 smoke 会直接失败。

`admission-mvp-production-evidence.sh` 是生产 admission MVP 的聚合证据入口。主站节点默认执行 SSO public smoke、admission public smoke、Web auth browser smoke 和 admission DB readiness，并写入 `infra/generated/admission-mvp-production-evidence.json`。如果 `EXTERNAL_STUDENT_SOURCE_ENABLED=true`，聚合入口会在 `ADMISSION_MVP_PRODUCTION_RUN_EXTERNAL_STUDENT_SOURCE_SMOKE=auto` 默认模式下强制运行 `external-student-source-smoke.sh`，确保外部学籍源不只“配置完整”，而是真的可连接、可读取；未启用外部源时，本地 fallback 表仍由 readiness 检查非空。Koishi 节点用 `ADMISSION_MVP_PRODUCTION_EVIDENCE_MODE=koishi` 执行同一入口时，会运行 `koishi-admission-production-evidence.sh`。普通聚合 evidence 允许真实 QQ E2E 被记录为 skipped，只能作为生产 smoke；最终上线验收必须在主站节点使用 `make prod-admission-mvp-final-evidence`，并在 Koishi 节点使用 `make prod-admission-mvp-final-koishi-evidence`。主站 final evidence 等价于显式设置 `ADMISSION_MVP_PRODUCTION_E2E_REQUIRED=true`、`ADMISSION_MVP_PRODUCTION_E2E_WAIT=true`、`ADMISSION_E2E_QQ_ID=<small-account-qq>`、`ADMISSION_MVP_PRODUCTION_E2E_EXPECTED_STAGE=bot-released` 和 `ADMISSION_MVP_PRODUCTION_E2E_MAX_SESSION_AGE_MINUTES=180`，让聚合 evidence 把最新真实 QQ 解除禁言回写也纳入验收。

如果主站生产机没有 Node/Playwright，不要跳过公网浏览器 smoke。应先在有 Playwright 的运维机或 CI 上运行 `PUBLIC_WEB_AUTH_BROWSER_SMOKE_EVIDENCE_FILE=infra/generated/public-web-auth-browser-smoke-evidence-current.json ./infra/ops/public-web-auth-browser-smoke.mjs`，把生成的脱敏 evidence 复制到主站源码目录的同一路径；聚合入口会默认读取该文件。需要使用其他文件名时，再显式设置 `ADMISSION_MVP_PRODUCTION_BROWSER_SMOKE_EVIDENCE_FILE=infra/generated/<evidence>.json`。聚合入口会校验该 evidence 新鲜、目标域名正确、十个浏览器检查全部通过，并确认 `/identity` 直接入口、join 登录/注册入口、camera permission 与 fake media capture 成功。

`admission-mvp-final-evidence-verify.sh` 只校验已采集的脱敏 evidence 文件，不访问生产，也不输出 secret。最终验收时，主站节点生成 `infra/generated/admission-mvp-final-evidence.json` 和 `infra/generated/admission-join-e2e-evidence.json`，Koishi 节点生成 `infra/generated/admission-mvp-final-koishi-evidence.json`；把三份文件放在同一份仓库工作目录后运行 `make prod-admission-mvp-final-verify`。该校验要求三份 evidence 都在默认 180 分钟新鲜度窗口内，主站和 Koishi 聚合 evidence 没有 failed/skipped，主站 evidence 必须包含真实 QQ `bot-released`，Koishi evidence 必须包含 Koishi admission production evidence 且不能夹带真实 QQ E2E placeholder，join E2E 子证据必须显示 token 已消费、QQ 已绑定、存在 active student verification credential、后端记录 bot release 和 cancelled marker，并包含通过的 `release requires active student verification credential` 检查。

`admission-reviewer-readiness.sh` 是只读检查，用 bot 的 `/api/v1/bot/admission/freshman/applications/{id}/view` 接口验证 QQ 管理员是否同时满足：所在群属于该申请的 `management_guild_ids`、管理员 QQ 已在 StuHelper 绑定、绑定用户拥有 `admission:freshman:review` 能力。它不调用 `/review`，不会批准或驳回申请。最终 E2E 卡在 `material_submitted` 时，应先用这条检查确认至少一个管理员可审核，而不是让用户反复试。可通过 `make prod-admission-reviewer-readiness` 调用，必填 `ADMISSION_REVIEWER_READINESS_APPLICATION_ID` 和 `ADMISSION_REVIEWER_READINESS_OPERATOR_QQ_IDS`，输出脱敏 evidence 到 `infra/generated/admission-reviewer-readiness.json`。

真实 QQ 小号入群 E2E 证据：

```bash
# 如果生产 DB 的 group_admission_sessions 为 0，说明还没有真实 QQ 入群事件触发 admission；
# 先让一个不在目标群内的小号实际申请/进入 178037297，再采集证据。

# 等待真实入群事件并自动落盘脱敏 evidence
ADMISSION_E2E_QQ_ID=<small-account-qq> \
ADMISSION_E2E_GUILD_ID=178037297 \
ADMISSION_E2E_EXPECTED_STAGE=join-created \
./infra/ops/admission-join-e2e-wait.sh

# 入群事件触发后，先验证 Koishi/后端已经创建 canonical admission session
ADMISSION_E2E_QQ_ID=<small-account-qq> \
ADMISSION_E2E_GUILD_ID=178037297 \
ADMISSION_E2E_EXPECTED_STAGE=join-created \
./infra/ops/admission-join-e2e-evidence.sh

# 等待用户完成 join.stuhelper.com 流程并自动落盘脱敏 evidence
ADMISSION_E2E_QQ_ID=<small-account-qq> \
ADMISSION_E2E_GUILD_ID=178037297 \
ADMISSION_E2E_EXPECTED_STAGE=flow-completed \
./infra/ops/admission-join-e2e-wait.sh

# 用户完成 join.stuhelper.com 登录、QQ 绑定和学生认证/新生材料流程后，验证业务流程完成
ADMISSION_E2E_QQ_ID=<small-account-qq> \
ADMISSION_E2E_GUILD_ID=178037297 \
ADMISSION_E2E_EXPECTED_STAGE=flow-completed \
./infra/ops/admission-join-e2e-evidence.sh

# 若 flow-completed 来自新生材料提交，先只读确认至少一个管理员 QQ 有审核权限
ADMISSION_REVIEWER_READINESS_APPLICATION_ID=<freshman-application-id> \
ADMISSION_REVIEWER_READINESS_OPERATOR_QQ_IDS=<operator-qq-ids> \
ADMISSION_REVIEWER_READINESS_GUILD_ID=178037297 \
make prod-admission-reviewer-readiness

# 等待 Koishi 执行解除禁言并把 release 成功回写后端
ADMISSION_E2E_QQ_ID=<small-account-qq> \
ADMISSION_E2E_GUILD_ID=178037297 \
ADMISSION_E2E_EXPECTED_STAGE=bot-released \
ADMISSION_E2E_WAIT_TIMEOUT_SECONDS=900 \
./infra/ops/admission-join-e2e-wait.sh
```

Koishi 生产证据脚本会检查配置、容器环境、bot API 探针和 admission 相关日志，只输出脱敏结果，不输出 token：

```bash
KOISHI_COMPOSE_DIR=/www/server/panel/data/compose/koishi-napcat \
KOISHI_ADMISSION_BOT_SELF_ID=<botSelfID> \
./infra/ops/koishi-admission-production-evidence.sh
```

预期：HTTP status 为 200。Koishi 日志无 `admission 401` / `unauthorized`。Koishi 日志无 `duplicate command names: 举报`。Koishi 日志无 `pending-forward` 每分钟 500 循环。`stuhelper-group-guard:admission` 已加载，目标群数量为 1。evidence 文件写入 `infra/generated/koishi-admission-production-evidence.json`，其中只记录 token 是否存在、bot API 状态码和 body 大小，不包含真实 token。默认日志窗口是最近 2 小时；如需审计更早的历史错误，可显式设置 `KOISHI_ADMISSION_LOG_SINCE=24h`。长期运行的 Koishi 容器可能已经没有启动加载日志，脚本默认不要求加载日志必须出现在当前窗口；刚重启后可设置 `KOISHI_ADMISSION_REQUIRE_LOAD_LOG=true` 强制检查。

## 上线完成定义

同时满足以下条件才算完成：

- 本地仓库包含所有代码修复、配置模板、幂等脚本和 runbook；不存在只在生产手工修改的最终状态。
- `sso.stuhelper.com` 是唯一公开登录认证系统和 OIDC issuer。
- `join.stuhelper.com/verify/<token>?qq=<qq>` 是唯一公开加群验证入口；`join.stuhelper.com/verify` 和主站 `/verify*` 返回 404。
- 主站 `/health/live`、`/health/ready` 通过。
- Admission public smoke、Web auth browser smoke、DB readiness 通过。
- Koishi admission 插件在 OneBot/NapCat runtime 下使用 `qq` subject platform 调后端 admission API。
- 生产日志没有 admission 401、duplicate command、pending-forward 500 循环报错。
- 真实 QQ 小号入群 E2E 的 `admission-join-e2e-evidence.sh` 在 `flow-completed` 阶段通过，且 `admission-join-e2e-wait.sh` 在 `bot-released` 阶段通过，证明 Koishi 已执行并上报解除禁言。
- 主站 `make prod-admission-mvp-final-evidence`、Koishi 节点 `make prod-admission-mvp-final-koishi-evidence` 均通过，并且主站聚合 evidence、join E2E 子证据、Koishi evidence 三份文件经 `make prod-admission-mvp-final-verify` 校验通过。
- ChatLuna API key 等非 admission 报错不混入本次 admission 修复范围。
