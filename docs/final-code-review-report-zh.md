# StuHelper 综合 Code Review 报告

> 来源整合：
> 1. `.trellis/workspace/207/reports/continuous-review-summary-2026-03-18.md`
> 2. `.trellis/workspace/207/reports/review-validation-pool/master-validation-report-zh.md`
> 3. `review-validation-pool/raw/` 下全部已落盘验证报告（截至 `validation-0061`）
> 4. MR22 专项审查（codex）
> 5. 人工复审 + 代码验证（2026-03-21）
>
> 本报告已于 2026-03-21 根据实际代码状态全面更新，移除已修复项，保留仍需处理的问题。

---

## 一、执行摘要

StuHelper 仓库已具备以下工程能力：

- 后端中间件、认证、Cookie + CSRF 会话模型较成熟；
- OpenAPI 3.1 -> generated types -> shared typed client 主链完整运转；
- 模块边界整体可辨识；
- RBAC 后端建模完整，admin 主入口已切换到 `RequireAnyPermission`；
- manual student verification、identity review、config 管理等敏感域已完成闭环。

经过 2026-03-21 的集中整改，此前报告中的高风险 defect 已全部修复：

- `/auth/me` 外部依赖抖动不再 503（三环节独立降级，最小权限原则）；
- OAuth callback 不再出现半成功状态（buildUserInfo 前置于 cookie 写入）；
- 用户同步不再静默覆盖已有邮箱（COALESCE 保护）；
- 举报 action 契约已统一（spec/runtime/frontend 对齐）；
- 启动失败不再返回退出码 0；
- 私钥文件已从 git 索引移除，.gitignore 已覆盖。

**当前仓库剩余的问题不再是 defect，而是架构完整度和文档治理类工作。**

---

## 二、已修复问题记录

以下问题在 `feature/fix-rbac-and-verification-issues` 分支中完成修复。

### 2.1 私钥文件治理

| 状态 | 描述 |
|------|------|
| **代码侧已完成** | `.gitignore` 添加 `server/certs/*.key` / `*.pem`，`git rm --cached` 移除索引 |
| **待人工完成** | 旧密钥应视为已泄露，需手动轮换并改为环境变量 / secret manager 注入 |

### 2.2 /auth/me 外部依赖降级（原 MR22-M1）

- `handler_userinfo.go` `buildUserInfo` 三环节独立降级：
  - Casdoor 缓存失败 → warn + 用 JWT token fallback
  - UpsertUser 失败 → warn + 跳过
  - 能力查询失败 → warn + 返回空权限集（最小权限原则）
- `/auth/me` 不再因外部依赖抖动 503 踢用户。

### 2.3 Callback 半成功修复（原 MR22-M2）

- `handler_login.go` 重排顺序：`buildUserInfo → trackLoginTokens → setTokenCookies`
- 任何环节失败都是纯失败，不会出现"浏览器有 session 但回调报失败"的半成功状态。

### 2.4 OAuth callback state 参数修正

- `GetOAuthToken` 第三参数从 `h.appName` 改为传真实 `state`。
- casdoor-go-sdk v1.44.0 当前不消费此参数，但传真实值可防止未来 SDK 升级破坏。

### 2.5 user_sync email COALESCE 保护（原 MR22-M3）

- `user_sync.go`: `email = COALESCE(EXCLUDED.email, users.email)`
- 上游 email 为空时不再覆盖本地已有邮箱，与 `avatar_url` 保护逻辑一致。

### 2.6 举报 action 契约统一（原 Confirmed-Current #4）

- runtime binding 从 `reject/hide_review/delete_review` 统一为 spec 定义的 `reject/hide/delete`
- `service_report.go` switch cases 同步更新
- docs、bundled spec、generated types 全部对齐

### 2.7 启动失败返回语义修复（原 Confirmed-Current #5）

- `main.go`: 保存 `serverStartErr`，shutdown 后 `return serverStartErr`
- 启动失败时进程退出码正确反映错误状态

### 2.8 OpenAPI LDAP 文案残留修复（原 Confirmed-Current #9）

- `user-identity.yaml` summary 从 "提交学生认证（LDAP）" 改为 "提交学生认证"
- 已重新 bundle spec 和 regenerate TypeScript types

