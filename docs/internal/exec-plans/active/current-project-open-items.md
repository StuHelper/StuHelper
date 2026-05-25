---
type: internal
audience: maintainers
status: current
authoritative-source: this file
last-verified: 2026-05-25
---

# 当前项目待办

本文件是执行计划的唯一活跃入口。历史计划、已完成阶段和已废弃方案不再用未勾选
checkbox 表示当前待办。

## 已确认活跃任务

Open Platform v1 baseline 已落地；下一步应补齐生产运营面和用户可控面。完整目标见 [`docs/design/open-platform-v1.md`](../../../design/open-platform-v1.md)。

| 任务 | 范围 | 当前状态 | 完成标准 |
|------|------|----------|----------|
| 生产公网身份入口现场验证 | `infra/ops/identity-public-smoke.sh` + 宝塔 Nginx / DNS / TLS / 外部 Casdoor | 公网身份 smoke、脱敏 evidence、失败诊断字段、部署前 public ingress preflight、实际 Nginx 配置审计脚本 `infra/ops/nginx-public-ingress-preflight.sh`、默认固定公网目标且不会被开发 `.env` localhost 覆盖的公网诊断脚本 `infra/ops/public-identity-ingress-diagnostic.sh` 和外部 SSO 宝塔反代模板 `infra/nginx/baota-casdoor-sso.conf` 已落地；公网 smoke 现默认拒绝 localhost / 127.0.0.1 / ::1 / host.docker.internal 目标，避免开发环境结果误写成公网 evidence，并覆盖普通 authorize 未登录跳转、`prompt=login&max_age=0` 重新认证跳转、token / introspect / revoke 路由级错误、GET/POST logout、POST logout URL query / JSON body 拒绝、GET/POST UserInfo 缺 bearer、UserInfo URL query / body token 来源拒绝、OIDC discovery、OAuth authorization server metadata、Identity JWKS 和外部 SSO discovery/JWKS，核对 `response_modes_supported=query`、token / introspect / revoke endpoint 的 `client_secret_basic` / `client_secret_post` metadata，并验证 token/UserInfo/introspection/revoke 敏感响应带 `Cache-Control: no-store` 与 `Pragma: no-cache` 以及 401 `WWW-Authenticate` challenge；配置专用 approved `IDENTITY_PUBLIC_SMOKE_CLIENT_ID` / `IDENTITY_PUBLIC_SMOKE_REDIRECT_URI` 后，还会验证注册客户端 `prompt=none` 回调 `login_required`、`state` 和 RFC 9207 `iss`；同时配置 `IDENTITY_PUBLIC_SMOKE_CLIENT_SECRET` 后，还会用 `client_credentials` 真实换取 app-only token、验证 introspection active、UserInfo 拒绝、公开 Open Platform resource access API 对未授权随机资源返回 `fga_denied`，或对预授权 smoke 资源返回 `allowed`、revoke 后 inactive，且 evidence 不记录 secret/token；2026-05-24 现场基线验证已通过：`WEB_PUBLIC_URL=https://stuhelper.com IDENTITY_ISSUER=https://id.stuhelper.com CASDOOR_ISSUER=https://sso.stuhelper.com IDENTITY_PUBLIC_SMOKE_RETRIES=3 IDENTITY_PUBLIC_SMOKE_SLEEP_SECONDS=1 IDENTITY_PUBLIC_SMOKE_EVIDENCE_FILE=/tmp/stuhelper-identity-public-smoke.json ./infra/ops/identity-public-smoke.sh` 返回 14 通过、0 失败，覆盖主站 health、Identity discovery、OAuth authorization server metadata、JWKS、authorize 未认证跳转、`prompt=login&max_age=0` 重新认证跳转、token / introspect / revoke 路由级错误、GET/POST logout、GET/POST UserInfo 缺 bearer 和外部 Casdoor discovery；同日诊断脚本确认 dns.google 公共 DNS 已将 `stuhelper.com` 与 `id.stuhelper.com` 解析到 `81.70.178.230`，`sso.stuhelper.com` 通过 `sso.stuhelper.com.eo.dnse2.com.` 返回公共 A/AAAA，TLS、主站 health、Identity discovery / OAuth AS metadata / JWKS 和 Casdoor discovery 均通过；本开发机 resolver 仍把三个域名解析到 `198.18.0.x` fake-IP，因此本机默认诊断仍因 `dns_non_public_address` 标记为未完全通过，这属于本机代理 DNS 现象，不再是公共 DNS/Nginx/OIDC 入口故障；同日 SSO 宝塔 Nginx 发现默认 `/.well-known` 静态 location 抢占 Casdoor discovery，已备份 `/www/server/panel/vhost/nginx/sso.stuhelper.com.conf.bak.20260524T1501` 并新增 `location ^~ /.well-known/` 反代到 `127.0.0.1:8087`，`nginx -t` 和 reload 通过，公网与 `--resolve` 源站 curl 均返回 200，修复后 `identity-public-smoke` 再次 14 通过、0 失败（`/tmp/stuhelper-identity-public-smoke-after-sso-fix.json`）；仓库侧已在 `remote-preflight.sh` / `prod-deploy.sh` 拉镜像前阻断主站/id 本机 Nginx 漂移和公网公共 DNS/TLS/OIDC 漂移 | 目标生产入口修复后，主站生产机 `NGINX_PUBLIC_INGRESS_PROFILE=stuhelper ./infra/ops/nginx-public-ingress-preflight.sh` 通过、SSO 机器 `NGINX_PUBLIC_INGRESS_PROFILE=sso ./infra/ops/nginx-public-ingress-preflight.sh` 通过，`PUBLIC_INGRESS_PREFLIGHT_ENABLED=true ./infra/ops/remote-preflight.sh` 通过，且 `IDENTITY_PUBLIC_SMOKE_ENABLED=true ./infra/ops/identity-public-smoke.sh` 生成 `passed=true` evidence；默认 evidence 必须包含 logout query/body 和 UserInfo query/body token 来源拒绝项；配置专用 smoke client 时 evidence 必须包含 `Identity prompt=none 未登录错误回调` 通过项；配置 smoke client secret 时 evidence 必须包含 `Identity client_credentials token 签发`、`Identity client_credentials introspection active`、`Identity client_credentials UserInfo 拒绝 app-only token`、`Open Platform resource access API 拒绝未授权随机资源` 或 `Open Platform resource access API 允许已授权资源`、`Identity client_credentials revoke 成功` 和 `Identity client_credentials revoke 后 inactive` 通过项 |
| 第三方 token 最小化准入探针 | `server/internal/platform/casdoor/` + `infra/ops/` | 审批 app 前已自动创建 / 更新 Casdoor third-party application，强制 `token_format=JWT-Custom`、`TokenFields=[]` 并写入 token probe audit；legacy Casdoor 导入会拒绝非 `JWT-Custom` token format 以及手机号、学生认证、学校、身份类型等业务 token fields；审批路径已支持 command-backed runtime code-flow 探针、失败关闭和 `open_platform_token_probe_evidence` 入库；backend 镜像已内置自动 code-flow runner；生产发布会 bootstrap 专用 Casdoor token probe smoke app，并在 app 启动前通过 `open-platform-production-evidence.sh` 实测 `businessClaims=[]` 与 `metadata.nonceVerified=true` 且留档脱敏 evidence；聚合脚本默认拒绝 localhost / 127.0.0.1 / ::1 / host.docker.internal Casdoor/OpenFGA 目标，避免开发环境结果误写成生产准入证据；2026-05-24 生产已观察通过：创建专用普通用户 `stuhelper_token_probe_smoke`，token-probe app 改为 Password 登录和 `JWT-Custom` + `TokenFields=[]` / `TokenAttributes=[]`，runtime code-flow 返回 `businessClaims=[]`、`tokenClaimTypes=[access_token,id_token]`、`metadata.nonceVerified=true`；聚合 evidence `passed=true` 已写入 `/www/server/panel/data/compose/stuhelper/source/infra/generated/open-platform-production-evidence.json` 和 `/www/server/panel/data/compose/stuhelper/source/infra/generated/open-platform-production-evidence-20260524T1502.json` | 每个第三方 app approved 前自动完成 OIDC code flow probe；Casdoor 应用不是 `JWT-Custom`、token 出现业务 claim 或 ID Token nonce 不匹配时拒绝 approved |
| Open Platform 资源授权运行时验证 | 真实 OpenFGA 联调 | OpenFGA model、后端 grant / revoke / list / check API、OpenAPI、shared/admin client、管理后台资源授权 UI、集成测试和 `infra/ops/openfga-resource-access-smoke.sh` 已落地；生产发布会在启动 app 前通过 `open-platform-production-evidence.sh` 自动运行资源授权 smoke，并把 grant/check/list/list-after-revoke/revoke 结果汇总入同一份 evidence；2026-05-24 已在目标生产 OpenFGA store/model 上观察通过：在 `stuhelper-prod-backend` Docker 网络内执行 `openfga-resource-smoke`，`OPENFGA_API_URL=http://openfga:8080`、store `01KSC9AF2S7Y92RQRR7E0WQWSG`、model `01KSC9AF4Z3KCSB3DS0WNC6VQP`，返回 `readAfterGrant=true`、`writeAfterGrant=true`、`listedReadGrant=true`、`readAfterRevoke=false`、`writeAfterRevoke=false`、`listedReadAfterRevoke=false`；生产证据已留存在 `/www/server/panel/data/compose/stuhelper/source/infra/generated/openfga-resource-access-smoke-production-20260524T1439.json`，同日聚合 evidence `/www/server/panel/data/compose/stuhelper/source/infra/generated/open-platform-production-evidence-20260524T1502.json` 也记录 `openfgaResourceAccessSmoke.passed=true` | 现场验证证明 scope consent 与具体资源 tuple 分层且 fail-closed |

生产现场补充证据（2026-05-24）：已在目标生产库创建专用 approved
`identity-public-smoke` 客户端和 `resource.read` approved scope，并把
`IDENTITY_PUBLIC_SMOKE_CLIENT_ID`、redirect URI、client credentials scope 和
client secret 写入宝塔 Compose 目录的生产 env 文件；生产 env 备份位于
`/www/server/panel/data/compose/stuhelper/backups/env-before-identity-public-smoke-20260524T074404Z.tar.gz`。
清理当前 SSH shell 中早前误 `source` 遗留的空 `IDENTITY_PUBLIC_SMOKE_*` 变量后，完整生产
`identity-public-smoke.sh` 已通过 24 项、0 失败，并留档到
`/www/server/panel/data/compose/stuhelper/source/infra/generated/identity-public-smoke-production-full-20260524T074535Z.json`；
该 evidence 覆盖注册客户端 `prompt=none` 错误回调、混用 client authentication 拒绝、
`client_credentials` token 签发、introspection active、UserInfo 拒绝 app-only token、Open Platform
resource access API 拒绝未授权随机资源、revoke 成功和 revoke 后 inactive。

本地验证补充（2026-05-24）：按 OpenAPI-first 链路重新执行 `make generate`，同步
`server/api/openapi.bundled.yaml`、`server/internal/api/gen/server.gen.go` 和
`clients/shared/src/types/api.gen.ts`；随后再次执行 `make generate` 并比对三份生成文件
SHA-256，确认生成幂等。已通过 `go test -count=1 ./...`、`make check-infra-contracts`、
`make check-docs`、`make lint-spec`、`pnpm test:all`、`pnpm --dir admin test`、
`pnpm type-check:all`、`pnpm lint:all`、`make build`、`pnpm build:web`、`pnpm build:admin` 和
`pnpm build:uni:h5`。E2E 方面，默认 `127.0.0.1:3000` 已被本机既有 Vite 进程占用，
因此使用 `PLAYWRIGHT_WEB_PORT=3300 make e2e-web` 跑通 Web 47 项，并使用
`make e2e-admin` 跑通 Admin 26 项。

本地验证补充（2026-05-25）：新增 Web Playwright 覆盖 Open Platform consent/profile completion、开发者门户应用创建、展示资料更新、scope 新增 / 重提、pending app / scope / redirect URI 撤回、client secret 轮换、redirect URI 变更、开发者审计展示、用户授权应用 scope / 全应用撤销、实名与学生认证、手机号绑定、QQ 绑定码、学籍信息、OAuth callback、admission 入群邮箱 OTP、课程说明页、评课聚合 feed 排序/院系侧栏、教师主页热门/搜索、自己评价编辑/删除、课程收藏切换、评课详情点赞 / 回复 / 删除回复 / 加载更多、通知全部已读、单条通知标记已读并跳转和公开评价举报；新增 Admin Playwright 覆盖 dashboard、内容管理、用户系统、开放平台管理页、个人中心 tabs、已登录 404 fallback、教师管理新增/编辑/删除、敏感词新增/编辑/删除、评课审核动作、举报处置动作、实名审核通过/驳回、学生认证审核、新生材料审核、成员黑名单新增/解除、系统配置保存、学校 LDAP 配置保存、入群策略数值字段与管理群列表保存、Open Platform scope / redirect / app 审核、legacy Casdoor 应用导入、应用密钥轮换 / 暂停 / 恢复 / 吊销、资源授权 grant / revoke 和管理员授权撤销。新增覆盖发现教师主页仍显示通用教学中心标题，且热门教师接口失败会触发全页 ErrorBoundary，已修正为教师主页标题并在加载失败时显示空态。新增覆盖后再次通过 `PLAYWRIGHT_WEB_PORT=3300 make e2e-web`（47 项）、`make e2e-admin`（26 项）、`pnpm --dir clients run type-check:web`、`pnpm --dir clients/admin --filter @vben/web-ele typecheck`、相关单文件格式/静态检查、`pnpm --dir clients/admin lint` 和 `git diff --check`。

