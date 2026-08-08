---
type: internal
audience: maintainers, backend-dev, frontend-dev, qa, ops
status: current
authoritative-source: docs/product-specs/student-verification-and-group-admission.md + current repository state
last-verified: 2026-08-05
scope:
  - independent student verification domain
  - versioned academic roster snapshots
  - school verification adapters and campus connector contracts
  - phone credential and Casdoor projection orchestration
  - manual review and camera handoff reuse
  - admission eligibility consumption and Koishi durable actions
  - Web and Admin verification experiences
---

# 学生认证平台全面实施计划

## 目标

将现有分散在 `user`、`admission`、学籍导入脚本、Web 账号中心、Admin 和 Koishi 中的认证能力，
一次性切换为以下相互解耦的目标域：

1. 学生认证申请、尝试、学籍主体、凭据和实时资格；
2. Casdoor 手机号唯一写真源、本地只读投影与独立手机号凭据；
3. QQ 平台账号绑定；
4. 只消费资格结果的群聊 admission；
5. 只执行受控内网操作的校园连接器；
6. 版本化、原子激活且可回滚的本地学籍快照。

目标产品行为和验收 ID 以
[`docs/product-specs/student-verification-and-group-admission.md`](../../../product-specs/student-verification-and-group-admission.md)
为准；API、schema 和运行时仍分别以 OpenAPI、migration、源码与测试为真源。

## 已确认的实施边界

- 不修改已有编号 migration；结构演进从 `000026` 起新增 migration。结构 migration 提供 down；
  `000032` 是经产品 owner 明确授权的不可恢复清理，down 必须拒绝伪造恢复。
- OpenAPI 先于 Handler、shared client 和页面修改。
- 产品 owner 已明确放弃旧学生认证数据的备份、迁移、保留和兼容。使用单向 migration 清除旧
  `user_profiles` 学生字段、`user_identities`、旧 credential、新生/人工材料和旧名册；用户账号、QQ
  绑定、Casdoor/手机号状态、群策略与普通业务数据不在清理范围内。
- 本次 migration 不创建专用备份、导出或恢复摘要。但现有 `infra/ops/prod-deploy.sh` 会在执行 migration
  前创建并上传逻辑整库备份和物理基准备份；这些恢复介质可能同时包含旧认证事实与非目标域数据，
  不能靠行级 migration 选择性擦除。生产执行前必须单独批准恢复介质处置及清理后恢复锚点，脚本
  未完成相应受控流程前不得原样用于 `000032` 切换。
- Web API、Admin、Koishi 不持有 Oracle 凭据；未确认校园连接器生产节点前，只交付默认关闭的
  连接器契约、签名快照摄取和本地沙箱适配器。
- LDAP、Oracle、真实邮件、短信、Casdoor 和 QQ 的本地测试不能替代真实平台验收；未执行项必须
  明确标记为 `unverified` 或 `auth-blocked`。
- 不实现“实际未调用腾讯云实名 provider，却向用户声明已调用”的文案。普通界面使用中性的
  “实名信息校验”，隐藏 Oracle、表名和字段；隐私告知与审计必须与真实处理路径一致。
- 学号、姓名、证件号和学校手机号的新快照结构使用可逆密文与确定性 HMAC 成对存储；旧
  `academic.buaa_students` 在切换 migration 中直接清空，不作为迁移源、兼容源或回退真源。

## 已落地的仓库基线

- `studentverification` 已成为独立领域，申请、尝试、学籍主体、凭据、资格和 revision 均有独立
  Repository、Service、Handler 与 OpenAPI 契约。
- 学校方法、学校适配器、隐私告知、连接器操作、快照版本、质量门禁和 active pointer 已进入
  PostgreSQL 权威模型；在线请求不读取 Oracle。
- 手机号由用户当次手工输入，Casdoor 是唯一写真源；本地只保存受保护的完整值投影、HMAC、凭据和
  可对账 pending operation。学校号码一致时免短信，不一致时不泄露原因并回退短信。
- 人工审核、私有材料、能力驱动摄像头与跨设备 handoff 已从 admission 解耦并复用于所有学校。
- admission 只消费学生资格 decision + revision，Koishi 只消费 durable action；旧新生转发和旧认证
  写入口已经从活动路由、共享契约和插件注册中移除。
- `000032` 定义无备份、无映射、无双读的旧认证事实清理；旧私有对象只进入瞬时物理删除队列。
- `000032` 在清理前使用排他锁收敛旧事务，并永久拒绝旧 profile/identity/freshman/
  upsert 名册和无目标申请 ID 的凭据写入，防止旧容器在切换窗口回写已删事实。

## 剩余部署与真实平台验收缺口

