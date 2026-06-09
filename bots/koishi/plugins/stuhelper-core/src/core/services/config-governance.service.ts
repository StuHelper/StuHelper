import type { CommandPolicyRecord } from '@stuhelper/koishi-moderation-core'
import type {
  GuardGroupBindingRecord,
  GuardTemplateRecord,
} from '@stuhelper/koishi-shared'

import { serializeCommandPolicy, serializeGuardBinding, serializeGuardTemplate } from './page-serializers'
import type { ConfigGovernancePageData } from './page-types'

export interface ConfigGovernanceBuilderInput {
  generatedAt: string
  groupConfigs: Record<string, Record<string, unknown>>
  guildNames: Record<string, { name?: string; avatar?: string }>
  templates: GuardTemplateRecord[]
  bindings: GuardGroupBindingRecord[]
  commandPolicies: CommandPolicyRecord[]
  supportedCommandIds: string[]
}

export function buildConfigGovernanceData(input: ConfigGovernanceBuilderInput): ConfigGovernancePageData {
  const templateMap = new Map(input.templates.map((template) => [template.id, template.name]))

  return {
    generatedAt: input.generatedAt,
    workspaces: [
      { id: 'guild-config', label: '群配置' },
      { id: 'templates', label: '模板库' },
      { id: 'bindings', label: '同步绑定' },
      { id: 'command-policies', label: '命令策略' },
    ],
    groupConfigs: Object.entries(input.groupConfigs)
      .map(([guildId, config]) => ({
        guildId,
        guildName: input.guildNames[guildId]?.name || '',
        guildAvatar: input.guildNames[guildId]?.avatar || '',
        config,
      }))
      .sort((left, right) => left.guildId.localeCompare(right.guildId)),
    templates: [...input.templates]
      .sort((left, right) => left.name.localeCompare(right.name))
      .map((template) => ({
        ...serializeGuardTemplate(template),
        source: {
          kind: 'template-library' as const,
          label: '模板库',
        },
      })),
    bindings: [...input.bindings]
      .sort((left, right) => left.guildId.localeCompare(right.guildId))
      .map((binding) => ({
        ...serializeGuardBinding(binding),
        effectiveTemplateName: templateMap.get(binding.templateId) || binding.templateId,
      })),
    commandPolicies: [...input.commandPolicies]
      .sort((left, right) => left.commandId.localeCompare(right.commandId))
      .map(serializeCommandPolicy),
    supportedCommandIds: [...input.supportedCommandIds],
  }
}

export interface ConfigGovernanceServiceDeps {
  loadGroupConfigs: () => Promise<Record<string, Record<string, unknown>>> | Record<string, Record<string, unknown>>
  loadGuildNames: () => Promise<Record<string, { name?: string; avatar?: string }>> | Record<string, { name?: string; avatar?: string }>
  loadTemplates: () => Promise<GuardTemplateRecord[]>
  loadBindings: () => Promise<GuardGroupBindingRecord[]>
  loadCommandPolicies: () => Promise<CommandPolicyRecord[]>
  loadSupportedCommandIds: () => Promise<string[]> | string[]
}

export class ConfigGovernanceService {
  constructor(private readonly deps: ConfigGovernanceServiceDeps) {}

  async getPageData() {
    const [groupConfigs, guildNames, templates, bindings, commandPolicies, supportedCommandIds] = await Promise.all([
      this.deps.loadGroupConfigs(),
      this.deps.loadGuildNames(),
      this.deps.loadTemplates(),
      this.deps.loadBindings(),
      this.deps.loadCommandPolicies(),
      this.deps.loadSupportedCommandIds(),
    ])

    return buildConfigGovernanceData({
      generatedAt: new Date().toISOString(),
      groupConfigs,
      guildNames,
      templates,
      bindings,
      commandPolicies,
      supportedCommandIds: [...supportedCommandIds],
    })
  }
}