本地验证补充（2026-05-25）：Web / Admin Playwright E2E 已接入统一浏览器运行时门禁，
每个测试结束时检查未捕获 `pageerror`、关键静态资源加载失败和关键资源 HTTP 4xx/5xx；关键资源包含
`document`、`script`、`stylesheet`、`font` 和 `image`。该门禁发现 Admin 仍依赖
`https://unpkg.com/@vbenjs/static-source` 远程 logo，已改成本地/内联资源，PWA 图标也改为随
Admin 应用发布的本地 SVG。更新后已通过 `CI=1 PLAYWRIGHT_WEB_PORT=3406 make e2e-web`
（51 项）、`CI=1 make e2e-admin`（26 项）、`pnpm --dir clients run type-check:admin`、
`pnpm --dir clients run lint:admin`、`pnpm --dir clients/admin exec vitest run packages/@core/preferences/__tests__/config.test.ts`
和 `git diff --check`。同日用当前提交 `de5fd44f` 重建并启动本机生产等价栈
`make prod-parity-up`，镜像标签为 `prod-parity-de5fd44f`；API/Web/Admin readiness、基础业务
smoke、Identity public smoke（26 项）、OpenFGA resource access smoke、Web/Admin 生产镜像浏览器
smoke 和 observability smoke 均通过；随后将 prod-parity browser smoke 的关键资源检查同步收紧到
`document`、`script`、`stylesheet`、`font`、`image` 并复跑 `make prod-parity-smoke` 通过。浏览器 evidence 写入
`.run/prod-parity/browser-smoke-evidence.json`，截图写入 `.run/prod-parity/browser-smoke-screenshots/`。

本地验证补充（2026-05-25）：对照 Web / Admin 路由表与现有 Playwright 用例重新审计浏览器覆盖；
补齐 Web 用户中心快捷入口 `/user` 到 `/user/reviews` 的真实浏览器重定向，以及普通登录用户缺少
`review:create` capability 时访问 `/courses/reviews/post` 会被路由守卫导回首页且不渲染发布表单的
E2E 覆盖。新增覆盖后已通过单文件
`CI=1 PLAYWRIGHT_WEB_PORT=3408 pnpm exec playwright test tests/e2e/auth-flow.spec.ts`（8 项）和
全量 `CI=1 PLAYWRIGHT_WEB_PORT=3408 make e2e-web`（53 项）。

本地验证补充（2026-05-25）：Web / Admin Playwright fixture 和本机生产等价 browser smoke 的浏览器
运行时门禁继续收紧：除了 `requestfailed`，现在也会把 `document`、`script`、`stylesheet`、`font`、
`image` 关键资源的 HTTP 4xx/5xx 视为失败，避免关键资源实际返回错误页但网络层未失败时漏过。
新增门禁后已通过 `CI=1 PLAYWRIGHT_WEB_PORT=3409 make e2e-web`（53 项）、`CI=1 make e2e-admin`
（26 项）、`make prod-parity-browser-smoke`、`make prod-parity-smoke`、`make check-docs`、
`make check-infra-contracts`、`node --check infra/ops/prod-parity-browser-smoke.mjs`、
`bash infra/ops/tests/prod-parity-contract.sh` 和 `git diff --check`。

本地验证补充（2026-05-25）：Web / Admin Playwright fixture 的浏览器门禁继续从 API HTTP 5xx
收紧到未声明允许的 `fetch` / `xhr` HTTP 4xx/5xx，并同步把非网络状态类 `console.error`
纳入失败条件，防止页面断言未覆盖时漏掉客户端 API 错误或组件运行时报错。
允许清单仅保留被页面显式展示的业务状态：匿名 `auth/me` 401、Web 未绑定 QQ 404、未创建实名信息
404、入群链接冲突 / 过期 409 / 410、匿名读取课程收藏状态 401；Admin 仅允许匿名 `auth/me` 401。
收紧后已通过完整 `CI=1 PLAYWRIGHT_WEB_PORT=3410 make e2e`，覆盖 Web 53 项和 Admin 26 项。

本地验证补充（2026-05-25）：Web / Admin E2E 测试代码本身也纳入常用静态门禁。Web 新增
`tsconfig.e2e.json`，`@stuhelper/web` 的 `type-check` 现在同时执行 `vue-tsc` 和 E2E
`tsc --noEmit`，`lint` 现在覆盖 `src` 与 `tests/e2e`；Admin `@vben/web-ele` 的
`tsconfig.node.json` 覆盖 `playwright.config.ts` 与 `tests/e2e/**/*.ts`，`typecheck` 同时执行
`vue-tsc` 和 Node/Playwright `tsc --noEmit`。接入门禁时修复了 Web 摄像头 mock、发布评课 payload
断言和 Admin Open Platform E2E 的类型问题。已通过 `pnpm --dir clients type-check:all` 和
`pnpm --dir clients lint:all`，并复跑完整 `CI=1 PLAYWRIGHT_WEB_PORT=3410 make e2e`（Web 53 项、
Admin 26 项）。

本地验证补充（2026-05-25）：普通 Web Playwright E2E 从单桌面上下文扩展为
`desktop-chromium` 与 `mobile-chromium` 两个 project，使 Open Platform、认证、用户中心、课程 /
评课社区、实名 / 学生认证、通知和静态页面等 53 条交互用例同时覆盖桌面与移动视口。本轮扩展暴露了
移动端评课聚合页需要先打开“课程列表”抽屉才能操作院系侧栏，以及手机号绑定后立即切页会取消用户中心
后台请求的问题；测试已按真实交互路径打开抽屉，并补齐重定向后的用户中心 API mock 与网络空闲等待，
不放宽全局浏览器 / API 失败门禁。已通过 `CI=1 PLAYWRIGHT_WEB_PORT=3412 make e2e-web`（106 项）
和完整 `CI=1 PLAYWRIGHT_WEB_PORT=3413 make e2e`（Web 106 项、Admin 26 项）。

本地验证补充（2026-05-25）：Web 评课社区 E2E 继续补齐浏览交互的请求参数断言，覆盖评课聚合页
“最热”/“精选”排序传递 `sort=likes` / `sort=rating`、`page=1`、`pageSize=10`，院系侧栏展开课程列表传递
`departmentID=1`、`page=1`、`pageSize=100`，以及教师主页搜索传递 `q=王`、`sort=reviews`、
`pageSize=30`；用例继续在桌面和移动 project 下运行。新增覆盖后已通过单文件
`CI=1 PLAYWRIGHT_WEB_PORT=3421 pnpm --dir clients/web exec playwright test tests/e2e/course-community.spec.ts`
（6 项）、`pnpm --dir clients type-check:web`、`pnpm --dir clients lint:web` 和完整
`CI=1 PLAYWRIGHT_WEB_PORT=3422 make e2e-web`（106 项）。

本地验证补充（2026-05-25）：Web 高级搜索 E2E 继续补齐课程搜索与评价搜索双接口参数断言，
覆盖课程名搜索同时请求课程搜索 `q=数据结构`、`pageSize=50` 和评价搜索 `q=数据结构`、`pageSize=50`、
`sort=time`；新增“院系 + 教师 + 学期”组合搜索路径，断言课程列表查询传递 `departmentID=1`、
`pageSize=50`，评价搜索传递 `departmentID=1`、`teacherName=张教授`、`termID=2025-fall`、
`pageSize=50`、`sort=time`，并验证对应课程和评价结果渲染。新增覆盖后已通过单文件
`CI=1 PLAYWRIGHT_WEB_PORT=3423 pnpm --dir clients/web exec playwright test tests/e2e/journey-search.spec.ts`
（6 项）、`pnpm --dir clients type-check:web`、`pnpm --dir clients lint:web` 和完整
`CI=1 PLAYWRIGHT_WEB_PORT=3424 make e2e-web`（108 项）。

本地清理补充（2026-05-25）：审计 Web 课程列表页时确认当前页面只使用
`GET /api/v1/course/courses/grouped` 渲染院系分组，不再使用旧的服务端筛选查询构造器；已删除无生产引用的
`clients/web/src/modules/course/courseListQuery.ts` 和仅覆盖该废弃工具的单测文件。清理后 `rg` 确认无剩余引用，
并通过 `pnpm --dir clients test:web`（53 文件、230 项）、`pnpm --dir clients type-check:web` 和
`pnpm --dir clients lint:web`。

本地验证补充（2026-05-25）：Web 课程详情页“加载更多”E2E 继续从只检查页码扩展到完整请求参数断言，
验证初始评课列表和第二页请求都携带 `pageSize=20` 与 `sort=time`，分别请求 `page=1` 和 `page=2`，
并确认第二页评价真实追加渲染。新增覆盖后已通过单文件
`CI=1 PLAYWRIGHT_WEB_PORT=3425 pnpm --dir clients/web exec playwright test tests/e2e/review-actions.spec.ts`
（10 项）、`pnpm --dir clients type-check:web`、`pnpm --dir clients lint:web` 和完整
`CI=1 PLAYWRIGHT_WEB_PORT=3426 make e2e-web`（108 项）。

本地验证补充（2026-05-25）：Web Open Platform 开发者门户 E2E 继续补齐生产关键查询路径，
覆盖应用列表初始加载传递 `page=1`、`pageSize=10`、`status=all`，状态筛选传递 `status=pending`
并重置到第一页，分页传递 `page=2`，以及应用活动记录请求传递 `pageSize=10`；同时给状态筛选下拉补充
明确 `aria-label`，避免与“实名认证状态 / 学生认证状态”等权限复选框发生无障碍名称歧义。新增覆盖后已通过
单文件
`CI=1 PLAYWRIGHT_WEB_PORT=3429 pnpm --dir clients/web exec playwright test tests/e2e/open-platform-developer.spec.ts`
（6 项）、`pnpm --dir clients type-check:web`、`pnpm --dir clients lint:web` 和完整
`CI=1 PLAYWRIGHT_WEB_PORT=3430 make e2e-web`（110 项）。

本地验证补充（2026-05-25）：Web 用户中心授权应用 E2E 继续补齐 consent 活动链路，
覆盖授权活动列表请求传递 `pageSize=10`，并断言页面渲染 consent grant 活动的 app、scope、endpoint
和 result；撤销单个 `email.read` scope 后，测试验证应用仍保留、对应 scope 从应用授权清单移除，
授权活动刷新为 `open_platform.consent.revoked`，且展示被撤销的 scope。新增覆盖后已通过单文件
`CI=1 PLAYWRIGHT_WEB_PORT=3434 pnpm --dir clients/web exec playwright test tests/e2e/journey-user-center.spec.ts`
（16 项）、`pnpm --dir clients type-check:web`、`pnpm --dir clients lint:web` 和完整
`CI=1 PLAYWRIGHT_WEB_PORT=3435 make e2e-web`（110 项）。

本地验证补充（2026-05-25）：Web Open Platform consent E2E 继续补齐用户拒绝授权路径，
覆盖 `/consent?token=...` 载入 challenge 后点击“拒绝”，断言前端调用
`POST /api/v1/open-platform/consent/deny` 且 body 只包含 challenge token，并按后端返回的
`error=access_denied` 回调 URL 跳回第三方客户端。新增覆盖后已通过单文件
`CI=1 PLAYWRIGHT_WEB_PORT=3436 pnpm --dir clients/web exec playwright test tests/e2e/open-platform-consent.spec.ts`
（6 项）、`pnpm --dir clients type-check:web`、`pnpm --dir clients lint:web` 和完整
`CI=1 PLAYWRIGHT_WEB_PORT=3437 make e2e-web`（112 项）。

本地生产等价验证补充（2026-05-25）：按“先本机同构，再生产发布”的流程，用当前提交
`prod-parity-afde095e` 重新执行完整 `make prod-parity-up`。该入口在本机 Ubuntu 24.04 + Docker 中完成
共享 PostgreSQL 初始化、StuHelper / OpenFGA / Casdoor 独立数据库和账号创建、独立 Redis TLS/ACL 实例启动、
生产镜像构建、migration、OpenFGA/Casdoor bootstrap、API/Identity/OpenFGA/浏览器/观测 smoke，并最终输出
Web `http://127.0.0.1:28000`、Admin `http://127.0.0.1:28001/admin/`、Backend `http://127.0.0.1:28080`、
Grafana `http://127.0.0.1:23003`。其中 datastore evidence 显示 PostgreSQL 容器
`stuhelper-prod-parity-postgres` 内 `stuhelper` / `openfga` / `casdoor` 三库均已隔离，Redis 容器
`stuhelper-prod-parity-redis` 只在 StuHelper backend 网络中运行且未加入 `stuhelper-prod-parity-baota-net`；
浏览器 smoke evidence 记录 64 项检查通过，Identity evidence 记录 26 项检查通过。首次运行在
`proxy.golang.org` 依赖下载阶段遇到瞬时 `unexpected EOF`，复跑 `make prod-parity-smoke` 通过后，再次完整
`make prod-parity-up` 已成功收尾。

