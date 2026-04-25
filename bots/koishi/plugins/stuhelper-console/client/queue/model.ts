export interface GetNextFocusableIdOptions {
  ids: readonly string[]
  currentId: string
  removedId?: string
}

export function getNextFocusableId(options: GetNextFocusableIdOptions): string {
  const ids = options.removedId
    ? options.ids.filter((id) => id !== options.removedId)
    : [...options.ids]

  if (ids.length === 0) return ''

  if (ids.includes(options.currentId)) return options.currentId

  const currentIndex = options.ids.indexOf(options.currentId)
  if (currentIndex === -1) return ids[0]

  const fallbackIndex = Math.min(currentIndex, ids.length - 1)
  return ids[fallbackIndex] ?? ''
}
