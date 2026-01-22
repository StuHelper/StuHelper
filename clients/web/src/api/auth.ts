import { request } from './index'

// 登录响应
export interface LoginResponse {
  url: string
  state: string
}

// 回调响应
export interface CallbackResponse {
  user: {
    id: string
    name: string
    display_name: string
    email: string
    avatar: string
  }
}

// 用户信息
export interface UserInfo {
  id: string
  name: string
  email: string
  display_name: string
  avatar?: string
}

// 获取登录 URL
export const getLoginURL = () => {
  return request.get<LoginResponse>('/auth/login')
}

// 获取注册 URL
export const getSignupURL = () => {
  return request.get<LoginResponse>('/auth/signup')
}

// 处理 OAuth 回调
export const handleCallback = (code: string, state: string) => {
  return request.get<CallbackResponse>('/auth/callback', {
    params: { code, state }
  })
}

// 获取当前用户信息
export const getCurrentUser = () => {
  return request.get<UserInfo>('/auth/me')
}

// 登出
export const logout = () => {
  return request.post('/auth/logout')
}
