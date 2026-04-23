import type { GuildMember } from '../../api'
import type { ChatMessage } from '../../types'

type ChatElement = {
  type?: string
  attrs?: Record<string, unknown>
  children?: ChatElement[]
}

export type MessageNode =
  | { kind: 'text'; key: string; text: string }
  | { kind: 'at'; key: string; text: string }
  | { kind: 'face'; key: string; text: string }
  | { kind: 'image'; key: string; src: string; file?: string; openUrl?: string | null }
  | { kind: 'quote'; key: string; user: string; content: string }

export function buildMessageNodes(
  message: ChatMessage,
  sessionMessages: readonly ChatMessage[],
  members: readonly GuildMember[],
): MessageNode[] {
  const elements = toChatElements(message.elements)
  if (elements.length === 0) {
    return [{ kind: 'text', key: `${message.id}-text`, text: message.content ?? '' }]
  }

  return elements.flatMap((element, index) =>
    buildNodesForElement(element, `${message.id}-${index}`, sessionMessages, members),
  )
}

function buildNodesForElement(
  element: ChatElement,
  key: string,
  sessionMessages: readonly ChatMessage[],
  members: readonly GuildMember[],
): MessageNode[] {
  switch (element.type) {
    case 'text':
      return buildTextNode(key, readTextAttr(element.attrs))
    case 'at':
      return buildAtNode(key, element.attrs, members)
    case 'face':
      return buildFaceNode(key, element.attrs)
    case 'img':
    case 'image':
      return buildImageNode(key, element.attrs)
    case 'quote':
      return [buildQuoteNode(key, element, sessionMessages)]
    default:
      return buildChildNodes(element.children, key, sessionMessages, members)
  }
}

function buildTextNode(key: string, text: string): MessageNode[] {
  if (!text) {
    return []
  }
  return [{ kind: 'text', key, text }]
}

function buildAtNode(
  key: string,
  attrs: Record<string, unknown> | undefined,
  members: readonly GuildMember[],
): MessageNode[] {
  const id = readAttr(attrs, 'id')
  const name = readAttr(attrs, 'name') || members.find((member) => member.id === id)?.name || id || '?'
  return [{ kind: 'at', key, text: `@${name}` }]
}

function buildFaceNode(key: string, attrs: Record<string, unknown> | undefined): MessageNode[] {
  const faceId = readAttr(attrs, 'id') || '?'
  return [{ kind: 'face', key, text: `[表情:${faceId}]` }]
}

function buildImageNode(key: string, attrs: Record<string, unknown> | undefined): MessageNode[] {
  const src = readAttr(attrs, 'src') || readAttr(attrs, 'url')
  if (!src) {
    return []
  }

  return [{
    kind: 'image',
    key,
    src,
    file: readAttr(attrs, 'file') || undefined,
    openUrl: toOpenUrl(src),
  }]
}

function buildQuoteNode(
  key: string,
  element: ChatElement,
  sessionMessages: readonly ChatMessage[],
): MessageNode {
  const messageId = readAttr(element.attrs, 'id')
  const quotedMessage = sessionMessages.find((message) => message.id === messageId)
  const user = readAttr(element.attrs, 'user') || quotedMessage?.username || ''
  const content = resolveQuoteContent(element, quotedMessage)
  return { kind: 'quote', key, user, content }
}

function buildChildNodes(
  children: readonly ChatElement[] | undefined,
  key: string,
  sessionMessages: readonly ChatMessage[],
  members: readonly GuildMember[],
): MessageNode[] {
  return (children ?? []).flatMap((child, index) =>
    buildNodesForElement(child, `${key}-${index}`, sessionMessages, members),
  )
}

function resolveQuoteContent(element: ChatElement, quotedMessage?: ChatMessage): string {
  const attrContent = readAttr(element.attrs, 'content')
  if (attrContent) {
    return attrContent
  }

  const childContent = collectText(element.children)
  if (childContent) {
    return childContent
  }

  if (!quotedMessage) {
    return '[引用消息]'
  }

  return truncateText(extractPlainText(quotedMessage) || '[引用消息]')
}

function extractPlainText(message: ChatMessage): string {
  const elements = toChatElements(message.elements)
  if (elements.length === 0) {
    return message.content ?? ''
  }
  return collectText(elements)
}

function collectText(elements: readonly ChatElement[] | undefined): string {
  return (elements ?? []).map((element) => collectTextFromElement(element)).join('')
}

function collectTextFromElement(element: ChatElement): string {
  switch (element.type) {
    case 'text':
      return readTextAttr(element.attrs)
    case 'at':
      return `@${readAttr(element.attrs, 'name') || readAttr(element.attrs, 'id') || '?'}`
    case 'face':
      return `[表情:${readAttr(element.attrs, 'id') || '?'}]`
    case 'img':
    case 'image':
      return '[图片]'
    case 'quote':
      return readAttr(element.attrs, 'content') || collectText(element.children)
    default:
      return collectText(element.children)
  }
}

function truncateText(value: string): string {
  if (value.length <= 50) {
    return value
  }
  return `${value.slice(0, 50)}...`
}

function toChatElements(value: unknown): ChatElement[] {
  if (!Array.isArray(value)) {
    return []
  }
  return value.filter(isChatElement)
}

function isChatElement(value: unknown): value is ChatElement {
  return typeof value === 'object' && value !== null
}

function readTextAttr(attrs: Record<string, unknown> | undefined): string {
  return readAttr(attrs, 'content') || readAttr(attrs, 'text')
}

function readAttr(attrs: Record<string, unknown> | undefined, key: string): string {
  const value = attrs?.[key]
  if (typeof value === 'string') {
    return value
  }
  if (value === undefined || value === null) {
    return ''
  }
  return String(value)
}

function toOpenUrl(value: string): string | null {
  try {
    const url = new URL(value)
    if (url.protocol === 'http:' || url.protocol === 'https:') {
      return url.toString()
    }
  } catch {}
  return null
}
