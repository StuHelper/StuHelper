import type { StuhelperModule } from './module-contract'

export interface StuhelperModuleRegistry {
  list(): readonly StuhelperModule[]
  get(id: string): StuhelperModule | null
}

export function createModuleRegistry(modules: readonly StuhelperModule[]): StuhelperModuleRegistry {
  const byId = new Map<string, StuhelperModule>()

  for (const module of modules) {
    const id = module.manifest.id
    if (byId.has(id)) {
      throw new Error(`duplicate StuHelper module id: ${id}`)
    }
    byId.set(id, module)
  }

  const sorted = [...modules].sort(compareModules)

  return {
    list: () => [...sorted],
    get: (id: string) => byId.get(id) ?? null,
  }
}

function compareModules(left: StuhelperModule, right: StuhelperModule) {
  return left.manifest.order - right.manifest.order
    || left.manifest.id.localeCompare(right.manifest.id)
}
