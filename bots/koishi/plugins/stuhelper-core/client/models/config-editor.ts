import type {
  PolicyFormState,
  TemplateFormState,
} from './config-forms'
import type { GroupConfig } from '../types'

export const DISCARD_CHANGES_MESSAGE = '当前有未保存的改动，继续会丢失这些修改。'

const DEFAULT_FORBIDDEN_CONFIG: NonNullable<GroupConfig['forbidden']> = {
  autoDelete: false,
  autoBan: false,
  autoKick: false,
  muteDuration: 600000,
}

const DEFAULT_DICE_CONFIG: NonNullable<GroupConfig['dice']> = {
  enabled: true,
  lengthLimit: 1000,
}

const DEFAULT_ANTI_REPEAT_CONFIG: NonNullable<GroupConfig['antiRepeat']> = {
  enabled: false,
  threshold: 0,
}

const DEFAULT_BANME_CONFIG: NonNullable<GroupConfig['banme']> = {
  enabled: true,
  baseMin: 1,
  baseMax: 30,
  growthRate: 30,
  autoBan: false,
  jackpot: {
    enabled: true,
    baseProb: 0.006,
    softPity: 73,
    hardPity: 89,
    upDuration: '24h',
    loseDuration: '12h',
  },
}

const DEFAULT_OPENAI_CONFIG: NonNullable<GroupConfig['openai']> = {
  enabled: true,
  chatEnabled: true,
  translateEnabled: true,
}

const DEFAULT_ANTI_RECALL_CONFIG: NonNullable<GroupConfig['antiRecall']> = {
  enabled: false,
}

const DEFAULT_REPORT_CONFIG: NonNullable<GroupConfig['report']> = {
  enabled: true,
  autoProcess: true,
  includeContext: false,
  contextSize: 10,
}

export function cloneTemplateForm(state: TemplateFormState): TemplateFormState {
  return { ...state }
}

export function clonePolicyForm(state: PolicyFormState): PolicyFormState {
  return { ...state }
}

export function normalizeGroupConfigForEdit(config: GroupConfig | undefined): GroupConfig {
  const draft = clonePlainConfig(config ?? {})
  const banme = draft.banme

  return {
    ...draft,
    approvalKeywords: [...(draft.approvalKeywords ?? [])],
    keywords: [...(draft.keywords ?? [])],
    antiRecall: {
      ...DEFAULT_ANTI_RECALL_CONFIG,
      ...(draft.antiRecall ?? {}),
    },
    antiRepeat: {
      ...DEFAULT_ANTI_REPEAT_CONFIG,
      ...(draft.antiRepeat ?? {}),
    },
    banme: {
      ...DEFAULT_BANME_CONFIG,
      ...(banme ?? {}),
      jackpot: {
        ...DEFAULT_BANME_CONFIG.jackpot,
        ...(banme?.jackpot ?? {}),
      },
    },
    dice: {
      ...DEFAULT_DICE_CONFIG,
      ...(draft.dice ?? {}),
    },
    forbidden: {
      ...DEFAULT_FORBIDDEN_CONFIG,
      ...(draft.forbidden ?? {}),
    },
    openai: {
      ...DEFAULT_OPENAI_CONFIG,
      ...(draft.openai ?? {}),
    },
    report: {
      ...DEFAULT_REPORT_CONFIG,
      ...(draft.report ?? {}),
    },
  }
}

export function isTemplateFormDirty(state: TemplateFormState, snapshot: TemplateFormState | null) {
  return !snapshot || !isSameTemplateForm(state, snapshot)
}

export function isPolicyFormDirty(state: PolicyFormState, snapshot: PolicyFormState | null) {
  return !snapshot || !isSamePolicyForm(state, snapshot)
}

export function confirmDiscardChanges(
  dirty: boolean,
  confirm: (message: string) => boolean,
) {
  return !dirty || confirm(DISCARD_CHANGES_MESSAGE)
}

function isSameTemplateForm(left: TemplateFormState, right: TemplateFormState) {
  return left.id === right.id
    && left.name === right.name
    && left.muteDurationSeconds === right.muteDurationSeconds
    && left.kickAfterMinutes === right.kickAfterMinutes
    && left.reminderTemplate === right.reminderTemplate
    && left.exemptUsersText === right.exemptUsersText
    && left.enabled === right.enabled
}

function isSamePolicyForm(left: PolicyFormState, right: PolicyFormState) {
  return left.commandId === right.commandId
    && left.minAuthority === right.minAuthority
    && left.rolesText === right.rolesText
}

function clonePlainConfig<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T
}
