import type {
  CreateResourceRequest,
  ResourceItem,
  UpdateResourceRequest,
} from '@stuhelper/shared/api'

export type ResourceVisibility = CreateResourceRequest['visibility']

export interface ResourceFormState {
  title: string
  description: string
  category: string
  visibility: ResourceVisibility
  tagsText: string
  bindingsText: string
}

export function createEmptyResourceForm(): ResourceFormState {
  return {
    title: '',
    description: '',
    category: '',
    visibility: 'public',
    tagsText: '',
    bindingsText: '',
  }
}

export function resourceToForm(resource: ResourceItem): ResourceFormState {
  return {
    title: resource.title,
    description: resource.description ?? '',
    category: resource.category ?? '',
    visibility: resource.visibility,
    tagsText: resource.tags.join(', '),
    bindingsText: resource.bindings
      .map(binding => `${binding.type}: ${binding.value}`)
      .join('\n'),
  }
}

export function buildResourceMetadataPayload(
  form: ResourceFormState,
): UpdateResourceRequest {
  return {
    title: form.title.trim(),
    description: optionalTrimmed(form.description),
    category: optionalTrimmed(form.category),
    visibility: form.visibility,
    tags: splitListText(form.tagsText),
    bindings: parseResourceBindings(form.bindingsText),
  }
}

export async function buildCreateResourcePayload(
  form: ResourceFormState,
  file: File,
): Promise<CreateResourceRequest> {
  return {
    ...buildResourceMetadataPayload(form),
    filename: file.name,
    contentType: file.type,
    dataBase64: await readFileAsDataURL(file),
  }
}

function optionalTrimmed(value: string): string | undefined {
  const trimmed = value.trim()
  return trimmed || undefined
}

function splitListText(value: string): string[] {
  const seen = new Set<string>()
  const items: string[] = []
  for (const item of value.split(/[,\n，]+/)) {
    const trimmed = item.trim()
    if (!trimmed || seen.has(trimmed)) continue
    seen.add(trimmed)
    items.push(trimmed)
  }
  return items
}

function parseResourceBindings(value: string): UpdateResourceRequest['bindings'] {
  const bindings: UpdateResourceRequest['bindings'] = []
  const seen = new Set<string>()
  for (const line of value.split(/\r?\n/)) {
    const trimmed = line.trim()
    if (!trimmed) continue
    const separator = trimmed.includes(':') ? ':' : '='
    const index = trimmed.indexOf(separator)
    if (index <= 0 || index >= trimmed.length - 1) {
      throw new Error('invalid resource binding')
    }
    const type = trimmed.slice(0, index).trim()
    const bindingValue = trimmed.slice(index + 1).trim()
    if (!type || !bindingValue) {
      throw new Error('invalid resource binding')
    }
    const key = `${type}\u0000${bindingValue}`
    if (seen.has(key)) continue
    seen.add(key)
    bindings.push({ type, value: bindingValue })
  }
  return bindings
}

function readFileAsDataURL(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.addEventListener('load', () => {
      if (typeof reader.result !== 'string') {
        reject(new Error('invalid file reader result'))
        return
      }
      resolve(reader.result)
    })
    reader.addEventListener('error', () => {
      reject(reader.error ?? new Error('failed to read file'))
    })
    reader.readAsDataURL(file)
  })
}
