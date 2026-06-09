export const COMMAND_POLICY_IDS = {
  report: 'report',
  dice: 'dice',
  muteLottery: 'mute-lottery',
  guardStatus: 'guard-status',
  guardWarnings: 'guard-warnings',
  guardReviews: 'guard-reviews',
  guardMute: 'guard-mute',
  guardKickRequest: 'guard-kick-request',
  guardBlockRequest: 'guard-block-request',
  admissionAdmin: 'admission-admin',
} as const

export type CommandPolicyId = (typeof COMMAND_POLICY_IDS)[keyof typeof COMMAND_POLICY_IDS]

export const SUPPORTED_COMMAND_POLICY_IDS = [
  COMMAND_POLICY_IDS.report,
  COMMAND_POLICY_IDS.dice,
  COMMAND_POLICY_IDS.muteLottery,
  COMMAND_POLICY_IDS.guardStatus,
  COMMAND_POLICY_IDS.guardWarnings,
  COMMAND_POLICY_IDS.guardReviews,
  COMMAND_POLICY_IDS.guardMute,
  COMMAND_POLICY_IDS.guardKickRequest,
  COMMAND_POLICY_IDS.guardBlockRequest,
  COMMAND_POLICY_IDS.admissionAdmin,
] as const satisfies readonly CommandPolicyId[]