本地验证补充（2026-05-25）：Admin Playwright E2E 也从单浏览器上下文扩展为
`desktop-chromium` 与 `mobile-chromium` 两个 project，使管理后台核心壳、登录跳转、内容审核 /
举报处理、教师与敏感词 CRUD、用户系统配置、入群认证策略、Open Platform 应用审核 / 授权 / 同意撤销等
26 条管理用例同时覆盖桌面与移动视口。本轮扩展暴露了 profile 测试在移动布局中命中隐藏导航标题副本的问题；
断言已收敛到页面主体 `main` 内的认证用户信息和账号 tab，不放宽全局浏览器 / API 失败门禁。已通过
`CI=1 ADMIN_E2E_PORT=4176 make e2e-admin`（52 项）和完整
`CI=1 PLAYWRIGHT_WEB_PORT=3414 ADMIN_E2E_PORT=4177 make e2e`（Web 106 项、Admin 52 项）。

本地验证补充（2026-05-25）：Admin Open Platform 运营页 E2E 从静态渲染扩展到查询交互，
覆盖 token probe evidence 的应用 ID、审核人 ID、结果和 Client ID 筛选，以及 disclosure report 的统计窗口查询；
用例会断言 UI 输入实际进入 `GET /api/v1/admin/open-platform/token-probe-evidence` 的
`appID` / `reviewerUserID` / `result` / `clientID` / `page` / `pageSize` 查询参数，以及
`GET /api/v1/admin/open-platform/disclosure-report` 的 `windowHours` 查询参数，避免运营页表格能渲染但筛选条件未传递到后端。

本地验证补充（2026-05-25）：Koishi Console Playwright UI smoke 已在原有 `pageerror` 和
console error/warning tracker 基础上补齐关键资源门禁，把 `document`、`script`、`stylesheet`、
`font`、`image` 的 `requestfailed` 和 HTTP 4xx/5xx 视为失败，避免 NavRail / view anchor
断言通过但实际 chunk、样式、字体或图片资源损坏时漏过。新增门禁后已通过 `make e2e-koishi`
（13 项）和 `corepack yarn test:unit`（259 项）。

本地验证补充（2026-05-25）：Koishi Console Playwright UI smoke 继续从顶层 NavRail / ChatDock
扩展到配置治理二级工作区，覆盖“群配置 / 模板库 / 群绑定 / 命令策略”subnav 切换、URL hash 状态同步和
view-specific 编辑区渲染；同时通过真实 UI 保存 `e2e-template` guard template，验证 Console action API
能把模板 ID、名称、禁言时长、踢出阈值、提醒文案和豁免名单提交到后端并回显保存结果。新增覆盖后已通过
`make e2e-koishi`（15 项）和 `corepack yarn test:unit`（259 项）。

本地验证补充（2026-05-25）：Koishi Console 配置治理保存 E2E 继续补齐同一工作区剩余写路径；
在真实 Console UI 中先保存 `e2e-template` guard template，再切到“群绑定”保存 `onebot/1001`
到该模板的绑定并验证回显备注，最后切到“命令策略”保存 `report` 命令的 `authority=4` 和
`admin, moderator` 角色白名单并验证列表回显。扩展时发现 Element Plus select 的隐藏 input 会被
placeholder 截获点击，测试 helper 已改为点击 `.el-select__wrapper` 以匹配真实用户交互目标。
新增覆盖后已通过 `make e2e-koishi`（15 项）和 `corepack yarn test:unit`（259 项）。

本地机器人侧复验（2026-05-25）：在提交 `823b4acc` 上复跑 Koishi Console 与机器人核心测试。
`make e2e-koishi` 通过 15 项 Playwright UI smoke，覆盖 NavRail、Dashboard、Review、Identity、
Warns、Blacklist、Config、Roles、Settings、Logs、Subscriptions、System、ChatDock、配置治理 tabs，
以及真实 Console action 保存 guard template、群绑定和命令策略；`corepack yarn test:unit` 在
`bots/koishi` 工作目录通过 259 项、0 失败，覆盖 admission、绑定、群管 runtime modules、Console
scope、governance 写入、黑名单、举报、关键词、入群认证、反撤回、AI 超时等机器人侧单元路径。

本地验证补充（2026-05-25）：Koishi Console Playwright UI smoke 继续补齐全局搜索与实体上下文抽屉。
新增用例覆盖 `Control+K` 打开全局搜索、搜索 `logs` 跳转“日志检索”、数字实体搜索打开用户上下文抽屉、
空记录用户显示空态、抽屉内“警告记录”快捷跳转写入 `#warns?keyword=100000`，并把 ChatDock 测试收敛为
打开后关闭，避免共享浏览器会话残留浮层遮挡后续配置保存。该轮测试暴露了非 Mac 键盘快捷键只识别小写
`k`、Teleport 到 body 的搜索 / 抽屉 / 聊天浮层层级低于 Koishi 主界面、以及 isolated UI smoke
默认指向不存在的平台后端导致 Dashboard 显示 `TypeError: fetch failed` 的问题；已通过按键归一化、
提升 portal z-index、为 `scripts/ui-smoke.mjs` 提供可被真实环境变量覆盖的本地平台 stub 修复，并在
Dashboard E2E 中断言统计卡片出现且无“加载失败”。修复后已通过 `make e2e-koishi`（17 项）、
`corepack yarn test:unit`（259 项）、`node --check bots/koishi/scripts/ui-smoke.mjs` 和 `git diff --check`。

本地全量复验（2026-05-25）：在提交前代码状态 `44b1e0dd` 上复跑统一客户端 E2E 门禁，
`CI=1 PLAYWRIGHT_WEB_PORT=3420 ADMIN_E2E_PORT=4190 UNIAPPX_E2E_PORT=3134 make e2e`
通过 Web 106 项、Admin 56 项和 UniAppX 28 项；随后复跑 `make e2e-koishi`，通过 Koishi
Console UI 15 项。该轮复验覆盖 Web / Admin / UniAppX 的桌面与移动 Playwright project，以及
Koishi Console Chromium UI smoke；浏览器 `pageerror`、console error、关键静态资源和 API
4xx/5xx 门禁保持开启，未通过兜底或放宽断言来获得通过结果。

本地验证补充（2026-05-25）：Admin 操作日志页 E2E 从只断言表格渲染扩展到分页交互，
通过真实点击 Element Plus pagination 下一页按钮验证 `GET /api/v1/course/review/admin/logs`
携带 `page=2` 和当前 `pageSize=20` 查询参数，避免操作日志页面能展示第一页但分页请求未传递到后端。
新增覆盖后已通过单文件
`CI=1 ADMIN_E2E_PORT=4192 pnpm --dir clients/admin --filter @vben/web-ele exec playwright test tests/e2e/admin-surface.spec.ts`
（14 项）、完整 `CI=1 ADMIN_E2E_PORT=4193 make e2e-admin`（58 项）、
`pnpm --dir clients type-check:admin` 和 `pnpm --dir clients lint:admin`。

本地验证补充（2026-05-25）：Admin 内容管理页 E2E 继续补齐筛选参数断言，覆盖评课管理状态筛选
`status=pending_review`、举报管理状态筛选 `status=resolved`，以及敏感词管理分类 / 级别筛选
`category=comment`、`level=review`；用例等待对应 filtered response 完成后再切换页面，避免测试自身中止
仍在途 API 请求而触发 fail-closed 门禁。新增覆盖后已通过单文件
`CI=1 ADMIN_E2E_PORT=4195 pnpm --dir clients/admin --filter @vben/web-ele exec playwright test tests/e2e/admin-surface.spec.ts`
（16 项）、`pnpm --dir clients type-check:admin`、`pnpm --dir clients lint:admin` 和按顺序执行的完整
`CI=1 ADMIN_E2E_PORT=4197 make e2e-admin`（60 项）。

本地验证补充（2026-05-25）：Admin 用户系统页 E2E 继续补齐筛选参数断言，覆盖实名审核状态筛选
`status=verified`、学生认证审核 `schoolID=1001` + `status=verified`、新生材料审核
`status=rejected`，以及成员黑名单 `subjectID` / `guildID` / `platform` / `scopeType` / `source` /
`status` 组合筛选；用例复用 fail-closed Admin API mock，并等待对应 filtered response 完成后断言请求参数。
新增覆盖后已通过单文件
`CI=1 ADMIN_E2E_PORT=4199 pnpm --dir clients/admin --filter @vben/web-ele exec playwright test tests/e2e/admin-surface.spec.ts`
（18 项）、`pnpm --dir clients type-check:admin`、`pnpm --dir clients lint:admin`、完整
`CI=1 ADMIN_E2E_PORT=4200 make e2e-admin`（62 项）、`make check-docs` 和 `git diff --check`。

本地验证补充（2026-05-25）：Admin Open Platform 运营页 E2E 继续补齐审计事件和用户授权查询参数断言，
覆盖审计事件 `appID=42`、`userID=12`、`eventType=open_platform.consent.granted`、
`scope=email.read`、`page=1`、`pageSize=20`，以及用户授权列表 `appID=42`、`userID=12`、
`page=1`、`pageSize=20`；用例等待对应 filtered response 完成后断言请求参数，避免页面只渲染空表格但筛选条件未传递。
新增覆盖后已通过单文件
`CI=1 ADMIN_E2E_PORT=4202 pnpm --dir clients/admin --filter @vben/web-ele exec playwright test tests/e2e/admin-surface.spec.ts`
（20 项）、`pnpm --dir clients type-check:admin`、`pnpm --dir clients lint:admin` 和完整
`CI=1 ADMIN_E2E_PORT=4203 make e2e-admin`（64 项）。

本地验证补充（2026-05-25）：Admin 内容 / Open Platform 审核页 E2E 继续补齐剩余筛选参数断言，
覆盖教师管理姓名 / 院系筛选传递为 `search=李教授`、`departmentID=1`、`page=1`、`pageSize=20`，
以及 Open Platform 应用审核页状态切换传递 `status=approved`、`page=1`、`pageSize=20`。
新增覆盖后已通过单文件
`CI=1 ADMIN_E2E_PORT=4204 pnpm --dir clients/admin --filter @vben/web-ele exec playwright test tests/e2e/admin-surface.spec.ts`
（22 项）、`pnpm --dir clients type-check:admin`、`pnpm --dir clients lint:admin` 和完整
`CI=1 ADMIN_E2E_PORT=4205 make e2e-admin`（66 项）。

本地验证补充（2026-05-25）：本机生产等价 browser smoke 从浅层首页检查扩展为公开 Web 路由
矩阵，覆盖首页、登录页、认证回调错误态、入群认证链接、关于、隐私、条款、课程入口、课程列表、课程说明、评课聚合、搜索、教师主页、
写评课、用户中心各 tab、实名 / 学生认证、手机 / QQ 绑定、学籍信息、通知、开发者应用、Open Platform
授权与资料补全保护跳转、404 页面和 Admin 登录跳转；保护入口会同时验证落到登录页且保留原始 `redirect`。
同一批检查现在会分别用桌面 `1365x900` 和移动 `390x844` 视口运行，evidence 记录视口信息，截图文件名带视口后缀。
同时把页面触发的未声明允许 `fetch` / `xhr` HTTP 4xx/5xx 视为失败，以便发现页面壳能渲染但后端集成接口异常的问题。
每个检查使用独立 browser context，避免 Web 登录页、保护路由和 Admin 登录跳转之间共享 cookie / localStorage
造成误判；evidence 会记录每项检查命中的 `matchedText`，便于确认 smoke 不是只拿到空页面。
同一 browser smoke 也会把非网络状态类浏览器 `console.error` 纳入失败条件，避免组件运行时报错
只落在控制台而页面仍返回 200 时漏过；浏览器自动打印的 HTTP 状态行会单独记录为 ignored evidence，
其失败语义继续由关键资源 HTTP 4xx/5xx 与页面 `fetch` / `xhr` HTTP 4xx/5xx 门禁负责。

本地验证补充（2026-05-25）：本机生产等价 browser smoke 不再只验证空库页面壳。新增
`prod-parity-smoke-data.sh`，在本机 prod-parity PostgreSQL 中幂等写入专用院系、教师、课程、
已发布评课、回复和入群认证会话，刷新课程 / 教师评分统计及 `mv_teacher_public_stats`，并清理 prod-parity Redis
中的课程 / 评课缓存；脚本拒绝非 `prod-parity` PostgreSQL / Redis 容器，避免误用到生产。Browser smoke
新增 `requiredTexts` 断言，并把课程列表、课程详情 `/courses/900001`、课程评课详情
`/courses/900001/reviews`、评课聚合、教师主页和教师详情 `/teachers/900001` 都纳入桌面与移动视口检查。
本轮已通过真实 API 验证课程、教师、匿名评课预览数据和入群认证链接可见，并通过 `./infra/ops/prod-parity-browser-smoke.sh`
（64 项，桌面 / 移动各 32 项）；smoke data evidence 写入 `.run/prod-parity/smoke-data-evidence.json`，
browser evidence 写入 `.run/prod-parity/browser-smoke-evidence.json`。

本地生产等价补充（2026-05-25）：新增 `prod-parity-datastore-smoke.sh`，并接入
`make prod-parity-smoke`。该门禁在真实容器上验证共享 PostgreSQL 中 StuHelper / OpenFGA /
本地 SSO Casdoor 使用独立数据库和独立登录账号，跨库连接会被拒绝；同时验证 Redis 是
StuHelper Compose 内的独立 TLS/ACL 实例，使用自己的 `/data` volume，且没有加入外部 datastore
网络。脱敏 evidence 写入 `.run/prod-parity/datastore-smoke-evidence.json`；本轮已通过
`make prod-parity-datastore-smoke`、`make prod-parity-smoke`、`bash infra/ops/tests/prod-parity-contract.sh`、
`make check-docs`、`make check-infra-contracts` 和 `git diff --check`。

