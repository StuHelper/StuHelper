# StuHelper 校园连接器

校园连接器是在学校批准的校园网络节点上运行的独立应用层中继。它只执行已经登记的高层操作：

- 校验一次学校统一身份认证账号；
- 读取一次完整学籍快照并加密上传。

它不是 VPN、通用反向代理、SOCKS、任意 TCP 转发器、任意 SQL 执行器，也不负责判断学生资格、签发
学生凭据或处理 QQ 入群会话。生产 StuHelper 不需要获得校园网段路由，也不持有学校 LDAP 或 Oracle
凭据。

## 1. 连接与信任模型

```text
学校 LDAP / Oracle
        ↑ 仅批准的主机、端口、属性和固定查询
校园连接器节点
        ↓ 节点主动出站；TLS 1.3 mTLS + Ed25519 请求签名
StuHelper campus-connector gateway
        ↓ 校验节点、operation、revision、时限、nonce 和签名
学生认证 / 版本化学籍快照
```

- 节点只主动连接 `centralBaseURL`，中心不获得通用校园网入口。
- 节点证书用于 mTLS；独立 Ed25519 密钥用于请求和快照签名，避免把传输身份与业务签名混为一个
  密钥。
- 学籍快照在校园侧使用临时 X25519 密钥、HKDF-SHA-256 和 AES-GCM 加密，再由 Ed25519 签名；中心
  验证后才可进入暂存快照和质量门禁。
- LDAP 默认只允许 LDAPS 或 StartTLS。北航 owner 已确认目标仅开放 RFC1918 IPv4 的 389 端口时，允许为
  单个 operation 显式启用 `ldap_plain_private_network` 例外：必须固定私网 IP 和 389、设置
  `allowPlaintextPrivateNetwork=true`，不得配置域名、其他端口、代理、CA/TLS 名称或静默降级。该例外只
  约束校园节点到学校 LDAP 的最后一跳；节点到中心仍使用 TLS 1.3 mTLS。学校提供可验证的 TLS 端点后
  必须迁回 LDAPS/StartTLS。
- Oracle 默认使用 `oracle_tls`（TCPS）并验证 CA 与证书名。现有北航接线仅提供经 SSH 跳板访问的
  Oracle 1521 时，允许单个 operation 显式使用 `oracle_ssh_tunnel`：SSH 隧道由节点运维层预先建立，
  连接器只允许拨号到一个固定 loopback 高端口，并要求 `allowPlaintextSSHTunnel=true`、空 Oracle CA/TLS
  名称以及仅含该 loopback endpoint 的 `allowedDialTargets`。连接器不会执行 SSH、读取私钥或创建任意
  TCP 隧道，也不允许 Oracle listener redirect 到其他目标；学校提供 TCPS 后必须迁回 `oracle_tls`。
- 连接器禁止代理环境变量、HTTP 重定向和未登记目标。
- 交互密码不写磁盘、Redis、持久队列、日志或 Trace；连接器以有界可擦除字节缓冲处理，完成、取消
  或超时后清零。

## 2. 两份配置的职责

| 文件 | 所在位置 | 内容 | 是否包含秘密 |
|---|---|---|---|
| `registry-manifest.example.json` | StuHelper 控制面 | 节点公开身份、证书指纹来源、签名公钥、学校、operation、固定目标和限额 | 否 |
| `node-config.example.json` | 校园节点 | 与控制面相同的 operation、字段映射、secret reference 和本地证书路径 | 不直接包含口令 |
| 节点 secret 文件/环境 | 校园节点 | mTLS 私钥、签名私钥、快照接收公钥、LDAP 服务账号口令、用户指定的既有 Oracle 账号口令 | 是，不得提交仓库 |

registry manifest 是完整集合：应用新 manifest 时，旧 manifest 中存在而新 manifest 遗漏的 operation
会被禁用，不能靠遗漏字段静默保留。新增或轮换操作先保持 `validationStatus: pending`、`enabled: false`，
完成目标、证书、权限、字段映射和健康检查后，再以新 revision 显式启用。

