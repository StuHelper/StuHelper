export interface UniappxFeatureLink {
  title: string
  desc?: string
  icon?: string
  path: string
}

export const UNIAPPX_EXPERIMENTAL_NOTICE = 'uniappx 当前为实验性脚手架，登录与部分业务能力尚未开放。'

export const HOME_FEATURES: UniappxFeatureLink[] = [
  { title: '课程查询', desc: '快速查找课程信息（实验性）', icon: '📚', path: '/pages/course/index' },
  { title: '评课广场', desc: '查看课程评价（实验性）', icon: '⭐', path: '/pages/review/index' },
  { title: '个人中心', desc: '查看个人相关页面（实验性）', icon: '👤', path: '/pages/user/index' }
]

export const USER_MENU_ITEMS: UniappxFeatureLink[] = [
  { title: '消息通知', icon: '🔔', path: '/pages/user/notifications' }
]