本地生产等价复验（2026-05-25）：使用当前 HEAD `babfe8af` 执行 `make prod-parity-up`，
重建并滚动本机生产等价镜像 `stuhelper/backend:prod-parity-babfe8af`、
`stuhelper/frontend:prod-parity-babfe8af` 和 `stuhelper/admin:prod-parity-babfe8af`，
三个应用容器均为 healthy。完整启动链路已通过共享 PostgreSQL / 独立 Redis datastore smoke
（22 项）、prod-parity smoke data seed、基础业务 smoke（17 通过、1 个 Grafana URL 配置跳过）、
Identity public smoke（26 项）、OpenFGA resource access smoke、Web/Admin 生产镜像 browser smoke
（64 项，桌面 / 移动各 32 项）和 observability smoke（Prometheus、Grafana、Loki、Tempo、
Alertmanager、Alloy 共 6 项）。脱敏 evidence 保留在 `.run/prod-parity/datastore-smoke-evidence.json`、
`.run/prod-parity/smoke-data-evidence.json`、`.run/prod-parity/identity-public-smoke-evidence.json`、
`.run/prod-parity/browser-smoke-evidence.json` 和 `.run/prod-parity/observability-smoke-evidence.json`。

发布链路补充（2026-05-25）：`build-deploy-bundle.sh` 现在会在打包前要求当前目录是 Git
worktree 且 `git status --porcelain --untracked-files=all` 为空，防止未提交 / 未签名改动绕过本机
E2E 与 prod-parity 验证进入远端部署 bundle；新增 `deploy-bundle-contract.sh` 覆盖该顺序约束。
本轮已验证 dirty worktree 下 `make deploy-bundle` 会失败，并通过 `make check-docs`、
`make check-infra-contracts` 和 `git diff --check`；提交后已在干净工作区复跑 `make deploy-bundle`，
证明正常打包路径仍可用，并确认 bundle 包含新增运维脚本但不包含 `.run`、`.deploy` 或本地生产 env。

E2E 门禁补充（2026-05-25）：Web / Admin Vite E2E API stub 改为 fail-closed，除测试基础设施
允许的 SSE / vitals 请求外，未被 Playwright route 显式 mock 的 `/api/*` 会返回
`500 E2E_UNMOCKED_API`，并由 API 4xx/5xx fixture 门禁使测试失败。该门禁暴露并补齐了首页统计、
首页热门课程、搜索院系、AppShell 身份 / 学籍 / QQ 绑定、用户中心后台 tabs、授权审计列表、课程收藏
状态和写评课草稿等真实页面依赖的 mock。本轮已通过 `PLAYWRIGHT_WEB_PORT=3100 make e2e-web`、
`make e2e-admin`、完整 `PLAYWRIGHT_WEB_PORT=3100 make e2e`、`pnpm lint:web`、`pnpm lint:admin`、
`pnpm type-check:web`、`pnpm type-check:admin` 和 `git diff --check`。

UniAppX H5 E2E 补充（2026-05-25）：新增 `clients/uniappx` 独立 Playwright 配置和
`make e2e-uni` / `pnpm test:e2e:uni` 入口，完整 `pnpm test:e2e` 现覆盖 Web、Admin 与
UniAppX H5。新增用例在桌面与移动视口下覆盖 UniAppX 首页、课程列表、课程详情、评课广场、教师主页、
写评课草稿、个人中心、我的评课 / 投票 / 收藏、通知和认证页，并复用浏览器 `pageerror`、console error、
关键资源和 API 4xx/5xx 门禁；未 mock 的 `/api/v1/*` 会返回 `500 E2E_UNMOCKED_API`。该门禁发现 H5
运行时动态 `setTabBarItem` 会抛出空对象型 pageerror，已改为 H5 使用静态 tabBar 文案、非 H5 才同步
运行时 chrome；同时补齐 `static/tabbar/*.png` 图标资源，避免 H5 tabBar 图片请求缺失。后续补齐
`pages.json` 中文静态页面标题，并在 UniAppX H5 E2E 中断言浏览器标题，确保 H5 首屏 chrome 与默认
中文界面一致；补齐课程列表搜索、加载更多和课程详情跳转 E2E，验证 `q` / `page` / `sort` 查询参数、
分页追加和搜索结果替换；补齐发布评课完整提交 E2E，验证标题、成绩、正文和评分进入创建评课 payload、提交后清理
草稿并返回课程详情；补齐课程详情收藏切换和回复提交流程 E2E，验证收藏 POST/DELETE、回复列表加载、
回复 payload 和提交后回显；补齐评课广场排序、点赞、加载更多和课程跳转 E2E，验证 sort/page 查询参数、
vote payload、乐观计数更新和二页数据追加；补齐用户中心我的评课 / 投票 / 收藏分页和课程跳转、
通知分页、单条已读、全部已读以及退出登录 E2E，验证 `page` / `pageSize` / `voteType` 查询参数、
二页数据追加、通知已读 UI 状态和 logout 调用；同时补齐登录按钮发起校园 SSO 的交互 E2E，验证 `app=uniapp`、H5 不携带 native
platform、redirect 保留和 SSO 页面跳转；并补齐 SSO callback 成功路径 E2E，覆盖一次性 state 校验、
`exchange-native` 调用、state 清理和回到个人中心，修复页面提前删除 state 导致 store 二次校验失败的问题。已通过
`pnpm --dir clients test:e2e:uni`、`pnpm --dir clients test:uni`、`pnpm --dir clients type-check:uni`、
`pnpm --dir clients build:uni:h5` 和 `git diff --check`。

UniAppX H5 E2E 补充（2026-05-25）：继续收紧发布评课草稿路径。新增浏览器断言会填写标题、合法成绩
`A`、正文和两个评分维度后点击“保存草稿”，验证 `POST /api/v1/course/review/drafts` 的 body 包含
`courseID`、默认 `termID`、`title`、`content`、`grade=A` 和完整 `ratings`，且未选择教师时不会误发
`teacherID`。该补充发现 UniAppX 文案仍把 OpenAPI/后端枚举成绩字段描述为“年级 / 2024 级”，且正式提交路径未
复用共享 `normalizeReviewGrade`；已改为按共享成绩枚举规范化，中文文案改为“成绩（可选）”，中英文 placeholder
都只提示 `A+ / B / F` 这类合法成绩。H5 fixture 还为 Vite 开发时 `@vue/devtools-api` 内部脚本的
`net::ERR_ABORTED` 添加窄范围忽略，业务 script、静态资源和 API 门禁不放宽。新增覆盖后已通过
`CI=1 UNIAPPX_E2E_PORT=3137 make e2e-uni`（28 项）、`pnpm --dir clients type-check:uni`、
`pnpm --dir clients test:uni`（46 项）、`pnpm --dir clients build:uni:h5`、`make check-docs`
和 `git diff --check`。

UniAppX H5 E2E 补充（2026-05-25）：继续补齐 SSO callback 失败显示路径。新增浏览器覆盖
`/#/pages/auth/callback?code=...&state=wrong-state` 在本地保存 state 不匹配时不会调用
`POST /api/v1/auth/exchange-native`，会清理一次性 `stuhelper:sso-state`，页面展示本地化错误
“安全校验失败，请重新登录”，点击“重新登录”返回登录页；页面实现同步把 store 的内部英文错误
`invalid native SSO state` / `missing native SSO state` 映射到现有 i18n 文案，避免 H5 用户看到内部错误。
新增覆盖后已通过
`CI=1 UNIAPPX_E2E_PORT=3140 pnpm --dir clients/uniappx exec playwright test tests/e2e/surface.spec.ts`
（30 项）、`pnpm --dir clients type-check:uni` 和完整
`CI=1 UNIAPPX_E2E_PORT=3141 make e2e-uni`（30 项）。

本地验证补充（2026-05-25）：Web 发布评课页同步收紧成绩契约，成绩字段从任意文本输入改为共享
`REVIEW_GRADES` 枚举下拉，草稿保存、草稿签名、离开提示输入判断和最终发布 payload 都复用
`normalizeReviewGrade`，避免草稿保留 `90` / `优秀` 等 OpenAPI 不接受的旧自由文本。草稿 store 现在会拒绝
接口返回的非法成绩，并把可裁剪的合法成绩归一化为共享枚举；Web 中英文文案也改为只提示 `A+ / B / F`
这类合法成绩。新增/收紧的测试覆盖草稿 store 非法成绩响应、合法成绩归一化，以及发布评课 E2E 中
自动保存草稿和创建评课 payload 都携带 `grade=A+`；Web E2E fixture 同步把无草稿时
`GET /api/v1/course/review/drafts` 的 404 标记为预期业务状态。新增覆盖后已通过
`pnpm --dir clients/web exec vitest run src/stores/__tests__/draft.test.ts src/components/business/review/__tests__/reviewPayload.test.ts`
（17 项）、`CI=1 PLAYWRIGHT_WEB_PORT=3441 pnpm --dir clients/web exec playwright test tests/e2e/review-flow.spec.ts`
（2 项）、`pnpm --dir clients type-check:web` 和 `pnpm --dir clients lint:web`。随后全量
`CI=1 PLAYWRIGHT_WEB_PORT=3442 make e2e-web` 首轮暴露移动端用户认证用例会在 `page.goto()` 整页跳转时
取消 AppShell 未读通知请求；测试已改为每次进入已登录页面后等待
`GET /api/v1/course/review/user/notifications/unread-count` 落地，再继续下一次整页跳转，不放宽全局 API
失败门禁。稳定性修复后通过
`CI=1 PLAYWRIGHT_WEB_PORT=3443 pnpm --dir clients/web exec playwright test tests/e2e/user-verification.spec.ts`
（4 项）、再次通过 `pnpm --dir clients type-check:web` 和 `pnpm --dir clients lint:web`，并最终通过
`CI=1 PLAYWRIGHT_WEB_PORT=3444 make e2e-web`（112 项）。

本地全量客户端复验（2026-05-25）：在当前提交 `4f66346e` 上执行
`CI=1 PLAYWRIGHT_WEB_PORT=3445 ADMIN_E2E_PORT=4197 UNIAPPX_E2E_PORT=3138 make e2e`，通过 Web
112 项、Admin 66 项和 UniAppX H5 28 项。该轮复验覆盖三个客户端的桌面 / 移动 Playwright project，
浏览器 `pageerror`、console error、关键静态资源和 API 4xx/5xx 门禁保持开启。

本地生产等价复验（2026-05-25）：在当前提交 `605200fb` 上执行 `make prod-parity-up`，重建并启动
tag 为 `prod-parity-605200fb` 的 backend / frontend / admin 生产镜像。共享 PostgreSQL、Casdoor、
Redis、OpenFGA、MinIO、Grafana LGTM 与应用服务均健康，应用入口为 Web `http://127.0.0.1:28000`、
Admin `http://127.0.0.1:28001/admin/`、Backend `http://127.0.0.1:28080`、Grafana
`http://127.0.0.1:23003`。本轮自动 smoke 已通过基础 API 17 项、Identity public smoke 26 项、
datastore isolation 22 项、prod-parity browser smoke 64 项和 observability smoke；evidence 写入
`.run/prod-parity/datastore-smoke-evidence.json`、`.run/prod-parity/identity-public-smoke-evidence.json`、
`.run/prod-parity/browser-smoke-evidence.json` 和 `.run/prod-parity/observability-smoke-evidence.json`。
针对本轮 Web 成绩契约修复，还额外用真实浏览器访问 prod-parity Web 生产镜像
`/courses/reviews/post`，mock 登录态和后端接口，确认 `review-grade` 下拉在生产 bundle 中可见，选择
`A+` 后自动保存草稿 payload 携带 `grade=A+`，且无浏览器 pageerror 或 console error。

本地验证补充（2026-05-25）：对照 Web 路由表继续补齐 Open Platform 资料补全页的授权边界 E2E。
新增浏览器覆盖 `/complete-profile` 缺少 token 时不请求后端并显示 fail-closed 错误态；覆盖
`/complete-profile?token=...` 初次仍缺资料、点击“重新检查”后后端返回 `missingFields=[]`、页面显示
“资料已满足本次授权请求”，再点击“我已补全，继续”后调用
`POST /api/v1/open-platform/profile-completion/continue` 且 body 只包含 profile completion token，并按
后端返回的 `redirectURL` 跳回第三方客户端。新增覆盖后已通过
`CI=1 PLAYWRIGHT_WEB_PORT=3446 pnpm --dir clients/web exec playwright test tests/e2e/open-platform-consent.spec.ts`
（10 项）、`pnpm --dir clients type-check:web`、`pnpm --dir clients lint:web` 和完整
`CI=1 PLAYWRIGHT_WEB_PORT=3447 make e2e-web`（116 项）。

本地验证补充（2026-05-25）：继续对照 Web 认证路由守卫补齐 E2E。新增覆盖已登录用户访问
`/login?reauth=1&redirect=/user/reviews` 时仍停留在登录页，点击 SSO 登录后
`GET /api/v1/auth/login` 请求携带 `app=web`、`prompt=login`、`max_age=0` 和本应用内回跳地址；
同时覆盖 `/auth/callback` 收到超过 4096 字符的 OAuth 参数时 fail-closed 到
`/login?error=invalid_callback`，并确认不会进入后端 callback。新增覆盖后已通过
`CI=1 PLAYWRIGHT_WEB_PORT=3448 pnpm --dir clients/web exec playwright test tests/e2e/auth-flow.spec.ts tests/e2e/auth-callback-and-admission.spec.ts`
（32 项）、`pnpm --dir clients type-check:web`、`pnpm --dir clients lint:web` 和完整
`CI=1 PLAYWRIGHT_WEB_PORT=3449 make e2e-web`（120 项）。