示例中的域名、学校代码、schema、view、证书和密钥 ID 都是占位符，不能直接用于生产。不要把学校
服务账号、个人密码、Oracle DSN、私钥或真实学生标识写入 JSON、Git、聊天记录或工单。

## 3. Oracle 操作边界

能用 Navicat 或 DBeaver 登录 Oracle **只证明账号与当前网络路径大概率可用**，不证明固定业务查询所需
对象和字段均可读取。连接器只能使用用户明确指定的既有账号；StuHelper、部署脚本和代理不得创建、申请、
更换、授权、回收或修改 Oracle 账号，也不得建议为了本项目另建所谓“专用只读”或“更小权限”账号。
既有账号权限较宽时只记录风险，不以此为由操作 Oracle；应用侧仍强制只执行仓库中固定的 `SELECT`。

上线前只能用以下只读动作验证接线，不得通过尝试写入、DDL、PL/SQL、权限变更或读取未批准对象来做
“负向权限测试”：

1. 建立批准的登录会话，并确认实际 session username 与 `expectedUsername` 一致；
2. 执行连接器代码中固定的完整快照 `SELECT`，确认批准对象、必需列和可读行满足映射契约；
3. 通过只读数据字典查询记录现有权限风险，不创建对象、不改变会话、不调整账号或 grants；
4. 固定获批传输模式、目标主机、端口、service name、owner/object、字段映射和最大行数；TCPS 模式还
   必须固定 TLS 服务名和 CA，SSH 隧道例外则必须固定跳板机 host key、loopback 高端口和隧道远端。

TCPS 是默认且优先的 Oracle 传输。北航现有路径使用 SSH 跳板时，SSH 客户端/sidecar 必须由校园节点
运维层管理：私钥只放节点 secret store，`known_hosts` 必须固定预先审核的 host key，转发目标必须是
批准的 Oracle 地址与 1521，禁止 `GatewayPorts`、动态转发、SOCKS、代理环境变量和远端/本地任意端口。
连接器自身不解析 SSH 配置，只消费本机已经建立的单一 loopback 高端口。不得把一个公网可访问的 1521
转发端口填写为 `oracle_tls`，也不得因 SSH 已加密就声称 Oracle 原生 TCPS 已启用。

接口不接受外部 SQL、筛选条件、表名或字段名。任何需要 `INSERT`、`UPDATE`、`DELETE`、`MERGE`、
`FOR UPDATE`、DDL、PL/SQL、事务控制、`ALTER SESSION`、存储过程、作业、Flashback、LogMiner、CDC 或
账号/权限变更的验证与同步方案都超出当前边界，必须停止，不能以“探测权限”为由尝试执行。

## 4. 完整快照与增量同步

当前实现选择**周期性完整快照**作为第一条生产路径，默认示例每 7 天执行一次；失败使用有界退避重试。管理员也可以从 StuHelper 管理端创建受审计、可查询状态的手动完整同步任务。每次同步使用一条
Oracle `SELECT`，获得语句级读一致性；快照带源开始/截止时间、行数、schema/mapping 版本、明文摘要、
签名和端到端加密。中心先写入独立暂存版本，质量门禁通过后才原子激活。新完整快照没有的记录不会
继续出现在当前查询中，因此不存在旧 upsert 导入器的“毕业/退学记录永久残留”问题。

当前连接器**尚未实现 Oracle 增量读取**，且在“登录 + 固定 `SELECT`”边界下不得启用 Flashback、
LogMiner、CDC 或数据库日志读取。未来只有在不扩展当前 Oracle 操作边界、仍能由固定 `SELECT` 完成，
并同时满足以下数据契约时，才可以另行评审增量设计：

- 有稳定且永不复用的记录主键；
- 有由数据 owner 保证单调、精度充足的变更游标（例如可信更新时间或 SCN）；
- 能表达删除、失效、学号更换和同一主体多段学籍，而不只是新增/更新；
- 能定义游标边界、事务一致性、迟到事件、重复投递和断点恢复；
- 仍定期执行完整快照对账，以发现漏读、游标回退和删除遗漏。

因此，当前生产路径固定为完整快照。不得申请或变更权限来适配增量，也不得通过 `GXSJ > last_time`
之类条件自行推断增量正确性。

