import {
  AIModule,
  AntiRecallModule,
  AntirepeatModule,
  AuthModule,
  BanmeModule,
  ConfigModule,
  DiceModule,
  EventModule,
  GetAuthModule,
  HelpModule,
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
  type BaseModule,
} from '../core/modules'
import {
  adaptBaseModule,
  type BaseModuleDefinition,
  type ModuleDeps,
} from './base-module-adapter'
import type { Context } from 'koishi'

type BaseModuleRegistration = Omit<BaseModuleDefinition, 'order'>

export interface RuntimeModuleInstance {
  init(): Promise<void> | void
  dispose?(): Promise<void> | void
}

export interface RuntimeModule<TInstance extends RuntimeModuleInstance = RuntimeModuleInstance> {
  readonly id: string
  readonly order?: number
  create(ctx: Context, deps: ModuleDeps): TInstance
}

const BASE_MODULE_REGISTRATIONS: BaseModuleRegistration[] = [
  { id: 'warn', ModuleType: WarnModule },
  { id: 'keyword', ModuleType: KeywordModule },
  { id: 'manage-member', ModuleType: MemberManageModule },
  { id: 'manage-message', ModuleType: MessageManageModule },
  { id: 'manage-order', ModuleType: OrderManageModule },
  { id: 'antirepeat', ModuleType: AntirepeatModule },
  { id: 'welcome', ModuleType: WelcomeModule },
  { id: 'repeat', ModuleType: RepeatModule },
  { id: 'dice', ModuleType: DiceModule },
  { id: 'banme', ModuleType: BanmeModule },
  { id: 'antirecall', ModuleType: AntiRecallModule },
  { id: 'ai', ModuleType: AIModule },
  { id: 'config', ModuleType: ConfigModule },
  { id: 'log', ModuleType: LogModule },
  { id: 'subscription', ModuleType: SubscriptionModule },
  { id: 'help', ModuleType: HelpModule },
  { id: 'report', ModuleType: ReportModule },
  { id: 'getauth', ModuleType: GetAuthModule },
  { id: 'auth', ModuleType: AuthModule },
  { id: 'event', ModuleType: EventModule },
  { id: 'status', ModuleType: StatusModule },
  { id: 'manage-cross-group', ModuleType: crossGroupModule },
]

const BASE_MODULES: BaseModuleDefinition[] = BASE_MODULE_REGISTRATIONS.map(withBaseModuleOrder)
const RUNTIME_MODULES: RuntimeModule<BaseModule>[] = BASE_MODULES.map(adaptBaseModule)

export function getRuntimeModules(): readonly RuntimeModule<BaseModule>[] {
  return [...RUNTIME_MODULES].sort(compareRuntimeModules)
}

function compareRuntimeModules(left: RuntimeModule, right: RuntimeModule) {
  return (left.order ?? 0) - (right.order ?? 0)
}

function withBaseModuleOrder(definition: BaseModuleRegistration, order: number): BaseModuleDefinition {
  return { ...definition, order }
}
