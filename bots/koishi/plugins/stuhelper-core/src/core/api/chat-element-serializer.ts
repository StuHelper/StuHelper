type ChatElement = {
  type?: string
  attrs?: Record<string, unknown>
  children?: ChatElement[]
}

type QuotePayload = {
  id?: string
  messageId?: string
  user?: string
  content?: string
}

const ATTR_ESCAPE_MAP: Record<string, string> = {
  '&': '&amp;',
  '<': '&lt;',
  '>': '&gt;',
  '"': '&quot;',
  "'": '&#39;',
}

export function prependQuoteElement(
  elements: readonly ChatElement[] | undefined,
  quote?: QuotePayload,
): ChatElement[] {
  const messageId = resolveQuoteMessageId(quote)
  if (!messageId) {
    return [...(elements ?? [])]
  }

  const attrs: Record<string, string> = { id: messageId }
  if (quote.user) attrs.user = quote.user
  if (quote.content) attrs.content = quote.content

  return [{ type: 'quote', attrs }, ...(elements ?? [])]
}

function resolveQuoteMessageId(quote: QuotePayload | undefined): string {
  return quote?.id || quote?.messageId || ''
}

export function serializeChatElements(elements: readonly ChatElement[]): string {
  return elements.map(serializeElement).join('')
}

function serializeElement(element: ChatElement): string {
  if (!element?.type) {
    return ''
  }
  if (element.type === 'text') {
    return escapeText(readTextContent(element.attrs))
  }

  const attrs = serializeAttrs(element.attrs)
  const openTag = attrs ? `<${element.type} ${attrs}` : `<${element.type}`
  if (!element.children?.length) {
    return `${openTag} />`
  }
  return `${openTag}>${serializeChatElements(element.children)}</${element.type}>`
}

function readTextContent(attrs: Record<string, unknown> | undefined): string {
  const content = attrs?.content ?? attrs?.text ?? ''
  return typeof content === 'string' ? content : String(content)
}

function serializeAttrs(attrs: Record<string, unknown> | undefined): string {
  if (!attrs) {
    return ''
  }

  return Object.entries(attrs)
    .filter(([, value]) => value !== undefined && value !== null && value !== '')
    .map(([key, value]) => `${key}="${escapeAttribute(String(value))}"`)
    .join(' ')
}

function escapeText(value: string): string {
  return value.replace(/[&<>]/g, (char) => ATTR_ESCAPE_MAP[char] ?? char)
}

function escapeAttribute(value: string): string {
  return value.replace(/[&<>"']/g, (char) => ATTR_ESCAPE_MAP[char] ?? char)
}
