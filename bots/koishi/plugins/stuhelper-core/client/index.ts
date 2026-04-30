import { Context, icons } from '@koishijs/client'

import Index from './pages/index.vue'
import GroupIcon from './icons/group.vue'
import { icons as customIcons, Octicons } from './icons'
import './styles/tokens.css'
import './styles/primitives.css'
import './styles/dashboard-charts.css'
import './styles/shell.css'

const USED_OCTICONS = [
  'apps',
  'bar-chart',
  'chevron-down',
  'discussion',
  'gear',
  'graph',
  'log',
  'people',
  'person',
  'personadd',
  'sub',
  'three-bars',
  'tools',
  'warning',
  'x',
] as const

function registerOcticon(name: string) {
  if (!Octicons.paths[name]) {
    throw new Error(`Missing Octicon path: ${name}`)
  }

  icons.register(`stuhelperGroupCenter:octicons.${name}`, Octicons.create(name))
}

// 注册自定义图标
icons.register('stuhelperGroupCenter', GroupIcon)
icons.register('stuhelperGroupCenter:users', customIcons.users)
icons.register('stuhelperGroupCenter:shield', customIcons.shield)
icons.register('stuhelperGroupCenter:user', customIcons.user)

// 注册当前 UI 实际使用的 GitHub Octicons 图标。
for (const name of USED_OCTICONS) {
  registerOcticon(name)
}

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
