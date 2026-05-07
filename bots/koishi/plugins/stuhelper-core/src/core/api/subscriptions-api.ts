import type { Subscription } from '../../types'
import { error, success } from './api-response'
import {
  assertSubscriptionScope,
  filterSubscriptions,
  findScopedSubscriptionRawIndex,
  type WebSocketAPIContext,
} from './websocket-api-context'

export function registerSubscriptionsAPI(api: WebSocketAPIContext) {
  const { service, data, addAuthorityListener, resolveConsoleScope } = api

  addAuthorityListener('stuhelperGroupCenter/subscriptions/list', async function (params?: { fetchNames?: boolean }) {
    const scope = await resolveConsoleScope(this)
    const subsData = data.subscriptions.get('list') || []
    const scopedSubs = filterSubscriptions(subsData, scope)
    if (!params?.fetchNames) {
      return success(scopedSubs.map(sub => ({ ...sub, name: '', avatar: '' })))
    }
    return success(enrichSubscriptions(scopedSubs, service.cache.getCachedData()))
  })

  addAuthorityListener('stuhelperGroupCenter/subscriptions/add', async function (params: { subscription: Subscription }) {
    try {
      const scope = await resolveConsoleScope(this)
      assertSubscriptionScope(scope, params.subscription)
      const list = data.subscriptions.get('list') || []
      list.push(params.subscription)
      data.subscriptions.set('list', list)
      await data.subscriptions.flush()
      return success({ success: true })
    } catch (e) {
      return error(e instanceof Error ? e.message : '添加订阅失败')
    }
  })

  addAuthorityListener('stuhelperGroupCenter/subscriptions/remove', async function (params: { index: number }) {
    try {
      const scope = await resolveConsoleScope(this)
      const list = data.subscriptions.get('list') || []
      const rawIndex = findScopedSubscriptionRawIndex(list, scope, params.index)
      if (rawIndex < 0) return error('订阅不存在')
      list.splice(rawIndex, 1)
      data.subscriptions.set('list', list)
      await data.subscriptions.flush()
      return success({ success: true })
    } catch (e) {
      return error(e instanceof Error ? e.message : '移除订阅失败')
    }
  })

  addAuthorityListener('stuhelperGroupCenter/subscriptions/update', async function (params: { index: number, subscription: Subscription }) {
    try {
      const scope = await resolveConsoleScope(this)
      assertSubscriptionScope(scope, params.subscription)
      const list = data.subscriptions.get('list') || []
      const rawIndex = findScopedSubscriptionRawIndex(list, scope, params.index)
      if (rawIndex < 0) return error('订阅不存在')
      list[rawIndex] = params.subscription
      data.subscriptions.set('list', list)
      await data.subscriptions.flush()
      return success({ success: true })
    } catch (e) {
      return error(e instanceof Error ? e.message : '更新订阅失败')
    }
  })
}

function enrichSubscriptions(subscriptions: Subscription[], cacheData: any) {
  return subscriptions.map((sub) => {
    if (sub.type === 'group') {
      const cached = cacheData.guilds[sub.id]
      return { ...sub, name: cached?.name || '', avatar: cached?.avatar || guildAvatar(sub.id) }
    }
    const cached = cacheData.users[sub.id]
    return { ...sub, name: cached?.name || '', avatar: cached?.avatar || qqAvatar(sub.id) }
  })
}

function qqAvatar(userId: string) {
  return `https://q1.qlogo.cn/g?b=qq&nk=${userId}&s=640`
}

function guildAvatar(guildId: string) {
  return `https://p.qlogo.cn/gh/${guildId}/${guildId}/640/`
}