本地验证补充（2026-05-25）：继续对照 Web AppShell 交互补齐全局命令面板 E2E。新增覆盖从首页等待
shell 挂载后按 `Control+K` 打开命令面板、输入 `math` 调用
`GET /api/v1/course/courses/search?q=math&pageSize=10`、展示课程结果并点击进入 `/courses/77` 详情页；
该用例同时跑 desktop / mobile Chromium project，课程详情依赖的 reviews、rating stats、teachers 和
rating trend 接口均用 fail-closed mock 明确返回。覆盖过程中使用 Playwright MCP 事件探针确认 Chromium
向页面派发的组合键事件为 `key="K"`、`code="KeyK"`、`ctrlKey=true`，因此 `useCommandPalette`
改为统一用 `e.key.toLowerCase()` 判断 `k` / `escape`；`CommandPalette` 的 `role="dialog"` 同步补充
可访问名称，便于真实用户语义定位。新增覆盖后已通过
`CI=1 PLAYWRIGHT_WEB_PORT=3451 pnpm --dir clients/web exec playwright test tests/e2e/home.spec.ts`
（4 项）、`pnpm --dir clients type-check:web`、`pnpm --dir clients lint:web` 和完整
`CI=1 PLAYWRIGHT_WEB_PORT=3452 make e2e-web`（122 项）。

本地验证补充（2026-05-25）：继续补齐 Web AppShell 顶部通知铃的真实浏览器覆盖。`NotificationBell`
下拉面板补充 `aria-controls`、命名 `role="region"` 和显式 button 类型；新增 E2E 覆盖已登录用户进入
`/user/reviews` 后看到未读徽标、点击顶部通知按钮请求
`GET /api/v1/course/review/user/notifications?page=1&pageSize=5`、渲染通知预览、点击“全部已读”调用
`PUT /api/v1/course/review/user/notifications/read-all` 并清除徽标、继续点击单条通知调用
`PUT /api/v1/course/review/user/notifications/{id}/read` 后按 `sourceUrl` 跳转 `/about`。该用例同时跑
desktop / mobile Chromium project。除自动 E2E 外，还启动 `VITE_E2E_API_STUB=1` 的本地 Vite，通过
Playwright MCP `browser_run_code_unsafe` 注入同样的 API mock 并打开 `/user/reviews`，确认通知按钮
`aria-expanded=true` 且命名通知区域文本包含“顶部提醒 / 新的点赞 / 全部已读 / 查看全部”。新增覆盖后已通过
`CI=1 PLAYWRIGHT_WEB_PORT=3453 pnpm --dir clients/web exec playwright test tests/e2e/journey-user-center.spec.ts`
（18 项）、`pnpm --dir clients type-check:web`、`pnpm --dir clients lint:web` 和完整
`CI=1 PLAYWRIGHT_WEB_PORT=3454 make e2e-web`（124 项）。

本地验证补充（2026-05-25）：继续补齐 Web AppShell 语言切换入口的真实浏览器覆盖。新增首页 E2E
覆盖默认中文 `html[lang=zh-CN]`、点击“切换到英文”后首页文案切换为
`StuHelper Course Review Community`、按钮变为 `Switch to Chinese`、`localStorage.locale=en-US`，刷新页面后
仍保持英文，再点击切回中文并确认 `html[lang=zh-CN]` 和 `localStorage.locale=zh-CN`。该用例同时跑
desktop / mobile Chromium project。除自动 E2E 外，还启动 `VITE_E2E_API_STUB=1` 的本地 Vite，通过
Playwright MCP 注入匿名首页 API mock，确认默认中文按钮可见、切换后 `html.lang=en-US`、切回按钮可见、
刷新后仍保留英文且无 console error。新增覆盖后已通过
`CI=1 PLAYWRIGHT_WEB_PORT=3456 pnpm --dir clients/web exec playwright test tests/e2e/home.spec.ts`
（6 项）、`pnpm --dir clients type-check:web`、`pnpm --dir clients lint:web` 和完整
`CI=1 PLAYWRIGHT_WEB_PORT=3458 make e2e-web`（126 项）。

本地验证补充（2026-05-25）：继续补齐 Web Header 响应式导航的真实点击覆盖。新增浏览器用例按
Playwright project viewport 分支：桌面项目直接通过主导航点击“课程”，移动项目等待“菜单”按钮、
断言 `aria-expanded=false -> true`、打开 `#app-mobile-nav` 后点击“课程”，并确认路由进入 `/courses`、
移动菜单在路由切换后卸载且课程 hub 标题“评课社区@BUAA”可见。覆盖过程中首轮测试暴露用
`locator.isVisible()` 做同步分支判断会在移动端 Header 尚未完成渲染时误判到桌面分支；已改为先等待
AppShell 品牌可见，再按 viewport 宽度选择桌面 / 移动路径。除自动 E2E 外，还启动
`VITE_E2E_API_STUB=1` 的本地 Vite，通过 Playwright MCP 将视口设为 `390x844`，注入匿名课程 API mock，
实际点击“菜单 -> 课程”，确认最终 URL 为 `/courses`、菜单按钮展开状态从 `false` 到 `true`，导航后
`#app-mobile-nav` 数量为 0。新增覆盖后已通过
`CI=1 PLAYWRIGHT_WEB_PORT=3459 pnpm --dir clients/web exec playwright test tests/e2e/journey-browse.spec.ts`
（8 项）、`pnpm --dir clients type-check:web`、`pnpm --dir clients lint:web` 和完整
`CI=1 PLAYWRIGHT_WEB_PORT=3461 make e2e-web`（128 项）。

本地验证补充（2026-05-25）：继续补齐 Web AppShell 已登录用户菜单的真实浏览器覆盖。新增
`journey-user-center` E2E 覆盖已登录用户进入 `/user/reviews` 后打开顶部用户按钮，验证菜单为命名
`role="menu"` 且包含个人中心、开发者应用、实名认证、学生认证、绑定 QQ 和退出登录入口；点击
“退出登录”后断言调用 `POST /api/v1/auth/logout`、路由回到首页、顶部登录入口可见，并确认
`localStorage.stuhelper_user` 与 `localStorage.stuhelper_token_expiry` 均被清理。测试内对登出后的
`auth/me` 明确返回 401，避免退出后被登录态 mock 重新拉起。除自动 E2E 外，还启动
`VITE_E2E_API_STUB=1` 的本地 Vite，通过 Playwright MCP `browser_run_code_unsafe` 注入同等 API mock，
实际打开 `/user/reviews` 并点击用户菜单退出，确认登出请求、菜单项、首页跳转和本地会话清理均符合预期，
且无 console error。新增覆盖后已通过
`CI=1 PLAYWRIGHT_WEB_PORT=3462 pnpm --dir clients/web exec playwright test tests/e2e/journey-user-center.spec.ts`
（20 项）、`pnpm --dir clients run type-check:web`、`pnpm --dir clients run lint:web` 和完整
`CI=1 PLAYWRIGHT_WEB_PORT=3463 make e2e-web`（130 项）。

本地验证补充（2026-05-25）：继续补齐 Web AppShell 管理员入口的真实浏览器覆盖。新增独立管理员登录态
E2E，初始化 `canAccessAdmin=true` / `isPlatformAdmin=true` 用户进入 `/user/reviews`，验证顶部用户头像旁
“管理员”徽标可见，用户菜单包含“管理后台”，点击后按主站默认后台入口跳转到 `/admin/`。该用例同时跑
desktop / mobile Chromium project，避免普通用户菜单覆盖遗漏管理员专属入口。除自动 E2E 外，还启动
`VITE_E2E_API_STUB=1` 的本地 Vite，通过 Playwright MCP `browser_run_code_unsafe` 注入管理员 API mock，
实际打开 `/user/reviews`、点击用户菜单中的“管理后台”，确认徽标、菜单项和最终
`http://127.0.0.1:3466/admin/` URL 均符合预期，且无 console error。新增覆盖后已通过
`CI=1 PLAYWRIGHT_WEB_PORT=3465 pnpm --dir clients/web exec playwright test tests/e2e/journey-user-center.spec.ts`
（22 项）、`pnpm --dir clients run type-check:web`、`pnpm --dir clients run lint:web` 和完整
`CI=1 PLAYWRIGHT_WEB_PORT=3467 make e2e-web`（132 项）。

本地验证补充（2026-05-25）：继续补齐 Admin 顶栏用户下拉退出登录的真实浏览器覆盖。新增
`admin-core` E2E 覆盖已登录管理员进入 `/profile` 后点击顶栏头像按钮，打开 Vben user dropdown，
点击“退出登录”后显示页面内确认弹窗“是否退出登录？”，点击“确认”调用
`POST /api/v1/auth/logout`，随后发起 `GET /api/v1/auth/login`，并断言 `app=admin` 且
`redirect` 保留当前 `/profile` 回跳目标。测试过程中发现空头像会渲染隐藏 `img`，真实可点击目标是顶栏
fallback 文本为 `IN` 的 button；还发现主动 `page.goto('about:blank')` 会中断正在加载的 Vite 脚本，
已改为让 mocked SSO URL 自身返回 `about:blank`。除自动 E2E 外，还启动 Admin 本地 Vite，通过
Playwright MCP `browser_run_code_unsafe` 注入同样的管理员 session / logout / login mock，实际点击
顶栏头像下拉并确认退出，确认 logout/login 请求、`app=admin`、`redirect=http://127.0.0.1:4209/profile`
和最终 `about:blank` 跳转均符合预期，且无 console error。新增覆盖后已通过
`CI=1 ADMIN_E2E_PORT=4208 pnpm --dir clients/admin --filter @vben/web-ele exec playwright test tests/e2e/admin-core.spec.ts`
（6 项）、`pnpm --dir clients/admin --filter @vben/web-ele typecheck`、`pnpm --dir clients/admin lint`
和完整 `CI=1 ADMIN_E2E_PORT=4210 make e2e-admin`（68 项）。

本地验证补充（2026-05-25）：继续补齐 Admin dashboard 首页入口的真实浏览器导航覆盖。新增
`admin-surface` E2E 从 `/analytics` 点击“教师管理”进入 `/content/teachers` 并确认教师数据渲染，
再从 `/workspace` 点击“待处理举报”进入 `/content/reports`、点击“实名审核”进入
`/users/identity-review`，避免统计卡片和快捷入口只渲染但路由动作未被锁住。该用例同时跑 desktop /
mobile Chromium project。除自动 E2E 外，还启动 Admin 本地 Vite，通过 Playwright MCP 在 `390x844`
视口注入同等管理员 API mock，实际点击 analytics quick action 与 workspace 队列 / 快捷入口，确认最终
URL、目标页面数据和 Admin API 请求路径均符合预期。新增覆盖后已通过
`CI=1 ADMIN_E2E_PORT=4212 pnpm --dir clients/admin --filter @vben/web-ele exec playwright test tests/e2e/admin-surface.spec.ts`
（24 项）、`pnpm --dir clients run type-check:admin`、`pnpm --dir clients run lint:admin` 和完整
`CI=1 ADMIN_E2E_PORT=4214 make e2e-admin`（70 项）。

本地验证补充（2026-05-25）：继续补齐 UniAppX H5 未登录用户菜单认证边界。新增 E2E 覆盖未登录用户进入
`/#/pages/user/index`，点击“我的评课”后进入 `/pages/auth/login`，再点击“使用校园 SSO 登录”调用
`GET /api/v1/auth/login`，并断言 `app=uniapp`、不携带 `platform`、`redirect=/pages/user/reviews`。
覆盖过程中发现 H5 `uni.navigateTo` 会把已编码的 redirect 再编码成 `%252F...`，导致登录页原样使用
`options.redirect` 时可能把 `%2Fpages%2F...` 传给 SSO；已在 UniAppX 登录页对 redirect 做最多两次解码，
并限制只能回跳 `/pages/...` 站内路径。除自动 E2E 外，还启动 UniAppX 本地 H5，通过 Playwright MCP
`browser_run_code_unsafe` 注入匿名 session / login mock，实际从用户中心点击“我的评课”进入登录页并发起
SSO，确认地址栏可见框架二次编码但 login API 请求的 redirect 已规范化为 `/pages/user/reviews`，最终跳到
mock SSO 页面且无 console error。新增覆盖后已通过
`CI=1 UNIAPPX_E2E_PORT=3144 pnpm --dir clients/uniappx exec playwright test tests/e2e/surface.spec.ts`
（32 项）、`pnpm --dir clients run type-check:uni`、`pnpm --dir clients run test:uni` 和完整
`CI=1 UNIAPPX_E2E_PORT=3146 make e2e-uni`（32 项）。

