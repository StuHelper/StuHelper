/**
 * 通用文本 - 中文
 */
export default {
  actions: {
    confirm: '确认',
    cancel: '取消',
    save: '保存',
    saving: '保存中...',
    delete: '删除',
    deleting: '删除中...',
    edit: '编辑',
    search: '搜索',
    loading: '加载中...',
    loadMore: '加载更多',
    retry: '重试',
    submit: '提交',
    send: '发送',
    sending: '发送中...',
    reply: '回复',
    back: '返回',
    more: '更多',
    close: '关闭',
    clear: '清除',
    operationFailed: '操作失败，请重试',
    refresh: '刷新',
    learnMore: '了解更多'
  },
  status: {
    available: '可用',
    coming: '开发中',
    success: '成功',
    failed: '失败',
    pending: '处理中'
  },
  pagination: {
    total: '共 {total} 条',
    page: '第 {page} 页',
    pageSize: '每页 {size} 条',
    perPage: '每页',
    items: '条',
    pageLabel: '第 {page} 页',
    prevPage: '上一页',
    nextPage: '下一页',
    nav: '分页导航',
    pageSizeLabel: '每页显示条数'
  },
  empty: {
    data: '暂无数据',
    result: '未找到结果'
  },
  loadFailed: '加载失败',
  noMore: '没有更多了',
  time: {
    justNow: '刚刚',
    minutesAgo: '{n} 分钟前',
    hoursAgo: '{n} 小时前',
    daysAgo: '{n} 天前'
  },
  login: {
    title: 'StuHelper 统一登录',
    subtitle: '账号登录、认证与开放平台入口',
    identityLogin: '使用统一身份认证登录',
    signup: '注册账号',
    redirecting: '跳转中...',
    hint: '使用统一身份认证登录',
    loginFailed: '登录失败',
    signupFailed: '注册失败',
    networkError: '网络连接失败',
    loginUrlFailed: '获取登录链接失败',
    signupUrlFailed: '获取注册链接失败',
    invalidState: '无效的认证状态',
    callbackFailed: '认证回调处理失败',
    fetchUserFailed: '获取用户信息失败',
    phoneLogin: '验证码登录',
    phonePlaceholder: '请输入手机号',
    codePlaceholder: '请输入验证码',
    sendCode: '获取验证码',
    resendCode: '{n}s 后重新获取',
    verifyAndLogin: '验证并登录',
    verifying: '验证中...',
    otpSendFailed: '发送验证码失败',
    otpVerifyFailed: '验证码验证失败',
    otpSent: '验证码已发送',
    invalidPhone: '请输入正确的手机号',
    phoneHint: '使用手机号快捷登录',
    identityHint: '完成账号登录、学生认证与第三方应用授权'
  },
  openPlatformConsent: {
    loading: '正在加载 StuHelper Connect 授权请求...',
    loadFailed: '授权请求加载失败',
    invalidToken: '授权请求已失效或缺少 token，请返回账号中心或从发起应用重新开始授权',
    submitFailedTitle: '授权操作失败',
    submitFailed: '授权操作失败，请重试',
    connectEyebrow: 'StuHelper Connect',
    openIdentityHome: '返回账号中心',
    title: '{app} 请求通过 StuHelper Connect 访问你的 StuHelper 账户',
    identityLine: '当前 StuHelper 账号：{user}',
    appInfo: '接入应用',
    appName: '应用',
    description: '说明',
    redirect: '回调域名',
    links: '链接',
    homepage: '应用主页',
    privacy: '隐私政策',
    permissions: '将通过 Connect 披露以下信息',
    reason: '用途：{reason}',
    expiresAt: '有效至 {time}',
    accept: '允许',
    accepting: '正在允许...',
    deny: '拒绝',
    denying: '正在拒绝...',
    sensitivity: {
      low: '低敏感',
      medium: '中敏感',
      high: '高敏感',
      very_high: '极高敏感'
    }
  },
  openPlatformProfileCompletion: {
    loading: '正在检查 StuHelper 账户资料...',
    loadFailed: '资料补全请求加载失败',
    invalidToken: '资料补全请求已失效或缺少 token，请返回账号中心或从发起应用重新开始授权',
    submitFailedTitle: '继续授权失败',
    submitFailed: '继续授权失败，请重试',
    connectEyebrow: 'StuHelper Connect',
    openIdentityHome: '返回账号中心',
    title: '继续登录 {app} 前需要补全 StuHelper 账号资料',
    identityLine: '当前 StuHelper 账号：{user}',
    requiredFields: '需要补全',
    requestedPermissions: '应用请求通过 Connect 获取的信息',
    reason: '用途：{reason}',
    openAction: '去补全',
    continue: '我已补全，继续',
    continuing: '正在继续...',
    refresh: '重新检查',
    noMissingFields: '你的 StuHelper 账号资料已满足本次授权请求',
    expiresAt: '有效至 {time}'
  },
  meta: {
    description: 'StuHelper 评课社区 - 真实的课程评价与教学反馈平台，帮助学生选课决策',
    ogTitle: '评课社区 - StuHelper'
  },
  search: {
    placeholder: '搜索...',
    label: '搜索',
    clear: '清除搜索'
  },
  infoPages: {
    actions: {
      home: '返回首页',
      learningCenter: '进入学习中心'
    },
    about: {
      eyebrow: '关于',
      title: '关于 StuHelper',
      intro: 'StuHelper 是面向校园学习场景的课程评价与信息协作平台，目标是让选课、了解教师和获取学习信息更直接。',
      sections: {
        currentCapabilities: {
          heading: '当前能力',
          body: '现阶段已上线课程评价、教师详情、通知中心与基础审核后台，其他模块会按优先级逐步开放。'
        },
        contentPrinciples: {
          heading: '内容原则',
          body: '平台鼓励基于真实体验的理性表达，强调可验证、可追溯和对他人有帮助的信息。'
        },
        roadmap: {
          heading: '开发方向',
          body: '项目采用 OpenAPI 3 Spec-First、多端共享 API 客户端和 StuHelper 统一登录，后续会继续完善教学相关模块。'
        }
      }
    },
    privacy: {
      eyebrow: '隐私',
      title: '隐私政策',
      intro: 'StuHelper 仅收集维持服务运行所必需的数据，并尽量避免直接处理可识别个人身份的信息。',
      sections: {
        collection: {
          heading: '我们收集什么',
          body: '登录状态、必要的访问日志、课程互动产生的数据，以及用于安全校验的 Cookie。'
        },
        usage: {
          heading: '我们如何使用',
          body: '用于身份验证、风险控制、内容审核、功能统计，以及改进课程评价体验。'
        },
        control: {
          heading: '你的控制权',
          body: '你可以随时退出登录、删除或修改自己发布的内容；如需进一步的数据协助，请联系维护者。'
        }
      }
    },
    terms: {
      eyebrow: '条款',
      title: '服务条款',
      intro: '使用 StuHelper 即表示你同意遵守校园社区的基本规则，不发布违法、骚扰、虚假或恶意内容。',
      sections: {
        responsibility: {
          heading: '内容责任',
          body: '你需要对自己发布的测评、回复和举报信息负责，平台有权对违规内容进行隐藏、删除或限制。'
        },
        access: {
          heading: '账户与访问',
          body: '登录能力依赖统一身份认证系统，异常访问、批量抓取或绕过安全机制的行为会被限制。'
        },
        changes: {
          heading: '服务变更',
          body: '平台会持续迭代功能与规则，重要调整会通过站内公告或文档说明同步。'
        }
      }
    }
  },
  locale: {
    switchToEnglish: '切换到英文',
    switchToChinese: '切换到中文',
    englishShort: 'EN',
    chineseShort: '中'
  },
  bootstrap: {
    failed: '应用启动失败，请刷新页面后重试。',
    contactSupport: '如果问题持续存在，请联系维护者并附上控制台错误信息。'
  },
  loading: {
    label: '加载中'
  }
}
