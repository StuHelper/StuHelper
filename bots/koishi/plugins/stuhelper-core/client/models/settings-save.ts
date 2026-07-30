export type SettingsSaveStepStatus = 'confirmed' | 'unconfirmed' | 'not-run'

export interface SettingsSaveStepResult {
  key: string
  label: string
  status: SettingsSaveStepStatus
}

export interface SettingsSaveStep {
  key: string
  label: string
  run: () => Promise<void>
}

export class SettingsSaveStepFailure extends Error {
  constructor(
    readonly stepKey: string,
    readonly stepLabel: string,
    cause: unknown,
  ) {
    super(`${stepLabel}保存未确认：${describeCause(cause)}`, { cause })
    this.name = 'SettingsSaveStepFailure'
  }
}

export async function runSettingsSaveSteps(
  steps: readonly SettingsSaveStep[],
  onStatusChange: (results: readonly SettingsSaveStepResult[]) => void,
): Promise<void> {
  const results = steps.map<SettingsSaveStepResult>((step) => ({
    key: step.key,
    label: step.label,
    status: 'not-run',
  }))
  emitResults(results, onStatusChange)

  for (let index = 0; index < steps.length; index += 1) {
    const step = steps[index]
    try {
      await step.run()
      results[index].status = 'confirmed'
      emitResults(results, onStatusChange)
    } catch (cause) {
      results[index].status = 'unconfirmed'
      emitResults(results, onStatusChange)
      throw new SettingsSaveStepFailure(step.key, step.label, cause)
    }
  }
}

interface KeywordRuleIdentity {
  id: string
}

export interface SaveKeywordRuleChangesOptions<T extends KeywordRuleIdentity> {
  original: readonly T[]
  next: readonly T[]
  deleteRule: (id: string) => Promise<void>
  upsertRule: (rule: T) => Promise<void>
  onBaselineChange: (rules: readonly T[]) => void
  compareRules: (left: T, right: T) => number
}

export async function saveKeywordRuleChanges<T extends KeywordRuleIdentity>(
  options: SaveKeywordRuleChangesOptions<T>,
): Promise<void> {
  let baseline = cloneAndSort(options.original, options.compareRules)
  const submitted = cloneAndSort(options.next, options.compareRules)
  const submittedIDs = new Set(submitted.map((rule) => rule.id))

  for (const rule of [...baseline]) {
    if (submittedIDs.has(rule.id)) continue
    await options.deleteRule(rule.id)
    baseline = baseline.filter((item) => item.id !== rule.id)
    options.onBaselineChange(cloneAndSort(baseline, options.compareRules))
  }

  for (const rule of submitted) {
    await options.upsertRule(structuredClone(rule))
    const index = baseline.findIndex((item) => item.id === rule.id)
    if (index >= 0) {
      baseline.splice(index, 1, structuredClone(rule))
    } else {
      baseline.push(structuredClone(rule))
    }
    baseline.sort(options.compareRules)
    options.onBaselineChange(cloneAndSort(baseline, options.compareRules))
  }
}

function emitResults(
  results: readonly SettingsSaveStepResult[],
  onStatusChange: (results: readonly SettingsSaveStepResult[]) => void,
) {
  onStatusChange(results.map((result) => ({ ...result })))
}

function cloneAndSort<T>(
  items: readonly T[],
  compare: (left: T, right: T) => number,
): T[] {
  return items.map((item) => structuredClone(item)).sort(compare)
}

function describeCause(cause: unknown): string {
  if (cause instanceof Error && cause.message.trim()) {
    return cause.message
  }
  if (typeof cause === 'string' && cause.trim()) {
    return cause
  }
  return '请求结果未知'
}
