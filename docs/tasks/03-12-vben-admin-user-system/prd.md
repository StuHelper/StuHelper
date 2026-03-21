# Vben Admin + 双表 + Casdoor 定制 + 个人中心/学生认证

## Goal

为 StuHelper 后台管理系统完整引入 Vben Admin 框架，建立 Casdoor 用户 + StuHelper 业务的多表架构（实名认证、学生认证、RBAC 权限），定制 Casdoor SSO 登录页视觉风格，并实现带学生认证功能的个人中心页面。

## What I Already Know

### 现有状态（来自代码库调研）

- **管理后台**: 完全自建，7 个页面（Dashboard/Reports/Reviews/Teachers/SensitiveWords/Logs），Tailwind CSS + 手写组件
- **Vben 集成**: 仅 `@cimom/vben-effects-common-ui` 的 `CountTo` 组件在用
- **用户表**: 单表 `users`（id, external_id, username, email, avatar_url），`external_id` 关联 Casdoor
- **LDAP 模块**: 原型阶段，`server/internal/modules/ldap/client.go` 有 Login/QueryUserByUID，未接入主应用
- **用户中心**: 只读 Tab 页（评课/投票/收藏），无个人资料编辑、无认证状态
- **SSO**: 完整 OAuth2 流程，HttpOnly Cookie 会话，CSRF 保护，自动刷新
- **主页风格**: Glassmorphism 设计，主色 cyan (#06b6d4) / indigo (#4f46e5)
- **技术栈**: Vue 3.5 + Vite 6 + Tailwind v4 + Pinia + openapi-fetch

## Decisions

### D1: Vben Admin — 完整替换
- 管理后台完整迁移为 Vben Admin 脚手架项目结构
- 使用 Vben 的路由系统、权限控制、Layout 体系、主题配置、UI 组件
- 现有 7 个管理页面 + AdminLayout 需全部迁移/重写

### D2: 数据库架构 — 四层模型

```
Layer 1: Casdoor (外部)          → 管理认证账号、OAuth2 token
Layer 2: StuHelper users (核心)   → 平台身份，关联 Casdoor
Layer 3: 学籍数据库 (本地只读)    → 从 bj1 同步的学生/教职工数据
Layer 4: 业务扩展表 (StuHelper)   → 实名认证、学生认证、角色权限
```

#### 数据同步链路
```
北航 Oracle DB ──(每日凌晨 dump)──► bj1 (自定义视图，仅必要字段)
                                       ↓ (每日同步)
                                    StuHelper 本地学籍表 + Redis 缓存
```

#### 新增表

**user_identities — 实名认证**
```sql
user_identities (
  user_id          BIGINT REFERENCES users(id) UNIQUE,
  doc_type         VARCHAR(20) NOT NULL,     -- 'MAINLAND_ID' / 'HK_MACAU' / 'TW' / 'PASSPORT'
  doc_number_enc   BYTEA,                    -- 加密存储的证件号
  person_uid       VARCHAR(64) NOT NULL,     -- HMAC(doc_type + ':' + doc_number)，跨学籍唯一标识
  real_name        VARCHAR(100) NOT NULL,
  verified         BOOLEAN DEFAULT FALSE,
  verify_method    VARCHAR(20),              -- 'academic_db_match' / 'tencent_cloud' / 'manual'
  verified_at      TIMESTAMPTZ,
  -- 非大陆证件需上传照片
  doc_photo_front  TEXT,                     -- 证件正面照片存储路径/URL
  doc_photo_back   TEXT,                     -- 证件背面照片存储路径/URL (如适用)
  doc_photo_selfie TEXT,                     -- 手持证件自拍 (如适用)
  rejection_reason TEXT,                     -- 管理员拒绝原因
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
)
```

**user_profiles — 学生认证**
```sql
user_profiles (
  user_id            BIGINT REFERENCES users(id) UNIQUE,
  school_id          VARCHAR(10),            -- '10006' = 北航
  student_ids        JSONB,                  -- ['本科学号', '研究生学号']
  active_student_id  VARCHAR(50),            -- 当前有效学号
  verification_status VARCHAR(20) DEFAULT 'unverified', -- unverified/pending/verified/rejected
  verification_method VARCHAR(20),           -- 'ldap' / 'manual'
  phone              VARCHAR(20),
  phone_verified     BOOLEAN DEFAULT FALSE,
  consent_given_at   TIMESTAMPTZ,
  verified_at        TIMESTAMPTZ,
  created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
)
```

**school_configs — 学校认证配置**
```sql
school_configs (
  school_id            VARCHAR(10) PRIMARY KEY,
  school_name          VARCHAR(100) NOT NULL,
  verification_method  VARCHAR(20) NOT NULL,  -- 'ldap' / 'manual'
  ldap_config          JSONB,                 -- LDAP 连接配置（加密）
  academic_db_table    VARCHAR(100),           -- 本地学籍表名
  consent_text         TEXT,
  manual_form_fields   JSONB,                 -- 手动认证表单字段定义
  enabled              BOOLEAN DEFAULT TRUE,
  created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
)
```

**学籍本地表（从 bj1 同步）**
```sql
CREATE SCHEMA IF NOT EXISTS academic;
academic.buaa_students (xh, xm, sfzjlxdm, sfzjh, yxdm, zydm, bjdm,
                        xznj, rxnj, pyccdm, sjh, dzxx, xjztdm, synced_at)
```

#### 身份关联：升学场景
- `person_uid` = HMAC(doc_type + ':' + doc_number)
- 用 `person_uid` 与学籍表的 HMAC(sfzjlxdm + ':' + sfzjh) 匹配
- 一人多学籍（本科+研究生）全部关联，优选 `rxnj` 最新的记录

### D3: 实名认证流程

```
大陆身份证用户:
  输入姓名+身份证号 → 比对学籍表(XM+SFZJH)
  ├─ 匹配 → 自动通过 (verify_method='academic_db_match')
  ├─ 不匹配/学籍缺失 → 腾讯云身份核验 API
  │   ├─ 通过 → verify_method='tencent_cloud'
  │   └─ 失败 → 认证失败
  └─ 生成 person_uid

非大陆证件用户 (港澳台通行证/护照):
  输入姓名+证件类型+证件号 → 上传证件照片(正面+背面+手持自拍)
  → 比对学籍表(SFZJLXDM+SFZJH匹配)
  ├─ 匹配 → 自动通过
  └─ 不匹配/缺失 → 管理员后台手动审核
     ├─ 通过 → verify_method='manual'
     └─ 拒绝 → 填写拒绝原因，用户可重新提交
  └─ 生成 person_uid
```

### D4: 学生认证流程（实名认证之后）

```
选择学校 → 读取 school_configs
├─ 北航 (verification_method='ldap'):
│   展示风险同意书(consent_text) → 用户同意
│   → 输入学号+密码 → 后端 LDAP bind 验证
│   ├─ 成功 → verification_status='verified'
│   │   → 从学籍表读取姓名/院系/手机号
│   │   → 手机号比对: 一致→自动绑定 / 不一致→短信验证码
│   └─ 失败 → 提示错误
│
└─ 其他学校 (verification_method='manual'):
    填写详细表单(由 manual_form_fields 定义)
    → 上传学生证照片等证明材料
    → verification_status='pending'
    → 管理员后台审核
```

### D5: 权限系统 — 高级 RBAC + 用户组 + 个人分配

**RBAC 核心表**
```sql
roles (id, name, display_name, description, is_system)
permissions (
  id, name, module, action,
  scope_school_ids  JSONB DEFAULT NULL,  -- null=不限, ['10006']=仅北航
  scope_roles       JSONB DEFAULT NULL   -- null=不限
)
role_permissions (role_id, permission_id)
user_roles (user_id, role_id)
```

**用户组**
```sql
user_groups (id, name, display_name, description, created_by, created_at)
user_group_members (group_id, user_id)
user_group_permissions (group_id, permission_id)
```

**个人权限覆盖**
```sql
user_permissions (user_id, permission_id, granted)
-- granted=true 授予, granted=false 撤销（覆盖角色/组继承的权限）
```

**权限检查优先级**: 个人覆盖 > 用户组 > 角色 > 默认拒绝

**Scope 选择器**: 管理后台配置权限时，自动从 school_configs/roles 表读取已有数据作为选项

**访问控制规则（已确认）**:

| 功能 | 未登录 | 已登录未认证 | 已认证(非北航) | 北航认证学生 |
|------|--------|-------------|---------------|-------------|
| 查看评课 | 不可见 | 简略信息 | 简略信息 | 完整信息 |
| 发布评课 | 不可 | 不可 | 不可 | 可以 |
| 北航专属功能 | 不可 | 不可 | 不可 | 可以 |
| 特定功能 | 按权限配置 | 按权限配置 | 按权限配置 | 按权限配置 |

### D6: Casdoor UI 定制
- 在 Casdoor 控制台注入自定义 CSS/HTML/JS
- 视觉风格匹配 stuhelper.com 主页（Glassmorphism, cyan/indigo 配色）
- 代码存放在仓库中 `docs/casdoor/` 供参考和版本管理
- 配合前端全局 Loading 动画实现无缝过渡

### D7: 个人中心
- 右上角头像 + 下拉菜单（参考 Vben Admin）
- 个人中心页面：用户信息、实名认证状态、学生认证状态
- 可跳转到独立的认证页面
- 保留现有 Tab 功能（评课/投票/收藏）

## Open Questions

1. ~~Vben Admin 引入深度~~ → D1
2. ~~双表业务字段~~ → D2
3. ~~学生认证后的权限变化~~ → D5
4. ~~简略信息定义~~ → D8
5. ~~MVP 范围~~ → 全部纳入

## Requirements (evolving)

### 子任务 0: Vben Admin 完整替换
- 管理后台完整迁移为 Vben Admin 脚手架项目
- 现有 7 个管理页面全部迁移
- 新增：用户管理、权限管理、用户组管理、学校配置管理、实名/学生认证审核

### 子任务 1: 数据库架构
- 新增 user_identities / user_profiles / school_configs 表
- 新增 RBAC 表（roles / permissions / role_permissions / user_roles）
- 新增用户组表（user_groups / user_group_members / user_group_permissions）
- 新增个人权限覆盖（user_permissions）
- 新增学籍本地表 academic.buaa_students
- 学籍数据每日从 bj1 同步

### 子任务 2: Casdoor UI 定制
- 自定义 CSS 匹配 stuhelper.com 风格
- 代码存放仓库，附配置教程

### 子任务 3: 个人中心 + 认证
- 右上角头像+下拉菜单
- 个人中心页面（信息展示+认证状态）
- 实名认证流程（大陆身份证+非大陆证件）
- 学生认证流程（LDAP/手动）
- 手机号绑定（比对学籍+短信验证）

## Acceptance Criteria (evolving)

- [ ] 管理后台完整使用 Vben Admin 框架
- [ ] 数据库有 user_identities / user_profiles / school_configs / RBAC 全套表
- [ ] 北航学生可通过 LDAP 完成学生认证
- [ ] 非大陆证件用户可上传证件照片，管理员可审核
- [ ] 其他学校用户可手动填表+管理员审核
- [ ] 权限系统支持角色/用户组/个人覆盖三级
- [ ] 权限支持 school_id scope 限制
- [ ] 管理后台有权限配置 UI，scope 选择器自动聚合已有数据
- [ ] Casdoor 登录页匹配 stuhelper.com 视觉风格
- [ ] 个人中心展示认证状态，可跳转认证页面
- [ ] 右上角头像+下拉菜单，参考 Vben Admin
- [ ] 发布评课需要登录+北航学生认证
- [ ] 未登录不可查看评课，登录未认证只能看简略信息

## Definition of Done

- Tests added/updated (unit/integration where appropriate)
- Lint / typecheck / CI green
- Docs/notes updated if behavior changes
- Rollout/rollback considered if risky

### D8: 评课简略信息展示

**简略模式（未认证用户）**:
- 显示：评课标题 + 正文前 N 字符或前 N% 内容（取较小值）
- N 字符数和 N% 比例值可在管理后台配置修改
- 隐藏部分：保留实际内容长度占位，加高斯模糊效果 + 锁图标
- 提示文案："登录并完成学生认证后即可浏览全部信息"
- 点击模糊区域/提示 → 自动跳转登录（未登录）或学生认证（已登录未认证）

**完整模式（已认证用户）**:
- 显示全部字段（正文、评分细项等）

**配置项（管理后台可修改）**:
- `review_preview_chars`: 预览字符数（默认 20）
- `review_preview_percent`: 预览百分比（默认 20）
- 取两者较小值作为实际预览长度

## Out of Scope (explicit)

- 短信验证码接口接入（需要第三方 SMS 服务商，本次只预留接口）
- 腾讯云身份核验 API 接入（本次只预留接口，实际调用后续对接）
- bj1 同步脚本开发（本次只建本地学籍表结构，同步脚本由运维单独处理）
- 其他学校的 LDAP 配置（本次只实现北航，其他学校走手动审核通道）
- 移动端/小程序适配

## Technical Notes

### Key Files
- Admin layout: `clients/web/src/modules/admin/views/AdminLayout.vue`
- Auth store: `clients/web/src/stores/auth.ts`
- API client: `clients/web/src/api/client.ts`
- User table: `server/scripts/init.sql`
- LDAP client: `server/internal/modules/ldap/client.go`
- Homepage: `clients/web/src/modules/home/views/HomePage.vue`
- Router: `clients/web/src/router/index.ts`
- Design tokens: `clients/web/src/styles/main.css` (cyan #06b6d4, indigo #4f46e5)

### Constraints
- 保持现有 Tailwind v4 主题体系
- 保持现有 OpenAPI spec-first 开发流程
- 保持 HttpOnly Cookie 认证机制
- LDAP client 已有 Login/QueryUserByUID，可直接复用
- 证件照片需安全存储（加密/访问控制）
- 腾讯云身份核验 API 需要接入
