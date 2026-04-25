---
type: guide
audience: qa, frontend-dev
status: current
authoritative-source: this file
last-verified: 2026-04-19
---

# uniappx 原生认证 QA 清单

## 前置条件
- [ ] 后端已启动（docker compose up -d --wait）
- [ ] Zitadel OIDC 已配置且 well-known 可达
- [ ] uniappx 已配置 stuhelper:// scheme

## 登录流程
- [ ] 点击登录 → 调用 /api/v1/auth/login?platform=native → 获取授权 URL + state
- [ ] 打开系统浏览器跳转授权 URL → Zitadel 登录页正常显示
- [ ] 登录成功后 Zitadel 302 回 /api/v1/auth/callback?code=xxx&state=yyy
- [ ] 后端检测 native state → 302 重定向到 stuhelper://auth/callback?code=xxx&state=yyy
- [ ] App.vue 拦截 deep link → 跳转 /pages/auth/callback
- [ ] callback.vue 校验 state → 调用 authStore.exchangeNativeCode(code, state)
- [ ] exchangeNativeCode 调用 /api/v1/auth/exchange-native → 获取 accessToken + refreshToken
- [ ] token 持久化到 uni storage → 跳转首页

## Token 续期
- [ ] access token 过期后，App 尝试发起 /api/v1/auth/refresh（当前需确认：是否已注入 refresh_token）
- [ ] 续期成功 → 新 token 持久化（access/refresh）
- [ ] 续期失败 → 跳转登录页

## 登出
- [ ] 调用 /api/v1/auth/logout（Bearer token）→ 服务端撤销成功
- [ ] 清除本地 token → 跳转登录页
- [ ] 登出后旧 token 不可用（访问受保护端点返回 401）

## 异常场景
- [ ] state 不匹配 → callback.vue 显示错误并允许重试
- [ ] code_verifier 过期 → exchange-native 返回 400
- [ ] 网络中断 → 友好错误提示
- [ ] 冷启动通过 scheme 唤起 → 正确解析 deep link 参数
- [ ] 热启动（App 在后台）通过 scheme 唤起 → 正确处理

## 多设备
- [ ] 在另一设备登录 → 两设备独立会话
- [ ] 单设备登出 → 其他设备不受影响
- [ ] 全设备登出 → 所有设备 token 失效
