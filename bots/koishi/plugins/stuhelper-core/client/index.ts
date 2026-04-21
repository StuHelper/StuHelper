import { Context, icons } from '@koishijs/client'

import Index from './pages/index.vue'
import GroupIcon from './icons/group.vue'
import LogoIcon from './icons/logo.vue'
import { icons as customIcons, Octicons } from './icons'
import './styles/console-pages.css'

// 注册自定义图标
icons.register('stuhelperGroupCenter', GroupIcon)
icons.register('stuhelperGroupCenter:logo', LogoIcon)
icons.register('stuhelperGroupCenter:dashboard', customIcons.dashboard)
icons.register('stuhelperGroupCenter:config', customIcons.config)
icons.register('stuhelperGroupCenter:warn', customIcons.warn)
icons.register('stuhelperGroupCenter:blacklist', customIcons.blacklist)
icons.register('stuhelperGroupCenter:log', customIcons.log)
icons.register('stuhelperGroupCenter:subscription', customIcons.subscription)
icons.register('stuhelperGroupCenter:settings', customIcons.settings)
icons.register('stuhelperGroupCenter:chat', customIcons.chat)
icons.register('stuhelperGroupCenter:npm', customIcons.npm)
icons.register('stuhelperGroupCenter:box', customIcons.box)
icons.register('stuhelperGroupCenter:activity', customIcons.activity)
icons.register('stuhelperGroupCenter:git-branch', customIcons.gitBranch)
icons.register('stuhelperGroupCenter:roles', GroupIcon)
// stat 组件常用图标
icons.register('stuhelperGroupCenter:users', customIcons.users)
icons.register('stuhelperGroupCenter:ban', customIcons.ban)
icons.register('stuhelperGroupCenter:bell', customIcons.bell)
icons.register('stuhelperGroupCenter:alert-triangle', customIcons.alertTriangle)
icons.register('stuhelperGroupCenter:user-x', customIcons.userX)
icons.register('stuhelperGroupCenter:user-minus', customIcons.userMinus)
icons.register('stuhelperGroupCenter:shield', customIcons.shield)
icons.register('stuhelperGroupCenter:shield-alert', customIcons.shieldAlert)
icons.register('stuhelperGroupCenter:rss', customIcons.rss)
icons.register('stuhelperGroupCenter:user', customIcons.user)
icons.register('stuhelperGroupCenter:bar-chart-2', customIcons.barChart2)
icons.register('stuhelperGroupCenter:trending-up', customIcons.trendingUp)
icons.register('stuhelperGroupCenter:clock', customIcons.clock)

// 注册所有 GitHub Octicons 图标
// 使用方式: <k-icon name="stuhelperGroupCenter:octicons.tag" />
for (const [name, component] of Object.entries(Octicons.getAll())) {
  icons.register(`stuhelperGroupCenter:octicons.${name}`, component)
}

// 为了兼容，mark 现在指向 octicons.tag
icons.register('stuhelperGroupCenter:mark', Octicons.create('tag'))

export default (ctx: Context) => {
  ctx.page({
    name: 'StuHelper 群管中心',
    path: '/stuhelper',
    icon: 'stuhelperGroupCenter',
    component: Index,
    order: 500,
    authority: 4, // 设置默认权限等级为 4
  })
}
