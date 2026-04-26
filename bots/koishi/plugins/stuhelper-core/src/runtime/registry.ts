import {
  AIModule,
  AntiRecallModule,
  AuthModule,
  KeywordModule,
  LogModule,
  MemberManageModule,
  OrderManageModule,
  RepeatModule,
  ReportModule,
  SubscriptionModule,
  WarnModule,
  WelcomeModule,
} from '../core/modules'
import { antirepeatRuntimeModule } from '../core/modules/antirepeat.module'
import { banmeRuntimeModule } from '../core/modules/banme.module'
import { configRuntimeModule } from '../core/modules/config.module'
import { crossGroupRuntimeModule } from '../core/modules/crossGroupManage.module'
import { diceRuntimeModule } from '../core/modules/dice.module'
import { eventRuntimeModule } from '../core/modules/event.module'
import { getauthRuntimeModule } from '../core/modules/getauth.module'
import { helpRuntimeModule } from '../core/modules/help.module'
import { messageManageRuntimeModule } from '../core/modules/messageManage.module'
import { statusRuntimeModule } from '../core/modules/status.module'
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
  { id: 'manage-member', ModuleType: MemberManageModule },
  messageManageRuntimeModule,
  { id: 'manage-order', ModuleType: OrderManageModule },
  antirepeatRuntimeModule,
  { id: 'welcome', ModuleType: WelcomeModule },
  { id: 'repeat', ModuleType: RepeatModule },
  diceRuntimeModule,
  banmeRuntimeModule,
  { id: 'antirecall', ModuleType: AntiRecallModule },
  { id: 'ai', ModuleType: AIModule },
  configRuntimeModule,
  { id: 'log', ModuleType: LogModule },
  { id: 'subscription', ModuleType: SubscriptionModule },
  helpRuntimeModule,
  { id: 'report', ModuleType: ReportModule },
  getauthRuntimeModule,
  { id: 'auth', ModuleType: AuthModule },
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
