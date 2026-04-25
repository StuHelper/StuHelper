const JSON_INDENT = 2
const ISO_DATE_END = 10
const ISO_TIME_END = 19

export type PlatformConfig = Record<string, unknown>

export interface PlatformData {
  readonly generatedAt: string
  readonly modules: readonly ModuleSnapshot[]
  readonly auditEvents: readonly AuditEventSnapshot[]
}

export interface ModuleSnapshot {
  readonly manifest: ModuleManifest
  readonly enabled: boolean
  readonly status: ModuleStartupStatus
  readonly lastError: string | null
  readonly config: PlatformConfig
  readonly permissions: readonly PermissionDefinition[]
  readonly commands: readonly CommandSnapshot[]
  readonly events: readonly EventSnapshot[]
  readonly webui: readonly WebuiContribution[]
}

export type ModuleStartupStatus = 'pending' | 'loaded' | 'error' | 'disabled'

export interface ModuleManifest {
  readonly id: string
  readonly name: string
  readonly description: string
  readonly version: string
  readonly category: string
  readonly defaultEnabled: boolean
  readonly order: number
}

export interface PermissionDefinition {
  readonly id: string
  readonly label: string
  readonly description: string
}

export interface CommandSnapshot {
  readonly name: string
  readonly description: string
  readonly permission: string
}

export interface EventSnapshot {
  readonly name: string
}

export interface WebuiContribution {
  readonly id: string
  readonly label: string
  readonly section: string
}

export interface AuditEventSnapshot {
  readonly id: string
  readonly actor: string
  readonly moduleId: string
  readonly action: string
  readonly summary: string
  readonly payload: PlatformConfig
  readonly createdAt: string
  readonly updatedAt: string
}

export interface PlatformView {
  readonly generatedAt: string
  readonly modules: readonly ModuleListItem[]
  readonly selectedModule: SelectedModule | null
  readonly groupPolicyRows: readonly GroupPolicyRow[]
  readonly policyRows: readonly PermissionRow[]
  readonly auditRows: readonly AuditRow[]
}

export interface ModuleListItem {
  readonly id: string
  readonly name: string
  readonly description: string
  readonly category: string
  readonly enabled: boolean
  readonly statusText: string
}

export interface SelectedModule {
  readonly id: string
  readonly name: string
  readonly enabled: boolean
  readonly configText: string
}

export interface GroupPolicyRow {
  readonly id: string
  readonly moduleName: string
  readonly label: string
}

export interface PermissionRow {
  readonly id: string
  readonly moduleName: string
  readonly label: string
  readonly description: string
}

export interface AuditRow {
  readonly id: string
  readonly moduleName: string
  readonly actor: string
  readonly action: string
  readonly summary: string
  readonly createdAt: string
}

export function buildPlatformView(data: PlatformData, selectedModuleId?: string): PlatformView {
  const moduleNames = createModuleNameMap(data.modules)
  const selectedModule = selectModule(data.modules, selectedModuleId)

  return {
    generatedAt: data.generatedAt,
    modules: data.modules.map(toModuleListItem),
    selectedModule: selectedModule ? toSelectedModule(selectedModule) : null,
    groupPolicyRows: data.modules.flatMap((module) => toGroupPolicyRows(module)),
    policyRows: data.modules.flatMap((module) => toPermissionRows(module)),
    auditRows: data.auditEvents.map((event) => toAuditRow(event, moduleNames)),
  }
}

export function parseConfigText(text: string): PlatformConfig {
  let value: unknown

  try {
    value = JSON.parse(text)
  } catch {
    throw new Error('配置不是有效 JSON')
  }

  if (!isConfigObject(value)) {
    throw new Error('配置必须是 JSON 对象')
  }

  return { ...value }
}

export function formatTimestamp(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const iso = date.toISOString()
  return `${iso.slice(0, ISO_DATE_END)} ${iso.slice(ISO_DATE_END + 1, ISO_TIME_END)}`
}

function createModuleNameMap(modules: readonly ModuleSnapshot[]): ReadonlyMap<string, string> {
  return new Map(modules.map((module) => [module.manifest.id, module.manifest.name]))
}

function selectModule(
  modules: readonly ModuleSnapshot[],
  selectedModuleId?: string,
): ModuleSnapshot | null {
  return modules.find((module) => module.manifest.id === selectedModuleId) ?? modules[0] ?? null
}

function toModuleListItem(module: ModuleSnapshot): ModuleListItem {
  return {
    id: module.manifest.id,
    name: module.manifest.name,
    description: module.manifest.description,
    category: module.manifest.category,
    enabled: module.enabled,
    statusText: formatModuleStatus(module),
  }
}

function toSelectedModule(module: ModuleSnapshot): SelectedModule {
  return {
    id: module.manifest.id,
    name: module.manifest.name,
    enabled: module.enabled,
    configText: JSON.stringify(module.config, null, JSON_INDENT),
  }
}

function toGroupPolicyRows(module: ModuleSnapshot): GroupPolicyRow[] {
  return module.webui
    .filter((item) => item.section === 'policy')
    .map((item) => ({
      id: item.id,
      moduleName: module.manifest.name,
      label: item.label,
    }))
}

function toPermissionRows(module: ModuleSnapshot): PermissionRow[] {
  return module.permissions.map((permission) => ({
    id: permission.id,
    moduleName: module.manifest.name,
    label: permission.label,
    description: permission.description,
  }))
}

function toAuditRow(
  event: AuditEventSnapshot,
  moduleNames: ReadonlyMap<string, string>,
): AuditRow {
  return {
    id: event.id,
    moduleName: moduleNames.get(event.moduleId) ?? event.moduleId,
    actor: event.actor,
    action: event.action,
    summary: event.summary,
    createdAt: formatTimestamp(event.createdAt),
  }
}

function isConfigObject(value: unknown): value is PlatformConfig {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function formatModuleStatus(module: ModuleSnapshot): string {
  if (!module.enabled) return '停用'
  if (module.status === 'loaded') return '已加载'
  if (module.status === 'error') return '错误'
  return '待加载'
}
