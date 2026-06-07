import type { components } from '@stuhelper/shared/types'

export type ResourceBinding = components['schemas']['ResourceBinding']
export type ResourceVersion = components['schemas']['ResourceVersion']
export type ResourceItem = components['schemas']['ResourceItem']
export type ResourceDownloadURL = components['schemas']['ResourceDownloadURL']

export interface ResourcePagePayload {
  items: ResourceItem[]
  total: number
  page: number
  pageSize: number
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value)
}

function readString(record: Record<string, unknown>, key: string, message: string): string {
  const value = record[key]
  if (typeof value !== 'string') {
    throw new Error(message)
  }
  return value
}

function readOptionalString(
  record: Record<string, unknown>,
  key: string,
  message: string,
): string | undefined {
  const value = record[key]
  if (value === undefined) {
    return undefined
  }
  if (typeof value !== 'string') {
    throw new Error(message)
  }
  return value
}

function readInteger(record: Record<string, unknown>, key: string, message: string): number {
  const value = record[key]
  if (typeof value !== 'number' || !Number.isInteger(value)) {
    throw new Error(message)
  }
  return value
}

function readStringArray(record: Record<string, unknown>, key: string, message: string): string[] {
  const value = record[key]
  if (!Array.isArray(value) || value.some(item => typeof item !== 'string')) {
    throw new Error(message)
  }
  return value
}

function readBinding(payload: unknown, message: string): ResourceBinding {
  if (!isRecord(payload)) {
    throw new Error(message)
  }
  return {
    type: readString(payload, 'type', message),
    value: readString(payload, 'value', message),
  }
}

function readVersion(payload: unknown, message: string): ResourceVersion {
  if (!isRecord(payload)) {
    throw new Error(message)
  }
  const version = {
    id: readInteger(payload, 'id', message),
    versionNo: readInteger(payload, 'versionNo', message),
    mountID: readInteger(payload, 'mountID', message),
    objectKey: readString(payload, 'objectKey', message),
    filename: readString(payload, 'filename', message),
    contentType: readString(payload, 'contentType', message),
    sizeBytes: readInteger(payload, 'sizeBytes', message),
    createdAt: readString(payload, 'createdAt', message),
  }
  if (version.id <= 0 || version.versionNo <= 0 || version.mountID <= 0 || version.sizeBytes < 0) {
    throw new Error(message)
  }
  return version
}

export function readResourceItemPayload(
  payload: unknown,
  message = 'Invalid resource response',
): ResourceItem {
  if (!isRecord(payload)) {
    throw new Error(message)
  }
  const id = readInteger(payload, 'id', message)
  if (id <= 0) {
    throw new Error(message)
  }

  const visibility = readString(payload, 'visibility', message)
  if (visibility !== 'public' && visibility !== 'private') {
    throw new Error(message)
  }

  const bindings = payload.bindings
  if (!Array.isArray(bindings)) {
    throw new Error(message)
  }

  return {
    id,
    ownerUserID: readOptionalString(payload, 'ownerUserID', message),
    title: readString(payload, 'title', message),
    description: readOptionalString(payload, 'description', message),
    category: readOptionalString(payload, 'category', message),
    visibility,
    tags: readStringArray(payload, 'tags', message),
    bindings: bindings.map(item => readBinding(item, message)),
    latestVersion: readVersion(payload.latestVersion, message),
    createdAt: readString(payload, 'createdAt', message),
    updatedAt: readString(payload, 'updatedAt', message),
  }
}

export function readResourcePagePayload(
  payload: unknown,
  message = 'Invalid resource list response',
): ResourcePagePayload {
  if (!isRecord(payload) || !Array.isArray(payload.items)) {
    throw new Error(message)
  }

  const total = readInteger(payload, 'total', message)
  const page = readInteger(payload, 'page', message)
  const pageSize = readInteger(payload, 'pageSize', message)
  if (total < 0 || page <= 0 || pageSize <= 0) {
    throw new Error(message)
  }

  return {
    items: payload.items.map(item => readResourceItemPayload(item, message)),
    total,
    page,
    pageSize,
  }
}

export function readResourceDownloadURLPayload(
  payload: unknown,
  message = 'Invalid resource download response',
): ResourceDownloadURL {
  if (!isRecord(payload)) {
    throw new Error(message)
  }
  return { url: readString(payload, 'url', message) }
}
