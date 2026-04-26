import * as fs from 'fs'

import type { ChatMessage, UserContext } from '../../types'
import type { AIModule } from './ai.module'

interface AddMessageInput {
  readonly userId: string
  readonly message: ChatMessage
  readonly systemPrompt: string
  readonly contextLimit: number
}

export function initDataFiles(host: AIModule): void {
  if (!fs.existsSync(host.contextsPath)) {
    fs.writeFileSync(host.contextsPath, JSON.stringify({}), 'utf8')
  }
}

export function loadContexts(host: AIModule): void {
  try {
    const data = JSON.parse(fs.readFileSync(host.contextsPath, 'utf8'))
    for (const [userId, context] of Object.entries(data)) {
      host.userContexts.set(userId, context as UserContext)
    }
    host.data.writeLog(`[ai] Loaded ${host.userContexts.size} AI contexts`)
  } catch (error) {
    host.data.writeLog(`[ai] Failed to load AI contexts: ${error}`)
    host.userContexts.clear()
  }
}

export function saveContexts(host: AIModule): void {
  try {
    const data: Record<string, UserContext> = {}
    for (const [userId, context] of host.userContexts.entries()) {
      data[userId] = context
    }
    fs.writeFileSync(host.contextsPath, JSON.stringify(data, null, 2), 'utf8')
  } catch (error) {
    host.data.writeLog(`[ai] Failed to save AI contexts: ${error}`)
  }
}

export function cleanExpiredContexts(host: AIModule): void {
  const now = Date.now()
  let cleanCount = 0

  for (const [userId, context] of host.userContexts.entries()) {
    if (now - context.lastTimestamp > host.contextTimeout) {
      host.userContexts.delete(userId)
      cleanCount++
    }
  }

  if (cleanCount > 0) {
    host.data.writeLog(`[ai] Cleaned ${cleanCount} expired AI contexts`)
    saveContexts(host)
  }
}

export function addMessageToContext(host: AIModule, input: AddMessageInput): void {
  const context = getUserContext(host, input.userId, input.systemPrompt)
  context.messages.push(input.message)
  context.lastTimestamp = Date.now()

  if (context.messages.length > input.contextLimit + 1) {
    const systemMessage = context.messages[0]
    const recentMessages = context.messages.slice(-input.contextLimit)
    context.messages = [systemMessage, ...recentMessages]
  }

  saveContexts(host)
}

export function getUserContext(host: AIModule, userId: string, systemPrompt: string): UserContext {
  if (!host.userContexts.has(userId)) {
    host.userContexts.set(userId, {
      userId,
      messages: [{ role: 'system', content: systemPrompt }],
      lastTimestamp: Date.now(),
    })
  }

  const context = host.userContexts.get(userId)!
  if (context.messages[0].role === 'system' && context.messages[0].content !== systemPrompt) {
    context.messages[0].content = systemPrompt
  }
  return context
}
