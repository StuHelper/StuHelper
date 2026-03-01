/**
 * 认证相关 API
 */
import { request } from './index'

// 登录响应
export interface LoginResponse {
  url: string
  state: string
}

// 用户信息
export interface UserInfo {
  id: string
  name: string
  email: string
  displayName: string
  avatar?: string
  isAdmin?: boolean
}

// 回调响应
export interface CallbackResponse {
  user: UserInfo
  expiresIn: number // Access Token TTL（秒）
}

// 登出响应
export interface LogoutResponse {
  message: string
  ssoLogoutURL?: string
}

// 获取登录 URL
export function getLoginURL() {
  return request.get<LoginResponse>('/auth/login')
}

// 获取注册 URL
export function getSignupURL() {
  return request.get<LoginResponse>('/auth/signup')
}

// 处理 OAuth 回调
export function handleCallback(code: string, state: string) {
  return request.get<CallbackResponse>('/auth/callback', {
    params: { code, state }
  })
}

// 获取当前用户信息
export function getCurrentUser() {
  return request.get<UserInfo>('/auth/me')
}

// 登出
export function logout() {
  return request.post<LogoutResponse>('/auth/logout')
}

// 登出所有设备
export function logoutAll() {
  return request.post<{ message: string }>('/auth/logout-all')
}
