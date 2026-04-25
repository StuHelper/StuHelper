import type { ConfigGovernancePageData } from '../page-types'

export interface ConfigModelOptions {
  workspace?: string | null
}

type WorkspaceId = ConfigGovernancePageData['workspaces'][number]['id']

const DEFAULT_WORKSPACE: WorkspaceId = 'guild-config'

export function buildConfigGovernanceModel(
  data: ConfigGovernancePageData,
  options: ConfigModelOptions,
) {
  const currentWorkspace = resolveWorkspace(options.workspace)

  return {
    currentWorkspace,
    workspaceTabs: data.workspaces,
    templateRows: data.templates.map((item) => ({
      ...item,
      sourceLabel: item.source.label,
    })),
    bindingRows: data.bindings,
    policyRows: data.commandPolicies,
    supportedCommandIds: data.supportedCommandIds,
  }
}

function resolveWorkspace(value: string | null | undefined): WorkspaceId {
  if (value === 'templates' || value === 'bindings' || value === 'command-policies') {
    return value
  }
  return DEFAULT_WORKSPACE
}