## 5. 部署顺序

1. 在学校批准的稳定节点准备最小化容器运行环境；个人开发机只能用于开发验证，不能作为生产网关。
2. 生成并校验独立 PKI。默认输出受 `.gitignore` 保护，已有目录只校验、绝不覆盖：

   ```bash
   CAMPUS_CONNECTOR_GATEWAY_PUBLIC_HOST=connector.stuhelper.com \
     ./infra/ops/generate-campus-connector-pki.sh

   ./infra/ops/generate-campus-connector-pki.sh \
     --check \
     --gateway-host connector.stuhelper.com
   ```

   `gateway/` 只安装在中心，`node/` 只安装在校园节点，`registry/` 只含注册所需公开材料；节点登记完成
   后把 `authority/` 的两把 CA 私钥转移到离线 secret store。生成器使用标准 PEM/PKCS#8，并限制私钥
   为 `0600`；运行时仍兼容早期 Base64 裸密钥，但新部署不应继续生成自定义格式。
3. 复制示例文件为节点私有配置，替换 operation、学校代码、固定目标、适用的 TLS 名称、Oracle 对象、
   字段映射和 secret reference。示例保留默认 `oracle_tls`；若使用获批的北航 SSH 隧道，必须把 Oracle
   operation 改为 `upstreamProtocol: oracle_ssh_tunnel`、`targetHost: 127.0.0.1`、固定高端口、空
   `tlsServerName`/`caFile`、`allowPlaintextSSHTunnel: true`，并使 `allowedDialTargets` 恰好只有同一个
   loopback endpoint。北航明文 LDAP 例外必须同时满足上一节的固定私网 IP/389 和显式风险开关；不要
   修改程序以接受任意 SQL。
4. 用 registry 工具先做只读校验；审核通过后才使用 `--apply`。该工具需要访问 StuHelper PostgreSQL，
   manifest 只引用公开证书和公钥文件：

   ```bash
   go run ./cmd/campus-connector-registry \
     --manifest ../infra/campus-connector/registry-manifest.json \
     --reason "enroll approved campus connector"

   go run ./cmd/campus-connector-registry \
     --manifest ../infra/campus-connector/registry-manifest.json \
     --reason "enroll approved campus connector" \
     --apply
   ```

5. 在中心运行标准 `prod-deploy.sh`。首次发布时，migration 后会启动同一 backend image 的
   `campus-connector-bootstrap` 运行模式；它只提供 PostgreSQL/Redis/OTLP、mTLS Gateway 和 roster
   importer，不启动公网 API、OIDC、OpenFGA、token service 或普通业务 worker。然后启动校园连接器，
   确认中心收到心跳，LDAP/Oracle operation 分别显示稳定的健康码；不要用真实学生密码做周期探活。
6. 首次完整快照进入 `ready` 后检查行数、唯一性、状态码、密文/HMAC 成对生成、来源时间和校验和；
   自动激活默认关闭，首次上线应由授权管理员审阅并原子激活。
7. readiness 未通过时，标准发布会失败但保留 bootstrap 容器运行，不记录 release，也不启动新版
   App/Web/Admin。完成真实缺口后重新执行发布；门禁通过后脚本停止 bootstrap、确认 loopback 端口释放，
   再由固定为 `APP_RUNTIME_MODE=app` 的正式 App 接管 Gateway。已有健康 App Gateway 的普通升级会直接
   复用；映射但未监听或未知进程占用端口时 fail closed，不会自动停止或替换任何未知对象。
8. 演练连接器离线、证书吊销、签名错误、上游超时、快照突变和中心拒绝。相关学校方法必须 fail
   closed，但其他学校和不依赖该连接器的方法不受影响。

中心生产网关还需要一个不会终止 TLS 的公网 TCP 入口。仓库提供宝塔 Nginx stream 模板、严格渲染器、
应用脚本和有效配置预检，固定链路为：

```text
校园 Connector
  -> connector.stuhelper.com:9444
  -> Baota Nginx stream raw TCP passthrough
  -> 127.0.0.1:19444
  -> StuHelper Gateway TLS 1.3 mTLS
```

