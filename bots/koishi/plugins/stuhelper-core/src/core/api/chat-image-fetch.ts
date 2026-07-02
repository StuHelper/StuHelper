import { readFile } from 'node:fs/promises'
import { createHash } from 'node:crypto'

import type { Bot } from 'koishi'

import {
  assertConsoleGuildAccess,
  type ConsoleGuildScope,
} from './console-guild-scope'
import { getOneBotInternalMethod } from '../onebot-internal'
import type {
  ChatImageAccessRegistry,
  ChatImageFetchParams,
  ChatImageFetchRequest,
  ChatImageFetchResult,
  ChatImageLogger,
} from './chat-image-types'

export type {
  ChatImageAccessRegistry,
  ChatImageFetchParams,
  ChatImageFetchRequest,
  ChatImageFetchResult,
  ChatImageLogger,
} from './chat-image-types'

const HASH_ALGORITHM = 'md5'
const IMAGE_DATA_URL_PREFIX = 'data:'
const BASE64_ENCODING = 'base64'
const DEFAULT_IMAGE_MIME_TYPE = 'image/png'
const IMAGE_PROXY_HOST_SUFFIXES = Object.freeze([
  'gchat.qpic.cn',
  'multimedia.nt.qq.com.cn',
  'c2cpicdw.qpic.cn',
])
const IMAGE_MIME_BY_EXTENSION = Object.freeze(new Map<string, string>([
  ['.jpg', 'image/jpeg'],
  ['.jpeg', 'image/jpeg'],
  ['.png', 'image/png'],
  ['.gif', 'image/gif'],
  ['.webp', 'image/webp'],
]))

/** 聊天图片访问授权有效期：过期后必须等待消息重新推送才能再取图 */
const CHAT_IMAGE_ACCESS_TTL_MS = 30 * 60 * 1000
/** 注册表容量上限：超出时先清理过期条目，再按插入顺序淘汰最旧条目 */
const CHAT_IMAGE_ACCESS_CAPACITY = 5_000

export interface ChatImageAccessRegistryOptions {
  readonly ttlMs?: number
  readonly capacity?: number
  readonly now?: () => number
}

interface ChatImageAccessEntry {
  readonly guildIds: ReadonlySet<string | undefined>
  readonly expiresAt: number
}

export function createChatImageAccessRegistry(
  options: ChatImageAccessRegistryOptions = {},
): ChatImageAccessRegistry {
  const ttlMs = options.ttlMs ?? CHAT_IMAGE_ACCESS_TTL_MS
  const capacity = options.capacity ?? CHAT_IMAGE_ACCESS_CAPACITY
  const now = options.now ?? Date.now
  const entries = new Map<string, ChatImageAccessEntry>()

  return {
    remember(elements, guildId) {
      const at = now()
      for (const image of readImageElements(elements)) {
        rememberImageEntry(entries, image, guildId, at + ttlMs)
      }
      evictImageEntries(entries, capacity, at)
    },
    assertAllowed(params, scope) {
      const request = parseChatImageFetchParams(params)
      const key = accessKey(request)
      const entry = entries.get(key)
      if (entry && entry.expiresAt <= now()) {
        entries.delete(key)
      }
      if (!entry || entry.expiresAt <= now()) {
        throw new Error('image file is not attached to a delivered chat message')
      }
      assertImageScope(scope, entry.guildIds)
      return request
    },
  }
}

export async function fetchOneBotImage(
  bots: Iterable<Bot>,
  request: ChatImageFetchRequest,
  logger: ChatImageLogger,
): Promise<ChatImageFetchResult> {
  const imageErrors: string[] = []

  for (const bot of bots) {
    const getImage = getOneBotInternalMethod(bot, 'getImage')
    if (!getImage) continue

    try {
      const result = await getImage(request.file)
      const payload = await resolveOneBotImageResult(result, request)
      if (payload) return payload
      imageErrors.push('OneBot get_image did not return an image payload')
    } catch (error) {
      const message = toErrorMessage(error)
      imageErrors.push(message)
      logger.warn('OneBot get_image 获取图片失败: %s', message)
    }
  }

  const suffix = imageErrors.length > 0 ? `: ${imageErrors[imageErrors.length - 1]}` : ''
  throw new Error(`无法获取图片${suffix}`)
}

