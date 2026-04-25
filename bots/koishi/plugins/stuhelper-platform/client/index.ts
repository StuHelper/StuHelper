import type { Context } from '@koishijs/client'

import Page from './page.vue'
import './styles.css'
import 'virtual:uno.css'

const PAGE_PATH = '/stuhelper'

export default function apply(ctx: Context) {
  ctx.page({
    name: 'StuHelper 平台',
    path: PAGE_PATH,
    component: Page,
  })
}