1. 生产校园连接器节点、mTLS 证书、签名/加密密钥、吊销和轮换尚未部署演练。
2. 用户明确指定的既有 Oracle 账号、批准查询对象、字段/状态语义、传输加密和初次完整快照尚未由数据 owner 验收；不得为此创建、更换、授权或修改账号。
3. 北航 LDAP 的 LDAPS/StartTLS、证书链、目录属性、账号状态与学生主体 gate 尚未真实验收。
4. 邮件出站/入站签名、腾讯云短信、Casdoor 手机号更新后回读和对账尚未在真实平台验收。
5. QQ/NapCat 的真实加群事件、禁言、提醒、解禁、踢出和 ACK 尚未在目标群验收。
6. `000032` 尚未在生产等价副本执行；因此真实旧数据目前没有被本次开发过程清除。
   演练和生产切换还必须核对 Casdoor 中不存在历史 `verified_student` 业务角色成员；
   如确有存量，必须用 Casdoor 管理接口删除该角色成员关系并复核为零，不得删除用户或手机号。
   此外，现有生产部署脚本会在 migration 前创建逻辑与物理整库备份。删除含旧认证事实的既有
   备份/WAL 可能同时消除课程、内容等非目标域的灾难恢复锚点，超出本次行级清理可安全代办的范围；
   在产品 owner 与运维共同批准整份恢复介质处置和清理后新恢复锚点前，不得宣称“所有历史副本已
   清零”，也不得用现有脚本直接执行 `000032`。
7. Web/Admin/UniAppX H5 已通过 Desktop Chrome 与 Pixel 5 全量 E2E；学生认证页另有
   375/768/1024/1440 精确视口、44px 交互目标、横向溢出和减少动效浏览器契约。真实读屏器人工验收
   仍待执行。
8. 当前 Oracle 同步只实现语句一致的定期完整快照。源端尚未确认稳定游标和删除语义，因此没有启用
   增量同步，也不得自行扩大为 LogMiner/CDC 权限。

## 实施阶段

### 阶段 A：模型与干净切换基线

- [x] 新增版本化学校方法配置、适配器配置和健康状态表。
- [x] 新增 roster snapshot、record、active pointer、质量检查和摄取批次表。
- [x] 新增 enrollment subject、verification application、attempt 和 credential 扩展字段。
- [x] 新增 phone credential、pending operation 和 reconciliation 状态。
- [x] 新增最小 connector node/operation registry；secret 只保存 reference。
- [x] 为新表建立最小权限 grant、CHECK、唯一索引、有效状态部分索引和结构 down migration。
- [x] 增加 migration 静态契约测试与 PostgreSQL 集成测试。

### 阶段 B：OpenAPI 与后端领域服务

- [x] 先定义用户学校/方法、申请、实名信息、SSO、邮箱收发、人工审核、凭据与资格 API。
- [x] 定义手机号状态、创建操作、短信完成、换号和解绑 API。
- [x] 定义 Admin 学校方法、快照、材料、凭据、冲突和连接器健康 API。
- [x] 定义内部资格和手机号门槛 API；浏览器不能取得内部服务凭据。
- [x] 生成 Go/TypeScript 类型并通过 spec lint、重复生成幂等性和 route contract。
- [x] 实现 Handler → Service → Repository，所有 SQL 留在 Repository。
- [x] 资格由活动凭据和当前策略实时派生，revision 单调变化并通过 outbox 发布。

### 阶段 C：验证方法与依赖

- [x] 实现学校适配器注册表、北航学号/姓名/邮箱规则和声明式安全校验。
- [x] 实现本地实名信息 HMAC 比对、请求原文零持久化和统一不可枚举错误。
- [x] 将学校邮箱 OTP 收敛到独立申请；补入站邮件挑战接口与 provider 验证边界。
- [x] 实现 `buaa_ldap_bind` 连接器操作契约；默认因 TLS/授权 gate 关闭。
- [x] 复用并泛化人工表单、材料、摄像头和 handoff，去除对 admission 的身份依赖。
- [x] 实现完整快照暂存、质量门禁、原子激活、回滚、新鲜度和凭据重评估。

### 阶段 D：手机号与 Casdoor

- [x] 删除 LDAP 或学生认证成功时的自动绑号。
- [x] 本地保存完整号码的加密只读投影、脱敏值和 HMAC；Casdoor 仍是唯一写真源。
- [x] 用户始终手工输入手机号；北航名册匹配成功形成 `school_roster_phone_match`。
- [x] 名册未命中时静默回退腾讯云短信，形成 `sms_possession`。
- [x] 用 pending operation、Casdoor 更新、投影刷新、凭据激活和对账处理部分失败。
- [x] 内容发布通过独立手机号门槛契约，不从 `Phone` 非空或学生资格推断。

