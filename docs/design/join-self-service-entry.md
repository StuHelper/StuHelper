---
type: design
audience: product, frontend-dev, backend-dev, ops, maintainers
status: deprecated
authoritative-source: docs/product-specs/student-verification-and-group-admission.md + server/api/openapi.yaml + current source
last-verified: 2026-08-05
---

# Join 无验证码自助入口设计

> **已被替代。** 本文保留为 2026-06-08 的历史交互草案。当前 `/start` 只提供登录、当前账号条件
> 检查和前往独立学生认证/QQ 绑定页面的安全 continuation；不得在 JOIN 页面内复制学校认证流程。
> 文中的 `linked`、`verified`、`pendingReview`、旧 profile/student/freshman API 和内嵌表单不再是
> 当前契约。现行边界见[学生认证与群聊入群准入系统](../product-specs/student-verification-and-group-admission.md)。

## 背景

`join.stuhelper.com/verify/<code>` 是带 admission token 的入群认证入口。它能证明“这次浏览器流程对应某个 QQ 入群会话”，因此可以在登录后把当前 StuHelper 用户、token 中的 QQ 号和入群会话绑定起来。

现在还需要一个无验证码入口：用户没有 `verify/<code>` 链接时，也能在 `join.stuhelper.com` 上登录 StuHelper、完成学生认证、生成 QQ 绑定码，并通过机器人私聊完成 QQ 绑定。这个入口的体验应等价于“打开主站账号中心后进入学生认证和 QQ 绑定”，但浏览器地址不离开 `join.stuhelper.com`，除 Casdoor SSO 登录域名外不要求用户手动打开主站。

## 推荐路由

规范入口：

```text
https://join.stuhelper.com/start
```

路由名建议：

```text
join-self-service-start
```

页面标题建议：

```text
入群准备
```

选择 `/start` 的原因：

- 和 `/verify/<code>` 明确区分，用户不会误以为这是一个 token 链接。
- 足够短，适合写进 QQ 群公告、加群问题或机器人提示。
- 不占用 `/`。`join.stuhelper.com/` 继续返回 404，避免把 join 域变成主站别名。
- 不使用 `/bind`，因为页面不只做 QQ 绑定，还包含学生认证。

## 职责边界

`/verify/<code>` 是“已产生入群会话后的认证闭环”：

- 校验 admission token。
- 登录后消费 token。
- 将当前用户与 token 中的 QQ 入群会话绑定。
- 可推动该 admission session 进入 `linked`、`verified`、`pendingReview` 等状态。

`/start` 是“无验证码的入群前准备 / 自助补齐账号条件”：

- 不接收、不消费 admission token。
- 不信任 query 中的 QQ、群号、学校或来源参数。
- 只完成账号级事实：当前用户已登录、学生认证已通过、QQ 已绑定。
- QQ 绑定仍通过现有“网页生成绑定码 -> 私聊机器人发送 `绑定 <code>`”完成。
- 没有 token 时不能声明“当前入群会话已通过”或“会立即解禁某个群成员”。

如果用户已经被某个群禁言，并且管理员希望直接操作那条入群会话，仍应优先使用 `/verify/<code>` 或让管理员重发认证链接。`/start` 完成的是后续识别和自动放行所需的账号条件。

## 用户流程

1. 用户访问 `https://join.stuhelper.com/start`。
2. 页面在 join 域内尝试刷新当前登录态。
3. 未登录时展示登录和注册按钮。点击后调用 `/api/v1/auth/login` 或 `/api/v1/auth/signup`，redirect 使用当前完整 URL：`https://join.stuhelper.com/start`。
4. 用户在 `sso.stuhelper.com` 完成 Casdoor 登录。
5. OIDC 回调回到 StuHelper API 后，浏览器回到 `https://join.stuhelper.com/start`。
6. 页面展示一个紧凑流程：
   - 登录状态；
   - 学生认证；
   - QQ 绑定；
   - 完成状态。
7. 学生未认证时，在当前页面内展示学生认证表单。认证成功后停留在 `/start` 并刷新状态。
8. QQ 未绑定时，在当前页面内生成绑定码，展示机器人入口、绑定命令和复制按钮，并轮询当前用户 QQ 绑定状态。
9. 学生认证和 QQ 绑定都完成后，页面显示“入群准备已完成”，展示已绑定 QQ 和学生认证状态。

## 页面状态

页面不应复用主站 AppShell、账号中心导航或返回 `/identity` 的按钮。建议实现为 join 专用轻量页面，视觉上接近 `/verify/<code>`，但文案避免 admission token 语义。

| 状态 | 展示 |
|------|------|
| `loading` | 正在检查登录和认证状态 |
| `anonymous` | 登录 / 注册按钮；说明登录后回到当前页面 |
| `studentRequired` | 嵌入学生认证流程 |
| `qqRequired` | 嵌入 QQ 绑定码生成与轮询流程 |
| `complete` | 学生认证和 QQ 绑定均完成 |
| `error` | 状态加载失败，提供重试 |

`studentRequired` 和 `qqRequired` 可以按顺序展示，也可以展示为 checklist。推荐默认顺序是先学生认证、再 QQ 绑定，因为绑定成功消息可以直接告诉用户“当前账号已完成学生认证，加入受控群时会自动放行”。

## 前端实现方案

新增路由：

```ts
{
  path: "/start",
  name: "join-self-service-start",
  component: lazyLoad(
    () => import("@/modules/admission/views/JoinStartPage.vue"),
  ),
  meta: {
    title: "入群准备",
    layout: "none",
  },
}
```

不要把该路由标成 `requiresAuth`。原因是现有全局登录守卫会把未登录用户导向 `/login?redirect=/start`，而通用 LoginPage 会优先把相对 redirect 解析到主站 origin。`/start` 应在页面内部自管登录：

