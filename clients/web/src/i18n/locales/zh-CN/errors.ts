/**
 * 错误消息 - 中文
 * 与后端错误码对应，见 docs/api/error-codes.md
 */
export default {
  // 网络错误
  NETWORK_ERROR: '网络连接失败，请检查网络设置',
  TIMEOUT: '请求超时，请稍后重试',

  // 认证错误
  UNAUTHORIZED: '请先登录',
  TOKEN_EXPIRED: '登录已过期，请重新登录',
  INVALID_TOKEN: '登录信息无效，请重新登录',

  // 权限错误
  FORBIDDEN: '没有权限执行此操作',

  // 客户端错误
  BAD_REQUEST: '请求参数错误',
  NOT_FOUND: '请求的资源不存在',
  VALIDATION_ERROR: '数据验证失败',
  CONFLICT: '资源冲突',
  RATE_LIMIT_EXCEEDED: '请求过于频繁，请稍后重试',

  // 服务端错误
  SERVER_ERROR: '服务器错误，请稍后重试',
  SERVICE_UNAVAILABLE: '服务暂时不可用，请稍后重试',

  // 业务错误
  BUSINESS_ERROR: '操作失败',

  // 未知错误
  UNKNOWN: '发生未知错误',

  // 错误页面
  notFound: {
    title: '页面不存在',
    description: '你访问的页面可能已被移除或地址有误',
    backHome: '返回首页',
    goBack: '返回上页'
  },
  loadError: {
    title: '加载失败',
    description: '页面加载出现问题，请刷新重试',
    reload: '刷新页面'
  },

  // 认证回调
  authCallback: {
    loading: '正在登录中...',
    error: '登录失败',
    backToLogin: '返回登录',
    missingCode: '缺少授权码',
    missingState: '缺少 state 参数',
    loginFailed: '登录失败，请重试',
    orgMismatch: '当前 SSO 账户无权访问本系统',
    orgMismatchHint: '请注销当前账户后，使用正确的账户登录',
    ssoLogout: '注销并重新登录'
  }
}
