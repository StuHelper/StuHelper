export interface ShellCommandIntent {
  readonly id: string
  readonly kind: 'command'
  readonly title: string
  readonly hint: string
  readonly name: string
  readonly path: string
  readonly keywords: readonly string[]
}

interface ShellCommandDefinition {
  readonly name: string
  readonly usage?: string
  readonly description: string
  readonly group: string
  readonly keywords: readonly string[]
}

const STUHELPER_COMMANDS: readonly ShellCommandDefinition[] = [
  {
    name: '群审状态',
    usage: '[guildId]',
    description: '查看当前群待认证成员',
    group: '群审管理',
    keywords: ['guard-status', '待认证', '成员', '状态'],
  },
  {
    name: '群审警告',
    usage: '<memberId> [guildId]',
    description: '查看成员当前警告次数',
    group: '群审管理',
    keywords: ['guard-warnings', '警告', '成员'],
  },
  {
    name: '群审复核',
    usage: '[guildId]',
    description: '查看当前群待复核队列',
    group: '群审管理',
    keywords: ['guard-review', '复核', '队列'],
  },
  {
    name: '群审禁言',
    usage: '<payload>',
    description: '批量禁言待认证成员',
    group: '群审管理',
    keywords: ['guard-mute', '禁言', '批量'],
  },
  {
    name: '群审踢人申请',
    usage: '<memberId> <reason>',
    description: '提交踢人复核申请',
    group: '群审管理',
    keywords: ['guard-kick-request', '踢人', '复核申请'],
  },
  {
    name: '群审拉黑申请',
    usage: '<memberId> <reason>',
    description: '提交踢人并拉黑复核申请',
    group: '群审管理',
    keywords: ['guard-block-request', '拉黑', '复核申请'],
  },
  {
    name: '新生审核查看',
    usage: '<applicationID>',
    description: '查看新生认证申请',
    group: '新生审核',
    keywords: ['freshman', 'admission', 'application', '查看申请'],
  },
  {
    name: '新生审核通过',
    usage: '<applicationID> [+30d]',
    description: '通过新生认证申请',
    group: '新生审核',
    keywords: ['freshman', 'admission', 'approve', '通过申请'],
  },
  {
    name: '新生审核驳回',
    usage: '<applicationID> <reason>',
    description: '驳回新生认证申请',
    group: '新生审核',
    keywords: ['freshman', 'admission', 'reject', '驳回申请'],
  },
  {
    name: '新生黑名单解除',
    usage: '<qqID> <guildID|global>',
    description: '解除新生入群认证黑名单',
    group: '新生审核',
    keywords: ['freshman', 'blacklist', 'release', '解除黑名单'],
  },
  {
    name: '查询入群认证',
    usage: '<qqID>',
    description: '查询指定 QQ 的入群认证状态',
    group: '入群认证',
    keywords: ['admission', 'join', 'session', '查询认证'],
  },
  {
    name: '重发认证链接',
    usage: '<qqID>',
    description: '重发当前仍可继续的入群认证链接',
    group: '入群认证',
    keywords: ['admission', 'resend', 'link', '认证链接'],
  },
  {
    name: '重新生成认证链接',
    usage: '<qqID>',
    description: '取消旧会话并重新生成入群认证链接',
    group: '入群认证',
    keywords: ['admission', 'regenerate', 'link', '认证链接'],
  },
  {
    name: '跳过入群认证',
    usage: '<qqID>',
    description: '仅跳过当前群的入群认证',
    group: '入群认证',
    keywords: ['admission', 'skip', '跳过认证'],
  },
  {
    name: '清空入群未认证次数',
    usage: '<qqID>',
    description: '清空当前群成员的入群认证失败次数',
    group: '入群认证',
    keywords: ['admission', 'reset', 'failure-count', '未认证次数'],
  },
  {
    name: '解除入群拉黑',
    usage: '<qqID>',
    description: '解除当前群成员的入群拉黑状态',
    group: '入群认证',
    keywords: ['admission', 'blacklist', 'release', '解除拉黑'],
  },
  {
    name: '举报',
    usage: '<targetMemberId> <reason>',
    description: '举报群成员并触发审核',
    group: '公开命令',
    keywords: ['report', '举报群成员', '审核'],
  },
  {
    name: '骰子',
    usage: '[sides]',
    description: '投掷骰子',
    group: '公开命令',
    keywords: ['dice', 'roll', '娱乐'],
  },
  {
    name: '抽禁言',
    description: '随机抽取自己的禁言时长，带保底机制',
    group: '公开命令',
    keywords: ['mute-lottery', 'banme', '随机禁言', '保底'],
  },
]

export const SHELL_COMMAND_INTENTS: readonly ShellCommandIntent[] = STUHELPER_COMMANDS.map((command) => ({
  id: `c:${command.name}`,
  kind: 'command',
  title: command.usage ? `${command.name} ${command.usage}` : command.name,
  hint: `${command.group} / ${command.description}`,
  name: command.name,
  path: commandRoutePath(command.name),
  keywords: command.keywords,
}))

export function commandRoutePath(commandName: string): string {
  return `/commands/${commandName.replace(/\./g, '/')}`
}

export function searchCommandIntents(query: string): readonly ShellCommandIntent[] {
  const normalized = query.trim().toLowerCase()
  if (!normalized) return []

  return SHELL_COMMAND_INTENTS.filter((intent) => {
    return intent.title.toLowerCase().includes(normalized) ||
      intent.hint.toLowerCase().includes(normalized) ||
      intent.name.toLowerCase().includes(normalized) ||
      intent.keywords.some((keyword) => keyword.toLowerCase().includes(normalized))
  })
}
