import {
  AIModule,
  AntiRecallModule,
  AntirepeatModule,
  AuthModule,
  ConfigModule,
  EventModule,
  GetAuthModule,
  KeywordModule,
  LogModule,
  MemberManageModule,
  MessageManageModule,
  OrderManageModule,
  RepeatModule,
  ReportModule,
  StatusModule,
  SubscriptionModule,
  WarnModule,
  WelcomeModule,
  crossGroupModule,
} from '../core/modules'
import { banmeRuntimeModule } from '../core/modules/banme.module'
import { diceRuntimeModule } from '../core/modules/dice.module'
import { helpRuntimeModule } from '../core/modules/help.module'
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
  { id: 'manage-message', ModuleType: MessageManageModule },
  { id: 'manage-order', ModuleType: OrderManageModule },
  { id: 'antirepeat', ModuleType: AntirepeatModule },
  { id: 'welcome', ModuleType: WelcomeModule },
  { id: 'repeat', ModuleType: RepeatModule },
  diceRuntimeModule,
  banmeRuntimeModule,
  { id: 'antirecall', ModuleType: AntiRecallModule },
  { id: 'ai', ModuleType: AIModule },
  { id: 'config', ModuleType: ConfigModule },
  { id: 'log', ModuleType: LogModule },
  { id: 'subscription', ModuleType: SubscriptionModule },
  helpRuntimeModule,
  { id: 'report', ModuleType: ReportModule },
  { id: 'getauth', ModuleType: GetAuthModule },
  { id: 'auth', ModuleType: AuthModule },
  { id: 'event', ModuleType: EventModule },
  { id: 'status', ModuleType: StatusModule },
  { id: 'manage-cross-group', ModuleType: crossGroupModule },
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
