export const MODERATION_EVENT_TABLE = 'stuhelper_moderation_event'
export const MODERATION_REVIEW_TABLE = 'stuhelper_moderation_review'
export const MODERATION_MESSAGE_LEDGER_TABLE = 'stuhelper_moderation_message_ledger'
export const MODERATION_WARNING_TABLE = 'stuhelper_moderation_warning'
export const MODERATION_KEYWORD_RULE_TABLE = 'stuhelper_moderation_keyword_rule'
export const MODERATION_MEMBER_ROLE_TABLE = 'stuhelper_moderation_member_role'
export const MODERATION_COMMAND_POLICY_TABLE = 'stuhelper_moderation_command_policy'
export const MODERATION_FUN_PROFILE_TABLE = 'stuhelper_moderation_fun_profile'
export const MODERATION_REPORT_TABLE = 'stuhelper_moderation_report'

export const DEFAULT_REPEAT_THRESHOLD = 3
export const DEFAULT_REPEAT_WINDOW_SIZE = 3
export const DEFAULT_WARNING_EXPRESSION = 'warnings >= 3'
export const DEFAULT_MUTE_LOTTERY_SECONDS = 600

export const HIGH_RISK_ACTIONS = new Set(['kick', 'kick_and_block'])
