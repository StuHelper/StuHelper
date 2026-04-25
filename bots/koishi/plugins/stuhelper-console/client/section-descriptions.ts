import type { ConsoleSectionId } from './sections'

export const SECTION_DESCRIPTIONS: Record<ConsoleSectionId, string> = {
  dashboard: '汇总积压、异常和最近变更，作为统一工作入口。',
  enforcement: '连续处理人工复核与高风险动作，减少队列切换。',
  identity: '集中完成成员准入、认证和批量处置。',
  policy: '在统一后台内维护规则、权限、模板与群绑定。',
  audit: '统一检索事件与举报，回溯发生原因和处理结果。',
}
