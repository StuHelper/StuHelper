import {
  AIModule,
  KeywordModule,
  ReportModule,
  WarnModule,
} from '../core/modules'
import { antirecallRuntimeModule } from '../core/modules/antirecall.module'
import { antirepeatRuntimeModule } from '../core/modules/antirepeat.module'
import { authRuntimeModule } from '../core/modules/auth.module'
import { banmeRuntimeModule } from '../core/modules/banme.module'
import { configRuntimeModule } from '../core/modules/config.module'
import { crossGroupRuntimeModule } from '../core/modules/crossGroupManage.module'
import { diceRuntimeModule } from '../core/modules/dice.module'
import { eventRuntimeModule } from '../core/modules/event.module'
import { getauthRuntimeModule } from '../core/modules/getauth.module'
import { helpRuntimeModule } from '../core/modules/help.module'
import { logRuntimeModule } from '../core/modules/log.module'
import { memberManageRuntimeModule } from '../core/modules/memberManage.module'
import { messageManageRuntimeModule } from '../core/modules/messageManage.module'
import { orderManageRuntimeModule } from '../core/modules/orderManage.module'
import { repeatRuntimeModule } from '../core/modules/repeat.module'
import { statusRuntimeModule } from '../core/modules/status.module'
import { subscriptionRuntimeModule } from '../core/modules/subscription.module'
import { welcomeRuntimeModule } from '../core/modules/welcome.module'
import {
  adaptBaseModule,
  type BaseModuleDefinition,
} from './base-module-adapter'
import type { RuntimeModule, RuntimeModuleInstance } from './types'

type BaseModuleRegistration = Omit<BaseModuleDefinition, 'order'>
type RuntimeModuleRegistration = BaseModuleRegistration | RuntimeModule<RuntimeModuleInstance>

const MODULE_REGISTRATIONS: RuntimeModuleRegistration[] = [
  { id: 'warn', ModuleType: WarnModule },
  { id: 'keyword', ModuleType: KeywordModule },
  memberManageRuntimeModule,
  messageManageRuntimeModule,
  orderManageRuntimeModule,
  antirepeatRuntimeModule,
  welcomeRuntimeModule,
  repeatRuntimeModule,
  diceRuntimeModule,
  banmeRuntimeModule,
  antirecallRuntimeModule,
  { id: 'ai', ModuleType: AIModule },
  configRuntimeModule,
  logRuntimeModule,
  subscriptionRuntimeModule,
  helpRuntimeModule,
  { id: 'report', ModuleType: ReportModule },
  getauthRuntimeModule,
  authRuntimeModule,
  eventRuntimeModule,
  statusRuntimeModule,
  crossGroupRuntimeModule,
]

const RUNTIME_MODULES = MODULE_REGISTRATIONS.map(createOrderedRuntimeModule)

export function getRuntimeModules(): readonly RuntimeModule<RuntimeModuleInstance>[] {
  return [...RUNTIME_MODULES].sort(compareRuntimeModules)
}

function compareRuntimeModules(left: RuntimeModule, right: RuntimeModule) {
  return (left.order ?? 0) - (right.order ?? 0)
}

function createOrderedRuntimeModule(
  definition: RuntimeModuleRegistration,
  order: number,
): RuntimeModule<RuntimeModuleInstance> {
  if ('ModuleType' in definition) {
    return adaptBaseModule({ ...definition, order })
  }

  return { ...definition, order }
}