function rememberImageEntry(
  entries: Map<string, ChatImageAccessEntry>,
  image: ChatImageFetchRequest,
  guildId: string | undefined,
  expiresAt: number,
) {
  const key = accessKey(image)
  const existing = entries.get(key)
  const guildIds = new Set(existing?.guildIds ?? [])
  guildIds.add(guildId)
  // 重新插入以刷新插入顺序，使容量淘汰近似 LRU
  entries.delete(key)
  entries.set(key, { guildIds, expiresAt })
}

function evictImageEntries(
  entries: Map<string, ChatImageAccessEntry>,
  capacity: number,
  at: number,
) {
  if (entries.size <= capacity) return
  for (const [key, entry] of entries) {
    if (entry.expiresAt <= at) {
      entries.delete(key)
    }
  }
  while (entries.size > capacity) {
    const oldest = entries.keys().next()
    if (oldest.done) break
    entries.delete(oldest.value)
  }
}

function parseChatImageFetchParams(params: ChatImageFetchParams): ChatImageFetchRequest {
  const file = requireImageFile(params?.file)
  const url = requireImageUrl(params?.url)
  return { file, url }
}

function requireImageFile(value: unknown): string {
  if (typeof value !== 'string' || !value.trim()) {
    throw new Error('image file is required')
  }

  const file = value.trim()
  if (/[\\/]/.test(file) || file.includes('\0')) {
    throw new Error('image file must be a OneBot message-segment file identifier')
  }
  return file
}

function requireImageUrl(value: unknown): string {
  if (typeof value !== 'string' || !value.trim()) {
    throw new Error('image url is required')
  }

  const url = new URL(value.trim())
  if (!isProxyableImageUrl(url)) {
    throw new Error(`unsupported image proxy host: ${url.hostname}`)
  }
  return url.toString()
}

function assertImageScope(scope: ConsoleGuildScope, guildIds: ReadonlySet<string | undefined>) {
  if (scope.kind === 'all') return
  for (const guildId of guildIds) {
    if (guildId && scope.guildIds.has(guildId)) return
  }

  const blockedGuildId = [...guildIds].find((guildId): guildId is string => Boolean(guildId))
  assertConsoleGuildAccess(scope, blockedGuildId, 'chat image')
}

async function resolveOneBotImageResult(
  result: unknown,
  request: ChatImageFetchRequest,
): Promise<ChatImageFetchResult | null> {
  if (!isRecord(result)) return null
  if (typeof result.base64 === 'string' && result.base64) {
    return fromBase64({
      base64: result.base64,
      fileName: readResultFileName(result),
      sourceUrl: request.url,
      source: 'onebot-base64',
    })
  }
  if (typeof result.url === 'string' && result.url) {
    return fetchRemoteImage(result.url, request)
  }
  if (typeof result.file === 'string' && result.file) {
    return readLocalOneBotImage(result.file, request.url)
  }
  return null
}

async function fetchRemoteImage(url: string, request: ChatImageFetchRequest): Promise<ChatImageFetchResult> {
  const parsed = new URL(url)
  if (!isProxyableImageUrl(parsed)) {
    throw new Error(`unsupported OneBot image url host: ${parsed.hostname}`)
  }

  const response = await fetch(parsed)
  if (!response.ok) {
    throw new Error(`OneBot image url returned HTTP ${response.status}`)
  }

  const buffer = Buffer.from(await response.arrayBuffer())
  const mimeType = resolveResponseMimeType(response.headers.get('content-type'), request.file)
  return buildDataUrlResult({
    buffer,
    mimeType,
    sourceUrl: request.url,
    source: 'onebot-url',
  })
}

