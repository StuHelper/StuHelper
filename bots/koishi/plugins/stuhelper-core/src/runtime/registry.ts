import { antirecallRuntimeModule } from '../core/modules/antirecall.module'
import { antirepeatRuntimeModule } from '../core/modules/antirepeat.module'
import { aiRuntimeModule } from '../core/modules/ai.module'
import { authRuntimeModule } from '../core/modules/auth.module'
import { banmeRuntimeModule } from '../core/modules/banme.module'
import { configRuntimeModule } from '../core/modules/config.module'
import { crossGroupRuntimeModule } from '../core/modules/crossGroupManage.module'
import { diceRuntimeModule } from '../core/modules/dice.module'
import { eventRuntimeModule } from '../core/modules/event.module'
import { getauthRuntimeModule } from '../core/modules/getauth.module'
import { helpRuntimeModule } from '../core/modules/help.module'
import { keywordRuntimeModule } from '../core/modules/keyword.module'
import { logRuntimeModule } from '../core/modules/log.module'
import { memberManageRuntimeModule } from '../core/modules/memberManage.module'
import { messageManageRuntimeModule } from '../core/modules/messageManage.module'
import { orderManageRuntimeModule } from '../core/modules/orderManage.module'
import { reportRuntimeModule } from '../core/modules/report.module'
import { repeatRuntimeModule } from '../core/modules/repeat.module'
import { statusRuntimeModule } from '../core/modules/status.module'
import { subscriptionRuntimeModule } from '../core/modules/subscription.module'
import { warnRuntimeModule } from '../core/modules/warn.module'
import { welcomeRuntimeModule } from '../core/modules/welcome.module'
import type { RuntimeModule, RuntimeModuleInstance } from './types'

const MODULE_REGISTRATIONS: RuntimeModule<RuntimeModuleInstance>[] = [
  warnRuntimeModule,
  keywordRuntimeModule,
  memberManageRuntimeModule,
  messageManageRuntimeModule,
  orderManageRuntimeModule,
  antirepeatRuntimeModule,
  welcomeRuntimeModule,
  repeatRuntimeModule,
  diceRuntimeModule,
  banmeRuntimeModule,
  antirecallRuntimeModule,
  aiRuntimeModule,
  configRuntimeModule,
  logRuntimeModule,
  subscriptionRuntimeModule,
  helpRuntimeModule,
  reportRuntimeModule,
  getauthRuntimeModule,
  authRuntimeModule,
  eventRuntimeModule,
  statusRuntimeModule,
  crossGroupRuntimeModule,
]

const RUNTIME_MODULES = MODULE_REGISTRATIONS.map(withOrder)

export function getRuntimeModules(): readonly RuntimeModule<RuntimeModuleInstance>[] {
  return [...RUNTIME_MODULES].sort(compareRuntimeModules)
}

function compareRuntimeModules(left: RuntimeModule, right: RuntimeModule) {
  return (left.order ?? 0) - (right.order ?? 0)
}

function withOrder(
  definition: RuntimeModule<RuntimeModuleInstance>,
  order: number,
): RuntimeModule<RuntimeModuleInstance> {
  return { ...definition, order }
}
