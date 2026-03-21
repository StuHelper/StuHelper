# 架构文档

`architecture/` 用来描述系统骨架和跨模块边界。这里优先写当前仓库真实在跑的结构，以及已经明确标成后续计划的大项。

## 文档索引

| 文档                                                                               | 说明                                       | 状态       |
| ---------------------------------------------------------------------------------- | ------------------------------------------ | ---------- |
| [product.md](product.md)                                                           | 产品形态、主要业务面和系统拓扑             | 现行       |
| [iam-architecture-design.md](iam-architecture-design.md)                           | IAM 架构设计：Zitadel + OpenFGA 迁移方案   | **新增**   |
| [openfga-model.fga](openfga-model.fga)                                             | OpenFGA 授权模型定义（可直接导入）         | **新增**   |
| [ecosystem-identity-and-authorization.md](ecosystem-identity-and-authorization.md) | 身份层、应用授权和访问控制边界（待迁移重写）| 待更新     |
| [frontend-architecture.md](frontend-architecture.md)                               | 前端 Monorepo、双后台入口和共享契约        | 现行       |
| [layered.md](layered.md)                                                           | 后端分层架构和文件拆分约定                 | 现行       |
| [follow-up-roadmap.md](follow-up-roadmap.md)                                       | 适合后续分支处理的大项计划                 | 现行       |

## IAM 迁移计划概览

当前正在进行从 Casdoor + 自建 RBAC 到 Zitadel + OpenFGA 的架构迁移：

- **身份层**：Casdoor → Zitadel（OIDC 认证、原生多租户、标准 Go OIDC 库）
- **角色管理**：6 张 DB 表 → Zitadel Project Roles + Go 静态映射（零 DB 查询）
- **资源授权**：应用 SQL 查询 → OpenFGA 关系型授权（Google Zanzibar 模型）
- **认证复用**：认证结果存 Zitadel 元数据，生态内所有应用可复用

详细方案见 [iam-architecture-design.md](iam-architecture-design.md)，授权模型见 [openfga-model.fga](openfga-model.fga)。