本地验证补充（2026-05-25）：继续补齐 UniAppX H5 评课广场未登录投票认证边界。新增 E2E 覆盖未登录用户进入
`/#/pages/review/index` 后点击评课点赞按钮，前端不会调用
`POST /api/v1/course/review/reviews/{id}/votes`，而是跳转登录页；随后点击“使用校园 SSO 登录”会调用
`GET /api/v1/auth/login`，并断言 `app=uniapp`、不携带 `platform`、`redirect=/pages/review/index`。
除自动 E2E 外，还启动 UniAppX 本地 H5，通过 Playwright MCP 在 `390x844` 视口注入同等 API mock，
实际点击评课点赞并进入 mock SSO，确认登录页地址栏仍可见框架二次编码但 login API 请求的 redirect
已规范化为 `/pages/review/index`，且未发送投票 mutation。新增覆盖后已通过
`CI=1 UNIAPPX_E2E_PORT=3148 pnpm --dir clients/uniappx exec playwright test tests/e2e/surface.spec.ts`
（34 项）、`pnpm --dir clients run type-check:uni`、`pnpm --dir clients run test:uni` 和完整
`CI=1 UNIAPPX_E2E_PORT=3150 make e2e-uni`（34 项）。

本地全量复验（2026-05-25）：在提交 `d82afd53` 上复跑统一客户端 E2E 门禁，
`CI=1 PLAYWRIGHT_WEB_PORT=3450 ADMIN_E2E_PORT=4206 UNIAPPX_E2E_PORT=3142 make e2e`
通过 Web 120 项、Admin 66 项和 UniAppX H5 30 项，覆盖本轮 Web 重新认证 / callback guard 与
UniAppX callback state mismatch 本地化修复后的组合状态。

本地工具链修复（2026-05-25）：排查发现本机 Codex Playwright MCP 已配置但当前会话未暴露
`browser_*` 工具；配置使用大写 server 名 `Playwright` 和动态 `@playwright/mcp@latest`。同时 GitLab /
ace-tool 仍是 Windows `cmd /c` 写法，`codex doctor` 在 Ubuntu 24.04 下报告这些 stdio 命令不可解析。
已将 `~/.codex/config.toml` 中 Playwright MCP 改为稳定的小写 server 名和 Linux 可执行的
`command = "npx"`、`args = ["-y", "@playwright/mcp@0.0.75", "--headless"]`，并将 GitLab / ace-tool
MCP 从 Windows `cmd /c` 改为 Ubuntu 下直接执行 `npx`。修复后 `codex mcp list` 显示 `playwright`
enabled，MCP `tools/list` 返回 `browser_navigate`、`browser_snapshot`、`browser_click` 等工具；
当前会话已通过 `tool_search` 暴露 `mcp__playwright__`，并实际用 `browser_navigate` / `browser_snapshot`
完成页面快照验证。

本地验证补充（2026-05-25）：当前会话再次通过 `tool_search` 暴露 `mcp__playwright__`，并用
`browser_navigate` 打开最小检查页、`browser_evaluate` 返回 title、heading 和 button 数量，确认
Playwright MCP 在 Codex 会话内已经可用。随后继续补齐 Koishi Console 订阅管理真实写路径 E2E：
新增用例进入“推送订阅”，通过 Element Plus Drawer 添加订阅、切换 feature checkbox、保存后断言卡片
feature 状态，再次编辑关闭“防撤回”、打开“禁言解除”，最后通过 popconfirm 删除订阅并确认目标卡片消失。
该用例覆盖 `stuhelperGroupCenter/subscriptions/add`、`update` 和 `remove` 的浏览器 UI 到 WebSocket API
链路。新增覆盖后已通过
`STUHELPER_UI_SMOKE_PORT=5147 corepack yarn --cwd bots/koishi test:ui`（18 项）和
`corepack yarn --cwd bots/koishi test:unit`（52 项）。

本地验证补充（2026-05-25）：本轮再次通过 `tool_search` 暴露 `mcp__playwright__`，先用
`browser_resize` 和 `browser_snapshot` 确认 MCP 可以操作当前浏览器，再启动隔离的临时 Koishi
实例 `http://127.0.0.1:5151`，使用 Playwright MCP 真实登录 admin、进入 `#system`，点击“刷新统计”、
确认“强制刷新缓存”、确认“清空缓存”，最终页面停在 `/stuhelper#system`，通知消息包含“缓存刷新完成”
和“缓存已清空”。同时补齐 Koishi Console 系统 / 缓存页 E2E：新增用例覆盖缓存统计刷新、强制刷新
ConfirmDialog、清空缓存 ConfirmDialog 与 NoticeStack 成功提示，确保 `cache/stats`、`cache/refresh`
和 `cache/clear` 的浏览器 UI 到 WebSocket API 链路不断裂。新增覆盖后已通过
`STUHELPER_UI_SMOKE_PORT=5149 corepack yarn --cwd bots/koishi test:ui`（19 项）和
`corepack yarn --cwd bots/koishi test:unit`（52 项）。

本地验证补充（2026-05-25）：继续补齐 Koishi Console 黑名单真实写路径。`scripts/ui-smoke.mjs`
内置平台 stub 改为对 `member-blacklist` 保持内存状态，支持 `list` 按 platform / subjectType /
scopeType / guildID / active status 过滤、按 page/pageSize 分页，并在 create / release /
release-by-subject 后返回与后续 list 一致的条目状态，避免 UI 测试只能验证“请求成功”而无法验证列表状态。
新增 E2E 用例进入“黑名单”，通过 Drawer 添加全局黑名单、确认全局范围风险弹窗、断言列表行出现并展示
全局范围与原因，再点击“解除”、确认解除弹窗并断言目标行从 active 列表消失。该用例覆盖
`stuhelperGroupCenter/blacklist/list`、`add` 和 `remove` 的浏览器 UI 到 WebSocket API 再到平台客户端
链路。Playwright MCP 也在临时 Koishi `http://127.0.0.1:5155` + 临时平台 stub
`http://127.0.0.1:5154` 上复验同一路径，最终停在 `/stuhelper#blacklist`，通知消息包含“已将
目标用户加入黑名单”和“已从黑名单解除目标用户”，目标行计数为 0，未发现新增 console error/warning。
新增覆盖后已通过 `STUHELPER_UI_SMOKE_PORT=5153 corepack yarn --cwd bots/koishi test:ui`（20 项）和
`corepack yarn --cwd bots/koishi test:unit`（52 项）。

本地生产等价复验（2026-05-25）：在提交 `45b401c2` 上重新执行 `make prod-parity-up`，构建并启动
tag 为 `prod-parity-45b401c2` 的 backend / frontend / admin 生产镜像。当前本机生产等价入口为
Web `http://127.0.0.1:28000`、Admin `http://127.0.0.1:28001/admin/`、Backend
`http://127.0.0.1:28080`、Grafana `http://127.0.0.1:23003`，三项应用容器均 healthy。本轮自动检查通过
基础 API 17 项、datastore isolation 22 项、Identity public smoke 26 项、prod-parity browser smoke
64 项和 observability smoke 6 项；smoke data seed 也通过。evidence 写入
`.run/prod-parity/datastore-smoke-evidence.json`、`.run/prod-parity/identity-public-smoke-evidence.json`、
`.run/prod-parity/browser-smoke-evidence.json`、`.run/prod-parity/observability-smoke-evidence.json` 和
`.run/prod-parity/smoke-data-evidence.json`。

接口硬化补充（2026-05-24）：审计发现 legacy disclosure API 文档要求 `client_id`，但
OpenAPI 未声明且 handler 只读取认证上下文 appID；同时服务端在已有 active consent 时未重新校验
`redirect_uri`。本轮已按 OpenAPI-first 补齐 `GET /api/v1/open-platform/userinfo`、
`/verification`、`/student`、`/phone` 的必填 `client_id` 参数和 cookie/bearer 安全声明，
handler 改为优先使用请求 `client_id`、保留认证 appID fallback，并在显式 Bearer app identity 与
请求 `client_id` 不一致时拒绝；service 在 disclosure 已授权路径也会校验 redirect URI 精确匹配，
并写入 `client_mismatch` / `redirect_uri_not_allowed` 拒绝审计。验证已通过
`make generate`、`make lint-spec`、`go test -count=1 ./internal/modules/openplatform`、
`go test -count=1 ./...`、`pnpm --dir clients/shared test`、`pnpm --dir clients/shared type-check`、
`pnpm type-check:all`（`clients/` 工作目录）、`make check-docs` 和相关文件 `git diff --check`。
后续同类审计又补齐 Open Platform legacy `/authorize` 与 `/userinfo`、`/verification`、`/student`、
`/phone` disclosure 查询入口的单值参数守卫，重复 `client_id`、`redirect_uri`、`scope`、`state`
或 `consent_base_url` 会在解析用户或调用 service 前返回 `400 invalid_param`；共享认证中间件也不再在
重复、空 Bearer 或非 Bearer `Authorization` 头存在时回退到浏览器 cookie，避免 API credential
语义和浏览器会话语义混用；bot 服务账号接口也会拒绝重复 `Authorization` 头且不会调用 verifier。
Open Platform 应用列表、管理员审计、管理员授权、开发者审计、token probe evidence、资源授权、
disclosure report 和用户授权审计查询也已拒绝重复的单值过滤参数，并拒绝 `page_size` / `pageSize`
同时出现的分页别名歧义。主站 / 管理后台 SSO 接入使用的 `/api/v1/auth/login`、`/signup`、
`/step-up` 和 `/callback` 也会拒绝重复 `redirect`、`platform`、`app`、`prompt`、`max_age`、`code`
或 `state`，避免应用选择、重新认证触发、回跳地址或授权码回调由框架默认取首值造成歧义；本地
`/api/v1/auth/refresh` 也已将 native JSON body refresh token 与浏览器 cookie refresh token 设为互斥
凭据来源，请求体存在时必须是合法 JSON 且不能再携带 auth/session/CSRF cookie，避免 malformed body
静默回退到 cookie 或 body refresh 绕过 cookie refresh 的 CSRF 语义。
新增回归测试已覆盖上述入口，并通过
`go test -count=1 ./internal/modules/openplatform`、`go test -count=1 ./internal/pkg/middleware` 和
`go test -count=1 ./...`。

## 近期已完成