### 2.9 此前已修复的项（非本轮，但经代码验证确认）

| 原编号 | 问题 | 验证结果 |
|--------|------|----------|
| #2 | Admin RequireAdmin 粗粒度 | `main.go:313` 已改为 `RequireAnyPermission`，43 个路由挂 `RequirePermission` |
| #3 | Identity review status drift | 前端已使用 `pending/verified/rejected`，与 backend OpenAPI 一致 |
| #6 | Admin typed client 旁路 | `admin/src/api/index.ts` 仅用 shared typed client，无 raw wrapper |
| #8 | Admin 路由门禁粗粒度 | `stores/auth.ts` 消费 `globalCapabilities`，`router/index.ts` 用 `hasRouteAccess` |
| #11 | Student verification reject reason | DB 有列、repository 读写、handler 返回、model 有字段，闭环完整 |

---

## 三、仍需处理的问题

### 3.1 Admin 侧 RBAC 前后端覆盖不对称

**性质**：功能完整度，非 defect。

**当前状态**：
- 后端已有完整 RBAC CRUD：roles、permissions、user roles/permissions、user groups/members/permissions
- 前端当前落地的管理页面主要是角色管理

**建议**：
1. 明确哪些后端 RBAC 能力计划前端覆盖
2. 未落地的能力在文档中标注"后端已就绪 / 前端未覆盖"
3. 属于 `03-12-vben-admin-user-system` 任务范围

### 3.2 多客户端 session boundary 文档化

**性质**：架构设计选择，非安全漏洞。

**当前状态**：
- Web/Admin：Cookie + CSRF + refresh
- uni-app x：本地 Bearer token + Authorization header
- 服务契约层允许 cookieAuth 与 bearerAuth 并存

**建议**：
1. 在架构文档中明确分端描述认证模型
2. 移动端用 Bearer token 是合理选择，但需文档说明边界
3. 决定是长期保留双模型还是逐步收敛

### 3.3 私钥轮换（人工操作）

**性质**：运维操作，非代码问题。

**待办**：
1. 生成新密钥对
2. 更新到部署环境（env / secret manager）
3. 删除本地 `server/certs/` 下的旧文件
4. 考虑添加 secret scanning / pre-commit 防护

---

## 四、Historical Fixed：历史问题，已确认修复

以下问题在此前的开发中已修复，不再作为现存缺陷。

| 原结论 | 修复证据 |
|--------|----------|
| School config 全量覆盖更新 | service 已使用可选字段 merge-update |
| Public profile 暴露 manualFormData | `UserProfile` 无该字段，仅 `AdminStudentVerificationItem` 有 |
| Contract source 严重分裂 | OpenAPI 3.1 -> generated TS -> shared typed client 主链完整 |
| Manual student verification 未对齐 | 请求契约、handler、service、持久化、admin review 全链已闭环 |

---

## 五、Needs Revalidation 已结项

原报告中 6 项 needs-revalidation，经复核全部结项：

| 原编号 | 问题 | 结论 |
|--------|------|------|
| N1 | Standalone admin 损坏 | 已验证正常运行，路由/登录/stores 完整 |
| N2 | Logs 页面缺失 | 路由和组件均存在 |
| N3 | OAuth callback state 参数 | 已修复（传真实 state） |
| N4 | Health/metrics 漂移 | spec `/health/live` + `/health/ready` 与 runtime 一致 |
| N5 | Worktree 可见性 | 工具链问题，非代码问题 |
| N6 | DB init sync | `init.sql` 包含所有必要列和 ALTER TABLE 升级语句 |

---

## 六、最终结论

经过集中整改，报告中所有 confirmed defect 已全部修复。当前仓库：

- **无阻塞性 defect**
- **无 contract drift**（spec/runtime/frontend 对齐）
- **无半成功状态**（callback 顺序已修正）
- **无静默错误吞没**（startup 返回值、/auth/me 降级日志）

剩余工作为功能完整度（RBAC 前端页面补全）和架构文档化（多端 session 模型描述），属于正常迭代范畴，不构成 code review 阻塞项。

---

## 七、交付说明

本报告可作为：
1. Code review 整改完成确认
2. 后续功能迭代的事实基线
3. 架构治理路线图的输入

剩余待办已收录在第三节，可直接进入排期。
