# API 设计文档

## 概述

RESTful API 设计规范，基础路径：`/api/v1`

---

## 1. 认证接口

| 接口 | 方法 | 说明 |
|------|------|------|
| `/auth/ldap` | POST | LDAP 认证 |
| `/auth/sso` | POST | SSO 认证 |
| `/auth/refresh` | POST | 刷新 Token |
| `/auth/logout` | POST | 登出 |

---

## 2. 用户接口

| 接口 | 方法 | 说明 |
|------|------|------|
| `/user/profile` | GET | 获取用户信息 |
| `/user/bindQQ` | POST | 绑定 QQ |
| `/user/bindWechat` | POST | 绑定微信 |

---

## 3. 课程接口

| 接口 | 方法 | 说明 |
|------|------|------|
| `/courses` | GET | 课程列表 |
| `/courses/:id` | GET | 课程详情 |
| `/courses/:id/reviews` | GET | 课程评价 |
| `/courses/:id/reviews` | POST | 发布评价 |

---

## 4. 资料接口

| 接口 | 方法 | 说明 |
|------|------|------|
| `/resources` | GET | 资料列表 |
| `/resources/upload` | POST | 上传资料 |
| `/resources/:id` | GET | 下载资料 |

---

## 5. 响应格式

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```