| 任务 | 完成状态 |
|------|----------|
| Open Platform 用户授权管理 | 主站用户中心已提供已授权应用列表；用户可按 scope 或整应用撤销授权，撤权使用页面内确认对话框展示影响范围，不再依赖浏览器原生 confirm；授权页、资料补全页和授权列表会按 scope 展示开发者提交的用途说明，列表还展示授权时间和从 disclosure granted 审计派生的最近成功披露时间；用户中心会展示当前用户自己的 consent grant/deny/revoke、disclosure granted/denied 和 replay_detected 授权活动摘要；用户明确拒绝授权会先写入 `open_platform.consent.denied` 审计再删除 challenge；用户主动撤销、管理员定向撤销或管理员吊销应用后，disclosure / UserInfo 重新返回 consent required，并写入 grant/revoke 审计事件。 |
| Open Platform 授权列表查询可扩展性 | `open_platform_user_consents` 已提供 active user 和 active app partial index；`open_platform_audit_events` 已提供 disclosure granted app/user/created partial index；用户授权列表、管理员按 app/user 查看 active consent 和 per-scope `lastUsedAt` 派生不会依赖审计表全量扫描，迁移后索引存在性由 Open Platform 集成测试锁定。 |
| Open Platform 审计 retention | `open_platform_audit_events` 已接入运行时后台 retention cleanup；高频 disclosure 与 `resource_access.checked` 审计默认保留 365 天，应用审批、consent、scope、redirect URI、密钥轮换、资源授权等运营审计默认保留 1095 天，仓库层使用参数化 chunked delete 和 `FOR UPDATE SKIP LOCKED`，避免生产无界大删除。 |
| Open Platform 开发者门户基础页 | 主站 `/developers/apps` 已提供应用列表、创建表单、状态筛选和 scope 审核状态展示，并从用户菜单提供入口。 |
| Open Platform 开发者门户完整化 | 主站 `/developers/apps` 已支持开发者提交应用、维护非 revoked 应用展示资料、查看 app / scope / redirect URI 审核状态、撤回 pending 应用 / scope / redirect URI 申请、新增 scope 或重提 rejected / withdrawn scope、申请 redirect URI 变更、轮换 approved 应用的 client secret，并查看自有应用生命周期、资料变更、审批、授权、披露、资源授权和 token 探针审计摘要；密钥轮换和撤回类敏感操作使用页面内确认表单采集审计原因，不再依赖浏览器原生 prompt；注册、legacy 导入、新增或重提 scope 均要求非空用途说明；开发者审计接口强制 owner 隔离并裁剪用户 ID、原始 token claims 和内部错误细节。 |
| Open Platform app 生命周期与密钥轮换 | 开发者可轮换自己 approved app 的 client secret；管理员可轮换 approved / suspended app secret、暂停 approved app、恢复 suspended app、吊销 pending / approved / suspended app；旧 secret 立即失效；恢复不会自动恢复用户已撤销 consent；吊销会撤回 pending 子申请并撤销该 app 全部 active user consent，生命周期与 consent 处置均写入 `open_platform_audit_events`。 |
| Open Platform 管理员审计查看 | 管理后台 `/open-platform/audit-events` 调用 `GET /api/v1/admin/open-platform/audit-events`，可按 app、用户、事件类型和 scope 检索 `open_platform_audit_events`，并查看 request ID 与 JSON metadata；app 恢复、资源授权 checked/granted/revoked 等事件已进入共享 audit taxonomy，admin 筛选项和开发者活动标签都有漂移测试覆盖。 |
| Open Platform 管理员授权处置 | 管理后台 `/open-platform/consents` 调用 `GET /api/v1/admin/open-platform/consents`，可按 appID 或 userID 查看 active consent；管理员可按用户撤销整 app 授权或单个 scope，撤销审计包含 actor、actorUserID、reason 和 `source=admin_console`。 |
| OIDC / OAuth metadata 兼容 | `id.stuhelper.com` 已同时暴露 OIDC discovery `/.well-known/openid-configuration` 和 RFC 8414 OAuth authorization server metadata `/.well-known/oauth-authorization-server`，两者返回同一 issuer、endpoint、scope、grant 和 client authentication method 基线，兼容只消费 OAuth2 AS metadata 的网关 / 资源服务器；公网 identity smoke 已把两个 well-known 路径纳入门禁，公网诊断脚本也会单独检查 OAuth authorization server metadata 并在漏转时标记 `identity_oauth_as_metadata_not_proxied`。 |
| OIDC 静默授权兼容 | `id.stuhelper.com` discovery 已暴露 `prompt_values_supported=["none","login","consent"]`；未登录 `prompt=none` 授权请求会在校验 app、redirect URI 和 S256 PKCE 后回调 `error=login_required`，已登录但缺少资料或 consent 时回调 `error=interaction_required` / `error=consent_required`，不会创建临时 consent/profile challenge；公网 identity smoke 已把 discovery 中的 `prompt_values_supported` 纳入门禁。 |
| OIDC RP-Initiated Logout | `id.stuhelper.com` discovery 已暴露 `end_session_endpoint=/oauth2/logout`；第三方应用可用 `client_id` 或 `id_token_hint` 发起登出，`post_logout_redirect_uri` 必须精确匹配 approved app 已注册 redirect URI；请求一旦携带 `id_token_hint`，即使没有回跳 URI，也必须能验证为 StuHelper Identity ID token，access token / app-only token 不能作为 logout hint；存在当前 StuHelper 会话时会撤销 session 并清理认证 cookie，公网 identity smoke 已把 `end_session_endpoint` 和无回跳非 ID token hint 拒绝纳入门禁。 |
| OIDC refresh token / offline access | `offline_access` 已纳入 Open Platform scope catalog、开发者申请页和管理员导入 / 审计筛选项；只有经 app 审批和用户同意的授权码才会返回 refresh token，`refresh_token` grant 每次执行 rotation，并重新校验 app、scope、用户 active consent 和 token 发行时的 consent 指纹，用户撤权、重新确认授权或重新授权后旧 token 都无法继续刷新；刷新请求携带 `scope` 时只能收窄到原 grant 子集且必须保留 `openid offline_access`，下一代 refresh token 继承收窄后的 scope，扩展返回 `invalid_scope`；公网 identity smoke 已把 `refresh_token` grant 和 `offline_access` scope 纳入 discovery 门禁。 |
| OIDC offline_access / openid 组合约束 | Identity authorize request 现在会拒绝缺少 `openid` 的 `offline_access` 组合，防止第三方获得语义不完整、后续 refresh grant 又无法使用的离线授权；token exchange 对历史或异常 authorization code 也会拒绝 `offline_access` without `openid`；refresh grant 和 refresh-token introspection 对历史或异常 refresh token 同样要求同时具备 `openid offline_access`，service 和 handler 测试锁定该契约。 |
| OIDC client credentials / app-only token | `id.stuhelper.com` token endpoint 已支持 `client_credentials` grant；仅允许 approved app 为已审批的 `resource.read` / `resource.write` scope 换取 app-only access token，不返回 `id_token` 或 `refresh_token`；introspection 会返回 RFC 7662 `token_type=Bearer`、StuHelper 扩展 `token_kind=access_token`、`grant_type=client_credentials`，并重新校验 app / scope 当前状态，UserInfo 拒绝该类 token；`/api/v1/open-platform/resources/access/check` 已支持 Bearer app-only token，缺少本次动作所需资源 scope 时返回 `allowed=false` / `reason=token_scope_missing`。 |
| OIDC refresh token TTL、存储和重放硬化 | StuHelper Identity refresh token 有效期已由 `IDENTITY_REFRESH_TOKEN_TTL` 控制，默认 2592000 秒；配置校验限制在 3600 到 2592000 秒，服务启动时将该 TTL 注入 refresh token Redis 存活时间和 payload 过期时间；Redis key 改为 refresh token 哈希，避免 keyspace 或备份直接暴露可使用 token；refresh grant 和 refresh token introspection 都要求 token hash 仍匹配 family `currentKey`，family 缺失、撤销或指向其他 token 时 fail-closed；access token 与 refresh token 会携带当前 active consent 指纹，UserInfo、introspection 和 refresh grant 会拒绝指纹不匹配的旧 token，防止用户撤权后又重新授权时旧 token 复活；refresh grant 会先消费当前 refresh token 再签发新 access token / ID token，避免并发重放失败请求产生 disclosure 副作用；同一 client 重放已消费 refresh token 会撤销当前 refresh token family，并写入 `iam.token.revoked` 审计事件；同一 client 用已消费 refresh token 调用 `/oauth2/revoke` 也会撤销当前 family，关闭登出与刷新 rotation 并发时旧 token no-op 的窗口；生产样板和迁移文档均已同步。 |
| OIDC token revocation 审计 | `/oauth2/revoke` 所属 client 成功撤销 Identity access token、当前 refresh token 或已被 rotation 消费的 refresh token 时会写入 `iam.token.revoked`；跨 client revoke 仍保持 no-op 且不写审计；审计只记录 access token JTI 或 refresh token family hash，不记录原始 token，且已消费 refresh token 的 revoke 会删除 used-token 记录以避免重复撤销审计；撤销黑名单 / refresh family 查找或删除失败时返回 `503 server_error`，不再把未确认落盘的撤销误报为 `200`。 |
| OIDC refresh token introspection | `/oauth2/introspect` 已支持 refresh token；只有 token 所属 client 能看到 `active=true`，并继续按当前 app approved、scope approved、用户 active consent 和发行时 consent 指纹重新校验；已 rotation 消费、revoke、旧 refresh token 触发 revoke family、重放撤销 family、当前授权失效或 consent 指纹变化后返回 `active=false`。 |
| OIDC client authentication hardening | `/oauth2/token`、`/oauth2/introspect`、`/oauth2/revoke` 支持 `client_secret_basic` 与 `client_secret_post`，但同一请求混用 Basic 头和 body `client_id` / `client_secret` 时返回 `invalid_client`，即使 body 参数为空、重复或只有空白也按“已携带”处理；重复 `Authorization` 头、任何非 Basic 或畸形 Basic 的 `Authorization` 头也返回 `invalid_client`，不会退回 body credential，防止凭据歧义和跨客户端探测。 |
| OIDC required parameter hardening | `/oauth2/token`、`/oauth2/introspect` 和 `/oauth2/revoke` 只接受 `application/x-www-form-urlencoded` body form 参数，缺失 Content-Type、JSON、text/plain 或任何 URL query 参数都返回 `400 invalid_request`，避免 client secret、authorization code、refresh token、access token 或 introspection token 被写入代理 / Web 访问日志，也避免非表单 body 被静默解析为空表单；`/oauth2/token` 要求非空 `grant_type`，`/oauth2/introspect` 和 `/oauth2/revoke` 通过 client authentication 后要求非空 `token`；缺失或空白必填参数统一返回 `400 invalid_request`，不会把空 token 当作 inactive token 或 revoke no-op；公网 identity smoke 已把 token 缺失 `grant_type` 的错误从旧的 `unsupported_grant_type` 更新为 `invalid_request`，并纳入 token JSON Content-Type、introspection text/plain Content-Type、revoke 缺失 Content-Type 以及 token / introspection / revoke URL query 参数的 `invalid_request` 门禁；配置 smoke client secret 时还会真实验证已认证 client 的缺失 `grant_type`、空 introspection token 和空 revoke token 都返回 `invalid_request`。 |
| OIDC single-value parameter hardening | `/oauth2/authorize`、`/oauth2/continue`、RP-Initiated Logout、`/oauth2/token`、`/oauth2/introspect` 和 `/oauth2/revoke` 会拒绝重复出现的 OAuth / OIDC 单值参数，避免 redirect URI、state、grant_type、token、token_type_hint 或 client authentication 参数由框架默认取首值造成歧义；`/oauth2/continue`、Open Platform consent 页和 profile completion 页的一次性 challenge token 必须恰好出现一次且非空；授权和登出端点返回 invalid authorization/logout request，RP-Initiated Logout POST 不接受 URL query 参数且有 body 时只接受 `application/x-www-form-urlencoded`，避免 logout hint、client 和 redirect 参数落入 URL 日志或非表单 body 被静默当作空请求；token / introspection / revoke 端点返回 `400 invalid_request` 且保持 no-store/no-cache；UserInfo 只接受单个 `Authorization: Bearer <access_token>`，任何 URL query、POST body 或重复 `Authorization` 头都会返回 `401 invalid_token` 并带 Bearer challenge，避免 access token 进入 URL / body 日志或 token 来源歧义。 |
| OIDC token type hint 兼容 | `/oauth2/introspect` 和 `/oauth2/revoke` 接受标准 `token_type_hint`；revoke 会按 `access_token` / `refresh_token` hint 优先查找，未找到时继续查所有支持 token 类型，未知 hint 按兼容策略忽略；公网 identity smoke 的 client credentials revoke / introspection 已携带 `token_type_hint=access_token`。 |
| OIDC token audience hardening | Identity Server 验证 access token 时要求 `aud` 包含 `client_id` 且 `azp` 与 `client_id` 一致；验证 ID token 时要求 `aud` 存在，多 audience token 必须携带属于 `aud` 的 `azp`，单 audience token 如携带 `azp` 也必须确认其属于 `aud`，防止签名有效但 audience / authorized party 错误的 token 被内部 UserInfo、introspection、资源访问或 logout hint 路径接受。 |
| OIDC opaque state 透传 | 授权成功、可回调授权错误、用户拒绝 consent 和 RP-Initiated Logout 回调会保留客户端提交的非空 `state` 原值，不做空白裁剪，避免严格客户端的 CSRF/state 校验因 opaque value 被改写而失败。 |
| Open Platform resource access credential hardening | 资源访问检查 API 首选 Bearer app-only token，兼容模式保留 body `clientID` / `clientSecret`；两种认证方式现在互斥，携带 Bearer 时 body 再传 client credential 会返回 invalid request，非 Bearer 或重复 Authorization 头会直接拒绝，避免应用密钥混入请求体或产生认证来源歧义。 |
| OIDC 敏感响应缓存硬化 | `/oauth2/token`、`/oidc/userinfo`、`/oauth2/introspect`、`/oauth2/revoke` 的成功和错误响应统一返回 `Cache-Control: no-store` 与 `Pragma: no-cache`，避免 token、UserInfo 或 introspection 结果被浏览器、代理或客户端调试缓存保存；Identity handler 单元测试和公网 identity smoke 契约测试覆盖 token/UserInfo/introspection/revoke 成功与错误路径。 |
| OIDC 401 challenge 兼容性 | token / introspection / revoke 的 `invalid_client` 401 响应会返回 Basic `WWW-Authenticate` challenge；UserInfo 的 `invalid_token` 401 响应会返回 Bearer `WWW-Authenticate` challenge，让标准 OAuth / OIDC 客户端能区分 client credential 失败和 Bearer token 失败；Identity handler 单元测试和公网 identity smoke 契约测试覆盖这些响应头。 |
| OIDC response mode 兼容性 | `id.stuhelper.com` discovery 已暴露 `response_modes_supported=["query"]`；授权请求显式携带 `response_mode=query` 时仍按 query 回调返回 code / error / `state` / RFC 9207 `iss`，`fragment`、`form_post` 或带空白的 response mode 会返回 invalid authorization request；Identity service 单元测试和公网 identity smoke 契约测试覆盖该 capability。 |
| OIDC ID token scope 最小化 | 授权码和 refresh token grant 只有在已授予 OAuth scope 包含 `openid` 时才返回 `id_token`；纯 `resource.read` / `resource.write` 授权码流只返回 user-delegated access token，UserInfo 会以 `invalid_token` 拒绝不含 `openid` 的 access token；Identity service 单元测试覆盖 token response、UserInfo 拒绝和 introspection 仍可按资源 scope 校验 active。 |
| OIDC authorization response issuer | `id.stuhelper.com` discovery 已暴露 `authorization_response_iss_parameter_supported=true`；授权成功回调和可回调授权错误响应都会携带 RFC 9207 `iss=https://id.stuhelper.com`，公网 identity smoke 已把该 discovery capability 纳入门禁。 |
| OIDC prompt=login / max_age 重新认证 | `id.stuhelper.com` discovery 已暴露 `prompt_values_supported=["none","login","consent"]`；第三方授权请求携带 `prompt=login` 或解析后为 0 的 `max_age` 时会转接到登录页并通过 `/api/v1/auth/login?prompt=login&max_age=0` 触发 Casdoor 真实重新认证，回跳前消费掉会造成循环的 reauth 参数，包括 `max_age=00` 这类零值表示；正数 `max_age` 按当前会话 `auth_time` 判断，缺失可信 `auth_time` 时 fail-closed 到重新认证；公网 identity smoke 已加入 `prompt=login&max_age=0` 黑盒跳转门禁。 |
| OIDC prompt=consent 强制授权确认 | `id.stuhelper.com` discovery 已暴露 `prompt_values_supported=["none","login","consent"]`；第三方授权请求携带 `prompt=consent` 时，即使用户已对本次 disclosure scope 授权，也会重新展示 StuHelper consent 页并刷新 consent 审计；`prompt=login consent` 会先触发重新认证，回跳后保留 `prompt=consent` 继续授权确认。 |
| StuHelper 本地 phone OTP 登录 / 自签会话退役 | 公开注册、登录和手机号验证码登录由 Casdoor 承担；StuHelper auth 路由只注册 OIDC 登录、回调、session refresh/logout、当前用户和 native exchange，不暴露 `/auth/otp/*` 或 `/auth/phone/*` 本地 OTP 登录端点；auth handler 不再持有 SMS/OTP 登录依赖，不再提供 phone user upsert 或自签 phone token 签发；`/auth/refresh` 拒绝遗留自签 refresh token，认证中间件拒绝遗留自签 access cookie；路由表、refresh 和 middleware 测试锁定该契约。 |
| StuHelper auth refresh/logout session locator 硬化 | `/api/v1/auth/refresh` 和 `/api/v1/auth/logout` 会在 session 撤销或 refresh rotation 前拒绝重复、空白或逗号折叠的 `X-Stuhelper-Session-ID`，并拒绝该 native header 与浏览器 `session_id` cookie 同时出现；原生 refresh 的 JSON body 与浏览器认证/session cookie 也继续互斥，避免 token/source 混用造成撤销或轮换语义歧义。 |
| Open Platform 管理审核 UI baseline | 管理后台开放平台应用审核页已支持 app / scope 列表、scope 批准 / 驳回、redirect URI 变更审核、legacy Casdoor 导入表单、secret 轮换、暂停、恢复、吊销；审计事件页可检索全部 `open_platform_audit_events`。 |
| Open Platform disclosure 限流与基础审计硬化 | Disclosure / UserInfo 路径已按 app、app+user、endpoint 和 consent challenge 维度执行 Redis 滑动窗口限流；超限返回 429；Redis 或审计不可用时 fail-closed；成功、拒绝、超限、手机号投影失败写入 `open_platform_audit_events`，并记录低基数 Prometheus 指标。 |
| Open Platform legacy disclosure client / redirect 硬化 | Legacy disclosure API 的 OpenAPI 契约已显式要求 `client_id`；handler 优先使用请求 `client_id` 并保留认证上下文 appID fallback；显式 Bearer app identity 与请求 `client_id` 不一致时拒绝并审计 `client_mismatch`；已有 active consent 的 disclosure 仍必须重新校验 `redirect_uri` 精确匹配，失败审计 `redirect_uri_not_allowed`。 |
| Open Platform token 最小化静态门禁与 code-flow 探针工具 | 管理员批准 app 前会在 Casdoor 创建 / 更新 third-party application，强制 `token_format=JWT-Custom` 和显式空 `TokenFields` 并写入 `open_platform.app.token_probe.passed`；legacy 导入时出现非 `JWT-Custom` token format 或业务 token field 会写入 `open_platform.app.token_probe.failed` 并拒绝 approved；运维脚本 `infra/ops/casdoor-token-minimization-probe.sh` 可用真实 authorization-code flow 解码 ID token / JWT access token 并阻断业务 claim。 |
| Open Platform runtime token 探针证据门禁 | 管理员批准 app 前可强制执行 runtime code-flow token 探针；探针 evidence 写入 `open_platform_token_probe_evidence`，结果写入 `open_platform.app.token_probe.runtime.passed/failed` 审计；管理后台 `/open-platform/token-probe-evidence` 可按 app、审核人、结果和 client ID 检索 evidence；生产配置要求 `OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED=true` 且提供非占位符 command runner；发布脚本会用专用 `CASDOOR_TOKEN_PROBE_SMOKE_*` app 执行同款 runtime smoke，聚合 evidence 也会在子 smoke 前验证该强制门禁开启并写入 `.configuration.runtimeTokenProbeRequired=true`；2026-05-24 生产聚合 evidence 已通过并留档到 `/www/server/panel/data/compose/stuhelper/source/infra/generated/open-platform-production-evidence-20260524T1502.json`。 |
| 生产公网身份入口 smoke | `infra/ops/identity-public-smoke.sh` 会验证 `stuhelper.com` health、`id.stuhelper.com` OIDC discovery、OAuth authorization server metadata、JWKS、OAuth/UserInfo GET/POST 基础路由、`response_modes_supported=query`、未登录 authorize 跳转、`prompt=login&max_age=0` 重新认证跳转、token / introspect / revoke 路由级错误、GET/POST logout、POST logout URL query / JSON body 拒绝、UserInfo URL query / body token 来源拒绝、token/UserInfo/introspection/revoke no-store/no-cache 响应头、401 `WWW-Authenticate` challenge 和 `sso.stuhelper.com` discovery/JWKS；配置专用 approved smoke client 后还会验证注册客户端 `prompt=none` 的 `login_required` + `iss` 错误回调；同时配置 client secret 后会真实执行 `client_credentials` token 签发、introspection active、UserInfo 拒绝 app-only token、公开 Open Platform resource access API 对未授权随机资源返回 `fga_denied`，也可配置预授权 smoke 资源要求返回 `allowed`，最后验证 revoke 和 revoke 后 inactive；`infra/ops/bootstrap-identity-public-smoke-client.sh` 可在发布时用已存在的 owner/reviewer 用户幂等创建或修复专用 approved smoke client、写回 `IDENTITY_PUBLIC_SMOKE_CLIENT_ID` / redirect URI / client secret，`prod-deploy.sh` 会在 `IDENTITY_PUBLIC_SMOKE_BOOTSTRAP_ENABLED=true` 时先执行该 bootstrap 并重新加载 env；结果写入脱敏 `infra/generated/identity-public-smoke-evidence.json`，不记录 secret/token；`prod-deploy.sh` 在 app/frontend/admin 启动后、业务 smoke 前自动运行，防止 Baota Nginx / DNS / TLS / issuer 配置漂移漏过发布。 |
| 生产 Identity public smoke 专用客户端现场验证 | 2026-05-24 已在生产库准备 `identity-public-smoke` approved client、`resource.read` scope request 和 approved scope，并把 smoke client env 写入生产宝塔 Compose 源码目录；完整公网 smoke 返回 24 通过、0 失败，证据文件为 `/www/server/panel/data/compose/stuhelper/source/infra/generated/identity-public-smoke-production-full-20260524T074535Z.json`。 |
| Disclosure 运营报表与异常重放告警 | Disclosure / UserInfo 路径的 app、app+user、endpoint、consent 限流阈值已接入环境变量；同 app、用户、endpoint、结果和 scope 的重复拒绝会触发 `open_platform.disclosure.replay_detected` 审计与 `open_platform_disclosure_replay_total` 指标；Prometheus 规则覆盖异常重放、拒绝峰值和超限；管理后台 `/open-platform/disclosure-report` 可查看 summary、endpoint、拒绝原因、限流维度和最近 replay 事件。 |
| Open Platform 资源 API/UI v1.1 baseline | OpenFGA model 已包含 `open_platform_app`、`user_profile.can_read_by_app`、`resource_item.can_read_by_app`、`resource_item.can_write_by_app`；管理员 API 和管理后台可 grant / revoke / list app 到具体资源的 tuple，grant / revoke 均要求非空审计原因；第三方应用 API 可用 client credentials 检查具体资源访问；缺少 `resource.read` / `resource.write` scope 或缺少 tuple 时返回 `allowed=false`，OpenFGA / 审计不可用时 fail-closed；管理员吊销 app 时会删除该 app 的 `resource_item` / `user_profile` OpenFGA tuple，并以 `source=app_lifecycle` 写入 resource revoke 审计，其他 app tuple 不受影响。 |
| OpenFGA 资源授权 smoke 门禁 | `infra/ops/openfga-resource-access-smoke.sh` 和 `server/cmd/openfga-resource-smoke` 会对真实 OpenFGA store/model 写入 `open_platform_app -> resource_item` read/write tuple，验证 check/list-objects，撤销 read tuple 后再次验证 check=false 且 list-objects 不再返回资源，撤销 write tuple 后验证 check=false；`prod-deploy.sh` 已在 bootstrap 后、app 启动前运行该 smoke；2026-05-24 生产 `stuhelper-prod-backend` 网络内现场 smoke 已通过并留档到 `/www/server/panel/data/compose/stuhelper/source/infra/generated/openfga-resource-access-smoke-production-20260524T1439.json`，聚合 evidence 也已通过并留档到 `/www/server/panel/data/compose/stuhelper/source/infra/generated/open-platform-production-evidence-20260524T1502.json`。 |
| M13：audit 写入上下文 | `audit.LogContext(ctx, event)` 已落地；Gin、登录/登出、风控锁定、Casdoor admin/app provisioning、service account、MFA、评课等关键审计调用点已迁入上下文写入。持久化使用 `context.WithoutCancel(ctx)` 保留 trace baggage / request-scoped values，避免请求取消导致安全审计丢失；集成测试覆盖 canceled context 下仍写入 `request_id` / `trace_id`。 |
| L3：phone 日志脱敏 lint | `tools/semgrep/stuhelper-security.yml` 已定义项目自定义 Semgrep 规则，阻断 phone/mobile 类 zap 字段记录原始值；`scripts/check-semgrep-custom-rules.sh` 会先验证规则 fixture，再扫描 `server/internal`；GitLab CI `custom_sast` 已接入该门禁。 |
| M12：outbox dead-letter 显式状态 | `domain_event_outbox.status` 已加入 `dead_letter`；统一 worker 达到 `MaxAttempts` 后写入 `dead_letter`，不再使用 100 年后的 `available_at` 表达终止失败；`outbox.MarkJobFailure` / `outbox.RequeueDeadLetterJob` 提供通用 repository API，资源清理、用户外部同步和评课 FGA 同步 worker 均已接入；集成测试覆盖 dead-letter 不被 claim、upsert 重置和显式重放。 |