- `onMounted` 调 `auth.bootstrapSession({ force: true })`。
- 未登录时显示按钮。
- 登录按钮调用 `auth.login(window.location.href)`。
- 注册按钮调用 `auth.signup(window.location.href)`。

需要抽出可复用 UI，而不是直接嵌入现有页面级组件：

- 从 `StudentVerificationPage.vue` 抽出 `StudentVerificationPanel.vue`，提供 `embedded` / `showBack` / `redirectAfterVerified` 之类的控制。
- 从 `QQBindingPage.vue` 抽出 `QQBindingPanel.vue`，提供 `embedded` / `showBack` 控制，并保留现有机器人入口缺失提示、绑定码生成、复制和轮询逻辑。
- 主站页面继续使用这些 panel 并保留自己的 header / 返回按钮。
- `JoinStartPage.vue` 只组合两个 panel 和流程状态，不出现主站账号中心导航。

路由隔离需要扩展 `clients/web/src/router/join-domain.ts`：

- `isJoinAdmissionPath` 继续只识别 `/verify/<code>` 和移动拍摄路径。
- 新增 `isJoinSelfServicePath`，只识别 `/start` 和 `/start/`。
- `shouldBlockJoinHostRoute` 对 admission path 和 self-service path 都放行。
- 新增或扩展“join-only path outside join host”判断，使 `stuhelper.com/start` 返回 404。

## 后端与 API

第一版不需要新增后端 API，复用现有接口：

- `GET /api/v1/auth/login`
- `GET /api/v1/auth/signup`
- `GET /api/v1/auth/me`
- `GET /api/v1/user/profile`
- `POST /api/v1/user/profile/verify`
- `GET /api/v1/user/schools`
- `GET /api/v1/user/qq-binding`
- `POST /api/v1/user/qq-binding/code`
- `POST /api/v1/bot/qq-binding/consume`

生产配置已经要求：

- `CORS_ORIGINS` 包含 `https://join.stuhelper.com`。
- `TOKEN_COOKIE_DOMAIN=.stuhelper.com`，登录回调后浏览器会话可用于 join 域。
- join 域 `/api/*` 反代到后端。

本地 `join.localhost` 场景下，如果没有共享 cookie domain，现有登录 URL 生成逻辑会在 absolute redirect 指向允许的 join origin 时使用该 origin 的 `/api/v1/auth/callback`，以保证 OIDC state cookie 能在回调域名上校验。

## Ingress 变更

保持 join 根路径 404，只放行明确的 SPA 入口：

```nginx
location = /start {
    proxy_pass http://stuhelper_web;
}

location = /start/ {
    proxy_pass http://stuhelper_web;
}
```

仍保留：

- `join.stuhelper.com/verify/<code>` -> Web SPA
- `join.stuhelper.com/admission/freshman/camera/<token>` -> Web SPA
- `join.stuhelper.com/api/*` -> Backend
- `join.stuhelper.com/assets/*` -> Web assets
- `join.stuhelper.com/` 和其他主站业务路径 -> 404
- `stuhelper.com/verify/*` 和 `stuhelper.com/start` -> 404

需要同步更新：

- `infra/nginx/baota-stuhelper.conf`
- `infra/nginx/prod-parity-local-ingress.conf`
- `infra/ops/install-local-prod-parity-ingress.sh`
- `infra/ops/nginx-public-ingress-preflight.sh`
- `infra/ops/admission-public-smoke.sh`
- `infra/ops/tests/prod-parity-contract.sh`

## 安全约束

- `/start` 不接受可改变业务判断的 query 参数。允许后续添加 `source=qq_group_notice` 这类纯埋点参数，但不得参与授权、绑定或认证决策。
- 不允许 `redirect` query 从 `/start` 透传到登录流程。登录 redirect 固定为当前 canonical URL。
- 页面不展示“已通过当前群认证”“已解禁”等 admission session 结果。
- QQ 号只能来自机器人消费绑定码时的 `session.userId`，不能由网页输入。
- 绑定码仍一次性、短 TTL、服务端哈希存储。
- 学生认证写接口继续使用登录态、CSRF、后端频控和原有审核/认证规则。

## 验收点

- `join.stuhelper.com/start` 和 `join.localhost/start` 渲染 join 自助入口。
- `join.stuhelper.com/start` 未登录时点击登录，SSO 后回到同一 URL。
- 登录后未学生认证用户可以在 join 域内完成学生认证。
- 登录后未绑定 QQ 用户可以在 join 域内生成绑定码，机器人私聊 `绑定 <code>` 后页面轮询到已绑定。
- 已完成学生认证和 QQ 绑定的用户看到完成态。
- `join.stuhelper.com/`、`join.stuhelper.com/courses`、`join.stuhelper.com/user/student-verification` 仍返回 404。
- `stuhelper.com/start` 返回 404。
- 现有 `join.stuhelper.com/verify/<code>` 流程不变。

## 可选后续增强

如果产品希望“已经入群但丢失验证码的人”在 `/start` 完成 QQ 绑定和学生认证后也能自动解禁，需要单独增加后端会话归并设计。候选方案是在 QQ 绑定码消费成功后，按 `qq_id` 查找该 QQ 的 active `joined_muted` admission sessions，在冲突规则满足时把 session user 关联到绑定用户；若用户已有有效学生身份，再推进到 `verified` 并出队解禁动作。

这个增强会改变 admission session 状态机，必须新增服务层事务、审计和测试。它不应混进第一版 `/start` 页面，否则无 token 页面会隐式获得 admission session 绑定能力，边界会变得不清晰。
