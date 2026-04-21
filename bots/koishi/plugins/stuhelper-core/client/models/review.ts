import type { ReviewPageData, ReviewWorkItem } from '../page-types'

export interface ReviewModelOptions {
  workspace?: string | null
  keyword?: string
  itemId?: string | null
}

export const REVIEW_KIND_LABELS: Record<ReviewWorkItem['kind'], string> = {
  review: '复核',
  admission: '准入',
  report: '举报',
}

export const REVIEW_ACTION_LABELS = {
  execute: '执行',
  reject: '驳回',
  approve: '放行',
  deny: '拒绝',
  defer: '延期',
  dismiss: '驳回举报',
  escalate: '升级',
  'create-review': '转复核',
} as const

export function buildReviewModel(data: ReviewPageData, options: ReviewModelOptions) {
  const workspace = options.workspace || ''
  const keyword = (options.keyword || '').trim().toLowerCase()
  const filteredItems = data.items.filter((item) => {
    if (workspace && item.kind !== workspace) {
      return false
    }
    if (!keyword) {
      return true
    }
    return [item.subjectLabel, item.reason, item.guildId, item.memberId, item.secondaryLabel]
      .filter(Boolean)
      .some((value) => String(value).toLowerCase().includes(keyword))
  })

  const selectedItem = filteredItems.find((item) => item.id === options.itemId) ?? filteredItems[0] ?? null
  const relatedEvents = data.events.filter((item) => selectedItem?.relatedEventIds.includes(item.id))

  return {
    filteredItems,
    selectedItem,
    relatedEvents,
    metrics: [
      { label: '全部工作项', value: data.items.length, note: '当前统一列表中的记录总数。' },
      { label: '待复核', value: data.items.filter((item) => item.kind === 'review').length, note: '来自管理员命令或策略触发的复核。' },
      { label: '待准入', value: data.items.filter((item) => item.kind === 'admission').length, note: '仍在限制中的准入成员。' },
      { label: '举报项', value: data.items.filter((item) => item.kind === 'report').length, note: '尚需继续处理的举报记录。' },
    ],
  }
}
