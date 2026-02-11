/**
 * 管理后台文本 - 中文
 */
export default {
  title: '管理后台',
  nav: {
    dashboard: '仪表盘',
    reports: '举报管理',
    reviews: '评论管理',
    logs: '操作日志'
  },
  dashboard: {
    title: '仪表盘',
    totalReviews: '总评论数',
    pendingReports: '待处理举报',
    todayReviews: '今日新增',
    weekReviews: '本周新增'
  },
  reports: {
    title: '举报管理',
    statusPending: '待处理',
    statusResolved: '已处理',
    statusRejected: '已驳回',
    reject: '驳回',
    hideReview: '隐藏评论',
    processSuccess: '举报处理成功',
    processFailed: '处理举报失败，请重试',
    empty: '暂无举报'
  },
  reviews: {
    title: '评论管理',
    allStatus: '全部状态',
    published: '已发布',
    hidden: '已隐藏',
    selectedCount: '已选 {count} 项',
    batchHide: '批量隐藏',
    batchRestore: '批量恢复',
    hide: '隐藏',
    restore: '恢复',
    updateSuccess: '评论状态更新成功',
    updateFailed: '更新评论状态失败，请重试',
    batchSuccess: '批量操作成功',
    batchFailed: '批量操作失败，请重试',
    tableContent: '内容',
    tableStatus: '状态',
    tableActions: '操作',
    empty: '暂无评论'
  },
  logs: {
    title: '操作日志',
    operator: '操作人',
    action: '操作',
    resource: '资源',
    time: '时间',
    loadMore: '加载更多',
    empty: '暂无日志'
  },
  pagination: {
    prev: '上一页',
    next: '下一页'
  }
}