DNS 只把 `connector.stuhelper.com` 指向生产边缘，不会自动建立 TCP 监听、内网路由或代理。先获得稳定、
已批准的校园节点 IPv4 出口 CIDR，再配置生产 env；不得用个人开发机当前公网地址冒充长期节点地址，也
不得配置 `0.0.0.0/0`：

```env
CAMPUS_CONNECTOR_GATEWAY_ENABLED=true
CAMPUS_CONNECTOR_GATEWAY_PUBLIC_HOST=connector.stuhelper.com
CAMPUS_CONNECTOR_GATEWAY_PUBLIC_PORT=9444
CAMPUS_CONNECTOR_GATEWAY_EXTERNAL_PORT=19444
CAMPUS_CONNECTOR_ALLOWED_SOURCE_CIDRS=REPLACE_WITH_APPROVED_CAMPUS_CONNECTOR_SOURCE_CIDRS
NGINX_PUBLIC_INGRESS_PROFILE=app-all
```

应用 Connector stream 入口：

```bash
# 先 dry-run，不写文件。
CAMPUS_CONNECTOR_ALLOWED_SOURCE_CIDRS=<approved-ipv4-cidr> \
  ./infra/ops/apply-baota-nginx-templates.sh --profile connector

# 再安装到 /www/server/panel/vhost/nginx/tcp/connector.stuhelper.com.conf。
sudo \
  CAMPUS_CONNECTOR_ALLOWED_SOURCE_CIDRS=<approved-ipv4-cidr> \
  ./infra/ops/apply-baota-nginx-templates.sh \
    --profile connector \
    --apply \
    --reload \
    --preflight
```

脚本渲染精确 `allow` 列表和末尾 `deny all`，只允许 public port 与 loopback upstream 不同，安装后先
运行 `nginx -t`，失败时恢复旧目标或移除新目标。生产防火墙也必须只对同一批准 CIDR 开放 TCP 9444；
逐条执行 `sudo ufw allow proto tcp from <approved-ipv4-cidr> to any port 9444`，随后用
`sudo ufw status numbered` 核对不存在任意来源规则。Nginx 必须原样透传：普通 HTTP 反代、
`listen ... ssl`、`ssl_certificate`、`proxy_ssl*`、CDN 代签证书或边缘 TLS 终止都会丢失客户端证书身份，
不能使用。启用前运行 `prod-deploy.sh` 预检；它会同时审计 Nginx 生效配置、allowlist、端口关系、SAN、
证书链、证书/私钥配对、有效期、文件权限和 snapshot key ID。

该出站应用层连接已经解决生产中心与校园节点之间的传输问题，生产中心不需要北航路由。EasyTier 或
Tailscale 最多只能作为可选底层链路，当前公网 mTLS 方案不依赖它们；即使将来采用 overlay，也不能替代
mTLS、Ed25519 签名、operation allowlist、重放保护、快照加密、固定 Oracle `SELECT` 或审计/质量门禁。

## 6. 运行与轮换

- `docker-compose.example.yml` 展示只读根文件系统、无 Linux capabilities、`no-new-privileges`、有界
  PID 和临时目录的最小容器基线。
- 节点、operation、上游和中心只记录稳定错误分类与不可逆请求引用；不得记录学号、姓名、证件号、
  手机号、学校密码、Oracle 行或材料。
- mTLS 证书、Ed25519 签名密钥、X25519 快照接收密钥和上游只读口令分别轮换。先登记新公钥/证书并
  完成健康验证，再撤销旧项；紧急吊销时该节点立即停止服务，不能降级为无证书或无签名连接。
- 心跳过期、版本不兼容、节点被吊销、operation 未验证、目标指纹不一致或熔断开启时，中心拒绝新
  请求。恢复后仍须依赖 revision 和幂等键，不能重放过期交互密码或覆盖已激活快照。
- 管理端手动完整同步先写入 PostgreSQL，再由节点轮询领取；同校同 operation 只有一个进行中任务，
  每次领取有 3 分钟租约、最多 5 次、总期限 24 小时。任务只引用批准的 operation，管理员原因只留在
  中心审计，不会下发节点。
