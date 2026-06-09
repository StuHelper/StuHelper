import {
  renderMessageTemplate,
  resolveGroupGuardMessages,
  type StuhelperGroupGuardMessageConfig,
} from '@stuhelper/koishi-shared'

export type GroupGuardMessages = ReturnType<typeof resolveGroupGuardMessages>

export type GroupGuardMessageProvider = () => GroupGuardMessages | Promise<GroupGuardMessages>

export interface GroupGuardMessageStoreRef {
  getMessages(): Promise<StuhelperGroupGuardMessageConfig>
}

export function createGroupGuardMessageProvider(
  store?: GroupGuardMessageStoreRef,
): GroupGuardMessageProvider {
  if (!store) {
    return () => resolveGroupGuardMessages()
  }
  return async () => resolveGroupGuardMessages(await store.getMessages())
}

export async function getGroupGuardMessages(provider?: GroupGuardMessageProvider) {
  return provider ? await provider() : resolveGroupGuardMessages()
}

export function groupGuardMessage(
  messages: GroupGuardMessages,
  key: keyof GroupGuardMessages,
  variables: Record<string, unknown> = {},
) {
  return renderMessageTemplate(messages[key], variables)
}
