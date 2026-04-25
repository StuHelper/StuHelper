import { Context } from 'koishi'

import { ModerationStore } from '@stuhelper/koishi-moderation-core'
import type { StuhelperCoreConfig as Config } from '@stuhelper/koishi-shared'

import { registerReviewClaimRecovery } from '../review-claim-recovery'

export function registerBackgroundJobs(ctx: Context, _config?: Config) {
  ctx.inject(['database', 'stuhelperGroupCenter'], (moduleCtx) => {
    registerReviewClaimRecovery(moduleCtx, new ModerationStore(moduleCtx))
  })
}
