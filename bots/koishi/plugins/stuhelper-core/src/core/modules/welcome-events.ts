import type { Session } from 'koishi'

import type { WelcomeModule } from './welcome.module'

export function registerWelcomeEventListeners(host: WelcomeModule): void {
  host.ctx.on('guild-member-added', async (session) => {
    await handleMemberJoin(host, session)
  })

  host.ctx.on('guild-member-removed', async (session) => {
    await handleMemberLeave(host, session)
  })
}

async function handleMemberJoin(host: WelcomeModule, session: Session): Promise<void> {
  if (!session.guildId || !session.userId) return

  const groupConfig = host.getGroupConfigs()[session.guildId] || {}
  if (groupConfig.welcomeEnabled === false) return
  if (groupConfig.welcomeEnabled === undefined && !groupConfig.welcomeMsg) return

  const welcomeMsg = groupConfig.welcomeMsg || host.config.defaultWelcome
  if (!welcomeMsg) return

  try {
    await session.send(host.formatWelcomeMessage(welcomeMsg, session.userId, session.guildId))
    console.log(`[WelcomeModule] Sent welcome message to ${session.userId} in ${session.guildId}`)
  } catch (error) {
    console.error(`[WelcomeModule] Failed to send welcome message: ${error}`)
  }
}

async function handleMemberLeave(host: WelcomeModule, session: Session): Promise<void> {
  if (!session.guildId || !session.userId) return

  const groupConfig = host.getGroupConfigs()[session.guildId] || {}
  if (groupConfig.goodbyeEnabled === false) return
  if (groupConfig.goodbyeEnabled === undefined && !groupConfig.goodbyeMsg) return

  const goodbyeMsg = groupConfig.goodbyeMsg || host.config.defaultGoodbye
  if (!goodbyeMsg) return

  try {
    await session.send(host.formatWelcomeMessage(goodbyeMsg, session.userId, session.guildId))
    console.log(`[WelcomeModule] Sent goodbye message to ${session.userId} in ${session.guildId}`)
  } catch (error) {
    console.error(`[WelcomeModule] Failed to send goodbye message: ${error}`)
  }
}