async function readLocalOneBotImage(filePath: string, sourceUrl: string): Promise<ChatImageFetchResult> {
  const buffer = await readFile(filePath)
  const mimeType = mimeTypeFromName(filePath)
  return buildDataUrlResult({ buffer, mimeType, sourceUrl, source: 'onebot-file' })
}

function fromBase64(input: {
  readonly base64: string
  readonly fileName: string
  readonly sourceUrl: string
  readonly source: string
}): ChatImageFetchResult {
  const { base64, fileName, sourceUrl, source } = input
  const mimeType = mimeTypeFromName(fileName)
  return {
    dataUrl: `${IMAGE_DATA_URL_PREFIX}${mimeType};${BASE64_ENCODING},${base64}`,
    hash: hashSourceUrl(sourceUrl),
    mimeType,
    source,
  }
}

function buildDataUrlResult(input: {
  readonly buffer: Buffer
  readonly mimeType: string
  readonly sourceUrl: string
  readonly source: string
}): ChatImageFetchResult {
  const { buffer, mimeType, sourceUrl, source } = input
  return {
    dataUrl: `${IMAGE_DATA_URL_PREFIX}${mimeType};${BASE64_ENCODING},${buffer.toString(BASE64_ENCODING)}`,
    hash: hashSourceUrl(sourceUrl),
    mimeType,
    source,
  }
}

function readImageElements(elements: unknown): ChatImageFetchRequest[] {
  if (!Array.isArray(elements)) return []
  return elements.flatMap((element) => readImageElement(element))
}

function readImageElement(element: unknown): ChatImageFetchRequest[] {
  if (!isRecord(element)) return []

  const nested = readImageElements(element.children)
  if (element.type !== 'img' && element.type !== 'image') return nested

  try {
    return [parseChatImageFetchParams({
      file: readAttr(element.attrs, 'file'),
      url: readAttr(element.attrs, 'src') || readAttr(element.attrs, 'url'),
    }), ...nested]
  } catch {
    return nested
  }
}

function readAttr(attrs: unknown, key: string): string {
  if (!isRecord(attrs)) return ''
  const value = attrs[key]
  return typeof value === 'string' ? value : ''
}

function readResultFileName(result: Record<string, unknown>): string {
  if (typeof result.file_name === 'string') return result.file_name
  if (typeof result.file === 'string') return result.file
  return ''
}

function resolveResponseMimeType(contentType: string | null, fileName: string): string {
  const mimeType = normalizeImageMimeType(contentType)
  if (mimeType) return mimeType
  if (contentType) throw new Error(`OneBot image url returned non-image content-type: ${contentType}`)
  return mimeTypeFromName(fileName)
}

function normalizeImageMimeType(value: string | null): string | null {
  if (!value) return null
  const mimeType = value.split(';')[0]?.trim().toLowerCase()
  return mimeType?.startsWith('image/') ? mimeType : null
}

function mimeTypeFromName(fileName: string): string {
  const lowerName = fileName.toLowerCase()
  for (const [extension, mimeType] of IMAGE_MIME_BY_EXTENSION) {
    if (lowerName.endsWith(extension)) return mimeType
  }
  return DEFAULT_IMAGE_MIME_TYPE
}

function isProxyableImageUrl(url: URL): boolean {
  if (url.protocol !== 'https:' && url.protocol !== 'http:') return false
  return IMAGE_PROXY_HOST_SUFFIXES.some((suffix) => url.hostname === suffix || url.hostname.endsWith(`.${suffix}`))
}

function accessKey(request: ChatImageFetchRequest): string {
  return `${request.file}\0${request.url}`
}

function hashSourceUrl(url: string): string {
  return createHash(HASH_ALGORITHM).update(url).digest('hex')
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function toErrorMessage(value: unknown): string {
  if (value instanceof Error) return value.message
  return String(value)
}
