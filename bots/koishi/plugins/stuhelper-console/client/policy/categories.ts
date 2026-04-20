export const POLICY_CATEGORIES = [
  {
    id: 'keyword-rules',
    label: '关键词规则',
    description: '维护关键词匹配、动作和命中后的处理方式。',
  },
  {
    id: 'command-policies',
    label: '命令权限',
    description: '控制关键命令的 authority 下限和角色白名单。',
  },
  {
    id: 'member-roles',
    label: '成员角色',
    description: '维护成员在群内的角色绑定，用于审核和命令分权。',
  },
  {
    id: 'guard-templates',
    label: '守卫模板',
    description: '管理群守卫模板、提醒文案和默认处罚时长。',
  },
  {
    id: 'guard-bindings',
    label: '群绑定',
    description: '将模板挂载到具体平台和群号，控制生效范围。',
  },
] as const

export type PolicyCategoryId = (typeof POLICY_CATEGORIES)[number]['id']
export type PolicyCategoryDefinition = (typeof POLICY_CATEGORIES)[number]

export const DEFAULT_POLICY_CATEGORY_ID: PolicyCategoryId = 'keyword-rules'

const POLICY_CATEGORY_ID_SET = new Set<string>(POLICY_CATEGORIES.map((item) => item.id))

export function isPolicyCategoryId(value: string | null | undefined): value is PolicyCategoryId {
  if (!value) return false
  return POLICY_CATEGORY_ID_SET.has(value)
}

export function resolvePolicyCategoryId(
  value: string | null | undefined,
): PolicyCategoryId {
  if (isPolicyCategoryId(value)) return value
  return DEFAULT_POLICY_CATEGORY_ID
}
