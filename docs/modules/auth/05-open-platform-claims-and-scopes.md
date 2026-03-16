# 开放平台的 Claims 与 Scopes 设计

这份文档定义第三方应用接入 StuHelper SSO 时，哪些用户事实可以被开放，哪些不能开放，以及应该通过什么 scope 获取。

## 设计目标

开放平台要同时满足两件事：

1. 第三方应用能拿到足够完成业务的用户事实
2. StuHelper 不泄露高敏感个人信息

因此默认策略是：

- **开放状态，不开放原始敏感值**
- **开放最小必要字段，不开放整份用户档案**

## 推荐 scope

| Scope | 说明 | 返回字段 |
| --- | --- | --- |
| `openid` | OIDC 基础身份 | `sub` |
| `profile.basic` | 基础公开资料 | `sub`、`username`、`avatar`（如适用） |
| `stuhelper.verification.identity_status` | 实名认证状态 | `identityVerified` |
| `stuhelper.verification.student_status` | 学生认证状态 | `studentVerified` |
| `stuhelper.education.actor_type` | 身份类型 | `actorType` |
| `stuhelper.education.school` | 学校归属 | `schoolID` |

## 默认不开放的字段

除非有单独的合规审批和更高等级授权，不对第三方应用开放：

- 真实姓名
- 学号
- 手机号
- 身份证号
- 证件照片
- 学生认证提交材料
- 实名认证原始材料

## 推荐返回模型

第三方应用最终应看到类似这样的最小事实：

```json
{
  "sub": "user_123",
  "identityVerified": true,
  "studentVerified": true,
  "actorType": "student",
  "schoolID": "10006"
}
```

## 不推荐的做法

### 不要把所有信息都塞进 Access Token

更合理的做法是：

- token 里放必要的 scope 和最小基础 claims
- 具体身份事实通过受 scope 控制的 userinfo / profile API 获取

这样有两个好处：

1. token 体积更可控
2. 字段变更和权限回收更容易管理

### 不要把“是否能拿到认证状态”做成全局默认

认证状态属于受限身份事实。  
第三方应用必须：

- 显式申请 scope
- 通过平台审核
- 明确用途

## 推荐的审核要求

第三方应用申请敏感身份事实时，至少要说明：

1. 为什么需要这些字段
2. 用于哪个功能
3. 是否可用更低敏感度字段替代
4. 数据会保存多久
5. 是否提供用户删除或解绑入口

## 与航小伴的关系

开放平台 scope 设计解决的是“第三方应用能知道哪些用户事实”，  
它不负责航小伴自己的业务授权。

例如：

- 第三方应用知道 `studentVerified=true`
- 不代表它自动拥有航小伴的评课发布权限

航小伴的权限仍然由航小伴后端自行判断。
