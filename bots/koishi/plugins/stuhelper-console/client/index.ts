import { Context } from '@koishijs/client'

import Page from './page.vue'
import './styles/base.css'
import './styles/surfaces.css'
import './styles/controls.css'
import 'virtual:uno.css'

export default (ctx: Context) => {
  ctx.page({
    name: 'StuHelper 群管中心',
    path: '/stuhelper',
    component: Page,
  })
}
