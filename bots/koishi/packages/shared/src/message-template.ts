import type {
  StuhelperBindingMessageConfig,
  StuhelperGroupGuardMessageConfig,
} from './types/index'

export type MessageTemplateVariables = Record<string, unknown>

export const DEFAULT_BINDING_MESSAGES: StuhelperBindingMessageConfig = Object.freeze({
  directOnly: '请在私聊中发送绑定命令。',
  missingCode: '请输入绑定码，例如：{command} ABCD1234',
  successVerified: '绑定成功，当前账号已完成学生认证，加入受控群时会自动放行。',
  successUnverified: '绑定成功。当前账号还未完成学生认证，请先回到 StuHelper 完成认证。',
  unavailable: '绑定失败，平台暂时不可用。',
  invalidCode: '绑定码无效或已过期，请重新生成后再试。',
  unauthorized: '机器人服务鉴权失败，请联系管理员检查后端配置。',
  conflict: '该 QQ 号或 StuHelper 账号已经绑定过其他对象。',
  notConfigured: '后端机器人接口尚未配置完成，请联系管理员。',
})

export const DEFAULT_GROUP_GUARD_MESSAGES: StuhelperGroupGuardMessageConfig = Object.freeze({
  admissionReminder: [
    '{at} ({memberId})请在 {minutes} 分钟内完成 StuHelper 学生身份认证：',
    '{authURL}',
    '{timeoutLine}',
  ].join('\n'),
  admissionTimeoutNormal: '通过后自动解除禁言，超时将移出群聊，可重新加群认证次数：{remainingRetryCount}。',
  admissionTimeoutWithFailures: [
    '通过后自动解除禁言，超时将移出群聊',
    '您已累计{failureCount}次未认证，可重新加群认证次数：{remainingRetryCount}',
  ].join('\n'),
  admissionTimeoutBlacklist: [
    '通过后自动解除禁言，超时将移出群聊',
    '您已累计{failureCount}次未认证，本次超时未认证将永久拉黑',
  ].join('\n'),
  backendPendingReminder: [
    '{at} {reminderTemplate}',
    '认证链接暂时无法创建，机器人会自动重试。',
  ].join('\n'),
  admissionReleaseCompleted: '',
  admissionKickTimeout: '{at} 认证超时，机器人将自动移出群聊。',
  admissionBlacklistKick: '{at} 认证失败次数已达上限，已加入入群黑名单，机器人将移出群聊。',
  antiRecallNotify: '{at} 检测到撤回消息：{content}',
  moderationMuteNotice: '{at} 因 {reason} 被禁言 {seconds} 秒。',
  moderationUnmuteNotice: '{at} 已解除禁言。原因：{reason}',
  moderationKickNotice: '{at} 因 {reason} 将被移出群聊。',
  freshmanForwardSummary: [
    '新生材料审核 {applicationId}',
    '姓名：{applicantName}',
    '学校：{schoolName}',
    '专业：{departmentOrMajor}',
    'QQ：{qqID}',
    '材料：{materialType}',
    '临时身份过期：{provisionalExpiresAt}',
    '通过：新生审核通过 {applicationId}',
    '驳回：新生审核驳回 {applicationId} <原因>',
  ].join('\n'),
  publicReportMissingArgs: '请提供被举报成员 ID 和举报原因。',
  publicCommandsDisabled: '公开命令已由 StuHelper WebUI 关闭。',
  muteLotteryGroupOnly: '抽禁言只能在群聊中使用。',
  commandAccessDenied: '命令权限不足。',
  diceResult: '{memberId} 投出了 d{sides} = {result}',
  muteLotteryResult: '{memberId} 本次自助禁言 {seconds} 秒。',
  muteLotteryPityResult: '保底触发，{memberId} 本次自助禁言 {seconds} 秒。',
  reportGroupOnly: '举报命令只能在群聊中使用。',
  reportRecordedAIUnavailable: '举报已记录。当前未启用 AI 审核，事件已进入人工处理范围。',
  reportAIReviewFailed: '举报已记录，但 AI 审核失败，事件已保留供人工处理。',
  reportHighRisk: '举报已提交，AI 判定为高风险，已进入踢人/拉黑人工复核队列。',
  reportMediumRisk: '举报已提交，AI 判定为中风险，已自动警告并禁言。',
  reportLowRisk: '举报已提交，AI 判定为低风险，已自动记警告。',
  reportNoAction: '举报已提交，AI 未判定出可执行违规动作。',
  admissionCommandGroupOnly: '入群认证命令只能在群聊中使用。',
  admissionCommandsDisabled: '入群认证管理员命令已由 StuHelper WebUI 关闭。',
  admissionCommandMissingQQ: '请提供要操作的 QQ 号。',
  admissionCommandMissingOperator: '无法识别命令执行者 QQ。',
  admissionCommandUnsupportedPlatform: '当前机器人平台不支持入群认证。',
  admissionCommandPolicyDisabled: '当前群未启用 StuHelper 入群认证。',
  admissionCommandNotFound: '未找到该 QQ 的入群认证记录。',
  admissionCommandInvalidState: '当前入群认证状态不允许该操作。',
  admissionCommandUnauthorized: '机器人服务凭据无权访问入群认证接口。',
  admissionCommandFailed: '入群认证命令执行失败。{error}',
  admissionCommandPlatformError: 'StuHelper 平台接口异常：{status} {message}',
  admissionCommandMissingResendURL: '后端没有返回可重发的认证链接。',
  admissionCommandStaleRecord: '入群认证记录已被其他任务处理，请重新查询当前状态。',
  admissionSkipSuccess: [
    '{at} ({qqID}) 已跳过本群入群认证并解除禁言。',
    '此操作只在本群生效，不代表 StuHelper 学生认证已通过。',
  ].join('\n'),
  admissionAlreadyVerifiedRegenerate: '{at} ({qqID}) 已完成 StuHelper 学生身份认证，已解除禁言，无需重新生成认证链接。',
  admissionResetFailureCountSuccess: '已清空 QQ {qqID} 在本群的入群未认证次数（原次数：{previousFailureCount}）。',
  admissionReleaseBlacklistNotFound: 'QQ {qqID} 在本群没有活动入群拉黑记录。',
  admissionReleaseBlacklistSuccess: '已解除 QQ {qqID} 在本群的入群拉黑状态；未认证次数未清空，如需重新计数请使用“清空入群未认证次数 {qqID}”。',
})

export function resolveBindingMessages(config?: Partial<StuhelperBindingMessageConfig>) {
  return { ...DEFAULT_BINDING_MESSAGES, ...config }
}

export function resolveGroupGuardMessages(config?: Partial<StuhelperGroupGuardMessageConfig>) {
  return { ...DEFAULT_GROUP_GUARD_MESSAGES, ...config }
}

export function renderMessageTemplate(template: string, variables: MessageTemplateVariables = {}) {
  if (!template.trim()) return ''
  return template.replace(/\{([A-Za-z0-9_]+)\}/g, (match, key: string) => {
    const value = variables[key]
    if (value === null || value === undefined) return ''
    return String(value)
  })
}
