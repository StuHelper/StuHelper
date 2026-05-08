import type { Subscription } from '../../types'
import type { WebSocketAPIContext } from './api-context'
import { error, success } from './api-response'
import {
  assertSubscriptionScope,
  filterSubscriptions,
  findScopedSubscriptionRawIndex,
} from './scope-filters'

export function registerSubscriptionsAPI(api: WebSocketAPIContext): void {
  api.addAuthorityListener('stuhelperGroupCenter/subscriptions/list', async function (params?: { fetchNames?: boolean }) {
    const scope = await api.resolveConsoleScope(this)
    const subscriptions = api.service.data.subscriptions.get('list') || []
    const scopedSubs = filterSubscriptions(subscriptions, scope)
    return success(buildSubscriptionList(api, scopedSubs, Boolean(params?.fetchNames)))
  })
  api.addAuthorityListener('stuhelperGroupCenter/subscriptions/add', async function (params: { subscription: Subscription }) {
    return handleSubscriptionAdd(api, this, params.subscription)
  })
  api.addAuthorityListener('stuhelperGroupCenter/subscriptions/remove', async function (params: { index: number }) {
    return handleSubscriptionRemove(api, this, params.index)
  })
  api.addAuthorityListener('stuhelperGroupCenter/subscriptions/update', async function (params: SubscriptionUpdateParams) {
    return handleSubscriptionUpdate(api, this, params)
  })
}

interface SubscriptionUpdateParams {
  readonly index: number
  readonly subscription: Subscription
}

function buildSubscriptionList(api: WebSocketAPIContext, subscriptions: Subscription[], fetchNames: boolean) {
  if (!fetchNames) {
    return subscriptions.map((sub) => ({ ...sub, name: '', avatar: '' }))
  }

  const cacheData = api.service.cache.getCachedData()
  return subscriptions.map((sub) => {
    if (sub.type === 'group') {
      const cached = cacheData.guilds[sub.id]
      return { ...sub, name: cached?.name || '', avatar: cached?.avatar || `https://p.qlogo.cn/gh/${sub.id}/${sub.id}/640/` }
    }
    const cached = cacheData.users[sub.id]
    return { ...sub, name: cached?.name || '', avatar: cached?.avatar || `https://q1.qlogo.cn/g?b=qq&nk=${sub.id}&s=640` }
  })
}

async function handleSubscriptionAdd(api: WebSocketAPIContext, client: unknown, subscription: Subscription) {
  try {
    const scope = await api.resolveConsoleScope(client)
    assertSubscriptionScope(scope, subscription)
    const list = api.service.data.subscriptions.get('list') || []
    api.service.data.subscriptions.set('list', [...list, subscription])
    await api.service.data.subscriptions.flush()
    return success({ success: true })
  } catch (cause) {
    return error(cause instanceof Error ? cause.message : '添加订阅失败')
  }
}

async function handleSubscriptionRemove(api: WebSocketAPIContext, client: unknown, index: number) {
  try {
    const scope = await api.resolveConsoleScope(client)
    const list = api.service.data.subscriptions.get('list') || []
    const rawIndex = findScopedSubscriptionRawIndex(list, scope, index)
    if (rawIndex < 0) return error('订阅不存在')
    api.service.data.subscriptions.set('list', removeAt(list, rawIndex))
    await api.service.data.subscriptions.flush()
    return success({ success: true })
  } catch (cause) {
    return error(cause instanceof Error ? cause.message : '移除订阅失败')
  }
}

async function handleSubscriptionUpdate(
  api: WebSocketAPIContext,
  client: unknown,
  params: SubscriptionUpdateParams,
) {
  try {
    const scope = await api.resolveConsoleScope(client)
    assertSubscriptionScope(scope, params.subscription)
    const list = api.service.data.subscriptions.get('list') || []
    const rawIndex = findScopedSubscriptionRawIndex(list, scope, params.index)
    if (rawIndex < 0) return error('订阅不存在')
    api.service.data.subscriptions.set('list', replaceAt(list, rawIndex, params.subscription))
    await api.service.data.subscriptions.flush()
    return success({ success: true })
  } catch (cause) {
    return error(cause instanceof Error ? cause.message : '更新订阅失败')
  }
}

function removeAt<T>(items: T[], index: number) {
  return items.filter((_, currentIndex) => currentIndex !== index)
}

function replaceAt<T>(items: T[], index: number, item: T) {
  return items.map((current, currentIndex) => currentIndex === index ? item : current)
}