### 阶段 E：Web 与 Admin

- [x] 复用现有品牌 token 建设统一 Button/Input/Select/Textarea/FormField/Alert/Steps。
- [x] 新建独立、能力驱动的学生认证向导，支持独立入口和安全 continuation。
- [x] 实现实名信息、学校 SSO、邮箱收发、人工审核和恢复状态。
- [x] 实现独立手机号绑定、换号、解绑和发布意图恢复。
- [x] 泛化摄像头拍摄和二维码 handoff，不提供普通文件选择入口。
- [x] Admin 实现学校方法、适配器、快照、凭据/冲突、人工审核和连接器健康管理。
- [x] 完成键盘焦点、语义状态、错误恢复、Desktop Chrome/Pixel 5 回归，以及
  375/768/1024/1440 精确视口和减少动效自动化契约。
- [ ] 使用真实读屏器完成学生认证、手机绑定和人工材料 handoff 的人工验收。

### 阶段 F：Admission 与 Koishi

- [x] admission 改为只调用资格契约，不复制永久 `student_verified` 真源。
- [x] 迁移状态机并同步重建全部进行中状态的部分唯一索引。
- [x] 动作携带 eligibility revision；撤销或策略变化 supersede 未执行动作。
- [x] 保留 durable outbox、claim/lease/ACK、SSE 唤醒和周期扫描。
- [x] 收敛原始平台标识和重复连接日志，补低基数关联、重连采样和故障注入测试。

### 阶段 G：旧数据单向清理与切换

- [x] 定义只清理学生认证域、明确保护用户账号、QQ、手机号、群策略和普通业务数据的边界。
- [x] 新增不可恢复的数据库清理 migration，并使旧 release 动作失效、进行中会话回到 requirements。
- [x] 清理事务持有旧写表排他锁，提交后由数据库 trigger 拒绝旧容器回写，并要求新凭据关联目标申请。
- [x] 新增私有对象瞬时删除队列；对象删除成功后物理删除队列行，不保留恢复摘要。
- [x] 移除 profile fallback、旧 student/identity/freshman 写入口及 `legacy_migrated` / `tencent_cloud` 契约值。
- [ ] 在生产等价副本验证清理零剩余、对象最终删除、OpenFGA/机器人旧投影撤销、
  Casdoor 历史 `verified_student` 成员关系为零、备份/WAL 处置符合 `MIG-025`，以及非目标域数据
  不受影响。

## 验证门禁

- `make check-docs`
- `cd server && make lint-spec && make generate && make check-drift`
- `cd server && make fmt && make lint && make test && make build`
- `cd clients && pnpm type-check:all && pnpm lint:all && pnpm test:web`
- `cd clients && pnpm test:e2e:web && pnpm test:e2e:admin`
- `cd bots/koishi && corepack yarn test:ui` 以及受影响 workspace 单测
- migration up/down、并发唯一性、部分失败、outbox 重放和 revision 栅栏集成测试
- 真实 LDAP、Oracle、邮件、短信、Casdoor 与 QQ 验收单独记录，不由本地测试替代

### 2026-08-05 本地验证结果

- 后端 `make lint` 0 issue；`go test -race -p 1 ./...` 全量通过；OpenAPI lint、生成、构建及两个
  campus connector 命令构建通过。
- OpenAPI bundled spec、Go server 类型和 TypeScript client 类型在再次 `make generate` 前后哈希一致。
  `make check-drift` 相对当前 Git HEAD 仍会报告本次尚未提交的预期生成差异；这不表示工作树内的生成物过期。
- Web、Shared、UniAppX、Admin 单元/组件测试分别为 476、78、64、145 项通过；全量类型检查和
  lint 通过；Web、Admin、UniAppX 生产构建通过。
- Web 全量 Playwright：323 passed、1 个按视口设计 skipped、0 failed；新增精确视口/减少动效契约
  定向 2/2；Admin Playwright：144/144；UniAppX H5 Playwright：72/72；Koishi Console
  Playwright：46/46。Koishi build、UI
  contract、unit 与 startup smoke 通过。
- 上述均为本地或 stubbed 依赖验证，不替代本节列出的真实平台和生产等价迁移验收。

## 回滚原则

- 新凭据模型上线后立即成为唯一学生资格真源，不存在旧读 feature flag 或兼容投影。
- 旧数据清理不可恢复，down migration 必须拒绝伪造或还原旧资格。
- 执行 `000032` 后不得回滚到仍依赖旧认证表、旧状态或旧 API 的应用版本；数据库不提供旧事实恢复。
  发布故障只能回滚到兼容新 schema 的版本或向前修复。
- admission 与 Koishi 动作在任何切换阶段都保留幂等 action ID、attempt 和 durable outbox。