## 待立项候选

下列内容来自历史计划的未完成描述，但没有被采纳为当前执行计划：

| 候选项 | 来源 | 当前状态 |
|--------|------|----------|
| Koishi 群管中心高阶运营工作流：更细粒度报表、历史版本、更复杂处置编排 | `docs/internal/exec-plans/archived/2026-04-19-koishi-moderation-center-implementation.md` | 待产品确认，不作为活跃任务 |

## 审查延后项

下列内容来自 2026-05-09 代码审查复核。它们已确认有改进价值，但需要独立
选型、schema 设计、部署策略或监控指标设计，不混入普通修复批次。

| 项 | 范围 | 当前状态 | 立项前置 |
|----|------|----------|----------|
| H8：uniappx refresh token 安全存储 | `clients/uniappx/src/api/native-session.ts` | 已完成：原生 App 会话 token 只通过 `globalThis.stuhelperSecureStorage` native secure-storage bridge 读写；普通 `uni.storage` 不再保存 refresh token，仅用于清理旧版 `stuhelper:native-tokens` 遗留数据；bridge 缺失、读写失败或遗留清理失败均显式抛出 `NativeSessionStorageError` 并 fail-closed | 已由 uniappx native-session / auth / shared-client 单元测试和类型检查覆盖；打包前需提供对应 iOS Keychain / Android EncryptedSharedPreferences bridge 实现 |
| M15：生产内部 Postgres SSL | `docker-compose*.yml` + `infra/ops/*` + runbook | 已完成：生产 init / preflight / deploy 统一调用 `require_production_postgres_ssl`，强制 `POSTGRES_ENABLE_SSL=on`、内部连接最低 `verify-ca`、应用 `DB_SSL_MODE=verify-full`，并检查三个 PostgreSQL URL 均包含 CA 校验；prod compose overlay 不再默认 `sslmode=disable`；开发初始化仍显式保留 `POSTGRES_INTERNAL_SSL_MODE=disable` | 已由契约测试覆盖 |
| L4：DB metrics table label | `server/internal/pkg/db/` + `server/internal/pkg/metrics/` | 已完成：DB wrapper 通过显式 `db.WithTableHint(ctx, table)` 读取低基数 table metadata；未提供或非法 table 统一归一为 `unknown`；`query_retry` / `query_row_retry` 通过 operation 维度表达，table 维度不再写空串或 `retry`；核心 Repository 已接入显式 table hint，未从 SQL 字符串猜表名 | 已由 DB / metrics 单元测试和后端全量测试覆盖 |
| L6：cache prefix metrics label | `server/internal/pkg/cache/` + `server/internal/pkg/metrics/` | 已完成：cache hit/miss/duration 指标改为 `backend` + 白名单 `namespace` 标签；`cache.Helper` 构造时显式指定 `course` / `review` namespace，默认和非白名单值归一到 `generic`，不会从 raw cache key 动态生成 Prometheus label；cache invalidation failure 也收敛到 namespace 标签 | 已由 metrics/cache 单元测试覆盖 |
| L12：MinIO read-only rootfs | `docker-compose.yml` | 已完成：`minio` 和 `minio-init` 均启用 `read_only: true` 并挂载 `/tmp` tmpfs；`minio` 只保留 `/data` volume，`minio-init` 将 `MC_CONFIG_DIR` 指向 `/tmp/.mc`，避免 mc 配置写入只读根文件系统 | 已由 compose 契约测试和临时容器启动验证覆盖 |

## 归档原则

- `docs/internal/exec-plans/active/` 只保留当前要推进的计划。
- 已完成计划进入 `docs/internal/exec-plans/completed/`。
- 被后续 ADR、设计或实现取代的计划进入 `docs/internal/exec-plans/archived/`
  或保留在 `docs/internal/design-snapshots/` 中并标记为历史快照。
- Runbook、QA checklist、发布检查表不是项目开发计划，不在本文件中跟踪。
