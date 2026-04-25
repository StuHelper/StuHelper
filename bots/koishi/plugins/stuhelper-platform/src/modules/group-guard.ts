import type { Context } from 'koishi'
import {
  registerGroupGuardRuntimeModels,
  startGroupGuardRuntime,
} from 'koishi-plugin-stuhelper-group-guard'
import type { StuhelperGroupGuardPluginConfig } from '@stuhelper/koishi-shared'

import type { StuhelperModule, StuhelperModuleContext } from '../module-contract'
import {
  createDefaultGroupGuardConfig,
  parseGroupGuardConfig,
  toGroupGuardPluginConfig,
  type GroupGuardConfig,
} from './group-guard-config'

const GROUP_GUARD_MODULE_ID = 'group-guard'
const GROUP_GUARD_VERSION = '0.1.0'
const GROUP_GUARD_ORDER = 10

export interface GroupGuardRuntimeAdapter {
  prepare(ctx: Context): void
  start(ctx: Context, config: StuhelperGroupGuardPluginConfig): void
}

const defaultRuntimeAdapter: GroupGuardRuntimeAdapter = {
  prepare: registerGroupGuardRuntimeModels,
  start: startGroupGuardRuntime,
}

export const groupGuardModule = createGroupGuardModule(defaultRuntimeAdapter)

export function createGroupGuardModule(
  runtimeAdapter: GroupGuardRuntimeAdapter,
): StuhelperModule<GroupGuardConfig> {
  return {
    manifest: {
      id: GROUP_GUARD_MODULE_ID,
      name: '群守卫',
      description: '管理群成员准入、消息巡检、举报和趣味群管命令。',
      version: GROUP_GUARD_VERSION,
      category: 'moderation',
      defaultEnabled: true,
      order: GROUP_GUARD_ORDER,
    },
    configSchema: {
      defaults: createDefaultGroupGuardConfig,
      parse: parseGroupGuardConfig,
    },
    permissions: createPermissions(),
    commands: createCommands(),
    events: createEvents(),
    webui: createWebuiContributions(),
    prepare: (koishi) => runtimeAdapter.prepare(koishi),
    setup: (context) => setupGroupGuardModule(context, runtimeAdapter),
  }
}

async function setupGroupGuardModule(
  context: StuhelperModuleContext<GroupGuardConfig>,
  runtimeAdapter: GroupGuardRuntimeAdapter,
) {
  const config = toGroupGuardPluginConfig(context.config)
  runtimeAdapter.start(context.koishi, config)
  await context.audit({
    moduleId: GROUP_GUARD_MODULE_ID,
    action: 'module.setup',
    summary: '群守卫模块已加载',
    payload: {
      targetGroupCount: config.guard.targetGroups.length,
      scanIntervalSeconds: config.scheduler.scanIntervalSeconds,
    },
  })
}

function createPermissions() {
  return [{
    id: 'group-guard.view',
    label: '查看群守卫',
    description: '查看群守卫模块状态和策略。',
  }, {
    id: 'group-guard.manage',
    label: '管理群守卫',
    description: '调整群守卫配置和群策略。',
  }, {
    id: 'group-guard.command.report',
    label: '使用举报',
    description: '允许提交群成员举报。',
  }, {
    id: 'group-guard.command.dice',
    label: '使用骰子',
    description: '允许使用骰子趣味命令。',
  }, {
    id: 'group-guard.command.mute-lottery',
    label: '使用抽禁言',
    description: '允许使用自助抽禁言命令。',
  }] as const
}

function createCommands() {
  return [{
    name: '举报 <targetMemberId> <reason:text>',
    description: '举报群成员并触发审核。',
    permission: 'group-guard.command.report',
  }, {
    name: '骰子 [sides:natural]',
    description: '投掷指定面数骰子。',
    permission: 'group-guard.command.dice',
  }, {
    name: '抽禁言',
    description: '随机抽取自己的禁言时长。',
    permission: 'group-guard.command.mute-lottery',
  }] as const
}

function createEvents() {
  return [{
    name: 'guild-member-added',
  }, {
    name: 'message',
  }, {
    name: 'message-deleted',
  }, {
    name: 'scan-tick',
  }] as const
}

function createWebuiContributions() {
  return [{
    id: 'group-guard.config',
    label: '群守卫配置',
    section: 'module',
  }, {
    id: 'group-guard.policy',
    label: '群守卫策略',
    section: 'policy',
  }] as const
}

export {
  createDefaultGroupGuardConfig,
  parseGroupGuardConfig,
  toGroupGuardPluginConfig,
  type GroupGuardConfig,
}
